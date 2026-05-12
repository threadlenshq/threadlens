// Package ai provides AI provider management and generation services.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/usage"
)

// SettingsGetter is a minimal interface for reading app_settings rows.
// repository.Repository satisfies this interface without an import cycle.
type SettingsGetter interface {
	GetSetting(ctx context.Context, key string) (string, bool, error)
}

// fallbackOrder is the provider:model IDs tried in sequence by GenerateAuto and GenerateForTask.
var fallbackOrder = []string{
	"copilot:gpt-5-mini",
	"claude-cli:haiku",
	"sdk:haiku",
	"gemini:2.5-flash",
}

// defaultProviders returns the direct providers that can run inside the API process environment.
// The host bridge is attached separately as an optional transport adapter.
func defaultProviders() []Provider {
	return []Provider{
		NewCLIProvider("copilot"),
		NewCLIProvider("claude"),
		&AnthropicProvider{},
		&GeminiProvider{},
	}
}

// Service is the AI generation service.  It wraps an ordered list of Provider
// implementations and mirrors the generateAuto / generateForTask / generateWithOpus
// behaviour from the Express ai.js module.
type Service struct {
	repo      SettingsGetter
	providers []Provider // ordered: copilot, claude-cli, sdk, gemini
	bridge    *BridgeProvider
	meter     usage.Meter

	mu             sync.Mutex
	cachedProvider Provider // last known-good provider for GenerateAuto
}

// NewService creates a new Service backed by the given settings getter and the
// default production providers.  Pass nil if no per-task model overrides are needed.
func NewService(repo SettingsGetter) *Service {
	return &Service{
		repo:      repo,
		providers: defaultProviders(),
		bridge:    NewBridgeProvider(),
		meter:     usage.NoopMeter{},
	}
}

// NewServiceWithUsage creates a Service with the given settings getter and usage meter.
func NewServiceWithUsage(repo SettingsGetter, meter usage.Meter) *Service {
	return &Service{
		repo:      repo,
		providers: defaultProviders(),
		bridge:    NewBridgeProvider(),
		meter:     meter,
	}
}

// NewServiceWithProviders creates a Service with the supplied providers list.
// Intended for testing only.
func NewServiceWithProviders(providers []Provider) *Service {
	directProviders, bridge := splitDirectProviders(providers)
	return &Service{providers: directProviders, bridge: bridge, meter: usage.NoopMeter{}}
}

func splitDirectProviders(providers []Provider) ([]Provider, *BridgeProvider) {
	directProviders := make([]Provider, 0, len(providers))
	var bridge *BridgeProvider
	for _, provider := range providers {
		if bp, ok := provider.(*BridgeProvider); ok {
			bridge = bp
			continue
		}
		directProviders = append(directProviders, provider)
	}
	return directProviders, bridge
}

// bridgeProvider returns the BridgeProvider from the bridge field, if present.
func (s *Service) bridgeProvider() *BridgeProvider {
	return s.bridge
}

// providerFor returns the Provider whose Name() matches the given provider tag
// (e.g. "copilot", "claude-cli", "sdk", "gemini").
func (s *Service) providerFor(providerTag string) Provider {
	// Map catalog provider tag -> binary name used by CLIProvider.Name()
	binaryName := providerTag
	if providerTag == "claude-cli" {
		binaryName = "claude"
	}
	for _, p := range s.providers {
		if p.Name() == binaryName {
			return p
		}
	}
	return nil
}

// invokeModelWithBridge attempts bridge → direct provider for bridge-compatible models,
// or falls through to the direct provider only for non-bridge-compatible models.
// The catalog provider tag (e.g. "copilot", "claude-cli") is forwarded to the bridge
// so it can route to the correct host runtime.
// The returned model ID is always the catalog model ID (m.ID), regardless of which
// underlying transport succeeded.
func (s *Service) invokeModelWithBridge(ctx context.Context, m *ModelEntry, systemPrompt, userMessage string, timeout time.Duration) (string, error) {
	if isBridgeCompatible(m.Provider) {
		if bp := s.bridgeProvider(); bp != nil {
			result, err := bp.GenerateWithProvider(ctx, m.Provider, m.Model, systemPrompt, userMessage, timeout)
			if err == nil {
				return result, nil
			}
			// Bridge failed — fall through to direct provider.
		}
	}
	return s.invokeModel(ctx, m, systemPrompt, userMessage, timeout)
}

// invokeModel calls the right provider for a ModelEntry with the given timeout.
func (s *Service) invokeModel(ctx context.Context, m *ModelEntry, systemPrompt, userMessage string, timeout time.Duration) (string, error) {
	p := s.providerFor(m.Provider)
	if p == nil {
		return "", fmt.Errorf("no provider registered for %q", m.Provider)
	}
	return p.Generate(ctx, m.Model, systemPrompt, userMessage, timeout)
}

// GenerateForTask resolves the model for taskID (honouring user overrides), then
// tries it followed by the fallback chain.  Mirrors Express generateForTask().
// Returns (result, usedModelID, error).
func (s *Service) GenerateForTask(ctx context.Context, taskID string, systemPrompt string, userMessage string) (string, string, error) {
	// Resolve the intended model ID up front so it is available for usage
	// metering even if generation fails.
	resolvedModelID, _ := s.resolveTaskModel(ctx, taskID)

	result, usedModelID, err := s.generateForTaskInner(ctx, taskID, systemPrompt, userMessage)

	// Use the actually-used model when available; fall back to resolved primary.
	meterModelID := usedModelID
	if meterModelID == "" {
		meterModelID = resolvedModelID
	}

	event := usage.Event{
		TaskID:     taskID,
		ModelID:    meterModelID,
		Operation:  "ai.generate_for_task",
		Success:    err == nil,
		RecordedAt: time.Now().UTC(),
	}
	if err != nil {
		event.Error = err.Error()
	}
	if s.meter != nil {
		_ = s.meter.Record(ctx, event)
	}

	return result, usedModelID, err
}

// generateForTaskInner is the core implementation of GenerateForTask without metering.
func (s *Service) generateForTaskInner(ctx context.Context, taskID string, systemPrompt string, userMessage string) (string, string, error) {
	task := GetTask(taskID)
	if task == nil {
		return "", "", fmt.Errorf("unknown task id: %s", taskID)
	}

	// Resolve user override or default model.
	modelID, err := s.resolveTaskModel(ctx, taskID)
	if err != nil {
		return "", "", err
	}
	primaryModel := GetModel(modelID)
	if primaryModel == nil {
		return "", "", fmt.Errorf("unknown model id: %s", modelID)
	}

	timeout := time.Duration(task.TimeoutMs) * time.Millisecond

	// Build candidate list: primary first, then fallbacks (skip primary duplicate).
	candidates := []*ModelEntry{primaryModel}
	for _, fbID := range fallbackOrder {
		if fbID == modelID {
			continue
		}
		if m := GetModel(fbID); m != nil {
			candidates = append(candidates, m)
		}
	}

	var firstErr error
	for i, candidate := range candidates {
		// First candidate is always tried without an availability check.
		// For subsequent candidates:
		//  - Bridge-compatible providers are gated by the bridge being available OR the direct
		//    provider being available (either path can succeed).
		//  - Non-bridge-compatible providers require their direct provider to be available.
		if i > 0 {
			bridgeCouldHandle := isBridgeCompatible(candidate.Provider) && s.bridgeProvider() != nil && s.bridgeProvider().CanServe(candidate.Provider)
			directOK := func() bool {
				p := s.providerFor(candidate.Provider)
				return p != nil && p.Available()
			}()
			if !bridgeCouldHandle && !directOK {
				continue
			}
		}

		result, err := s.invokeModelWithBridge(ctx, candidate, systemPrompt, userMessage, timeout)
		if err == nil {
			return result, candidate.ID, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}

	reason := "no available providers — ensure host CLI bridge is reachable, Copilot CLI or Claude CLI is installed, or set ANTHROPIC_API_KEY / GEMINI_API_KEY"
	if firstErr != nil {
		reason = firstErr.Error()
	}
	return "", "", fmt.Errorf("all AI providers failed for task %q (tried bridge, CLI, API-key providers): %s", taskID, reason)
}

// GenerateAuto tries the cached provider first, then walks the fallback chain.
// Mirrors Express generateAuto().
func (s *Service) GenerateAuto(ctx context.Context, systemPrompt string, userMessage string) (string, error) {
	const defaultTimeout = 60 * time.Second

	// Try cached provider.
	s.mu.Lock()
	cached := s.cachedProvider
	s.mu.Unlock()

	if cached != nil {
		modelID := autoModelForProvider(cached)
		if m := GetModel(modelID); m != nil {
			result, err := cached.Generate(ctx, m.Model, systemPrompt, userMessage, defaultTimeout)
			if err == nil {
				return result, nil
			}
			// Cached provider failed – clear it.
			s.mu.Lock()
			s.cachedProvider = nil
			s.mu.Unlock()
		}
	}

	// Walk fallback order.
	for _, fbID := range fallbackOrder {
		m := GetModel(fbID)
		if m == nil {
			continue
		}
		p := s.providerFor(m.Provider)
		if p == nil || !p.Available() {
			continue
		}
		result, err := p.Generate(ctx, m.Model, systemPrompt, userMessage, defaultTimeout)
		if err == nil {
			s.mu.Lock()
			s.cachedProvider = p
			s.mu.Unlock()
			return result, nil
		}
	}

	return "", fmt.Errorf("all AI providers failed. Ensure Copilot CLI or Claude CLI is installed, or set ANTHROPIC_API_KEY or GEMINI_API_KEY")
}

// autoModelForProvider maps a provider to its canonical fallback model ID.
func autoModelForProvider(p Provider) string {
	switch p.Name() {
	case "copilot":
		return "copilot:gpt-5-mini"
	case "claude":
		return "claude-cli:haiku"
	case "sdk":
		return "sdk:haiku"
	case "gemini":
		return "gemini:2.5-flash"
	}
	return ""
}

// GenerateWithOpus calls the claude CLI with the opus model and a 5-minute timeout.
// Mirrors Express generateWithOpus().  The 10 MiB output allowance is a best-effort
// constraint; Go's exec.Command buffers stdout in memory with no hard limit, so
// large outputs are acceptable.
func (s *Service) GenerateWithOpus(ctx context.Context, systemPrompt string, userMessage string) (string, error) {
	p := s.providerFor("claude-cli")
	if p == nil {
		return "", fmt.Errorf("claude CLI provider not registered")
	}
	return p.Generate(ctx, "opus", systemPrompt, userMessage, 5*time.Minute)
}

// StripMarkdown removes common markdown decoration from text, mirroring
// Express stripMarkdown():
//
//	**bold** → bold, __bold__ → bold, *italic* → italic, _italic_ → italic,
//	`code` → code, em dash (U+2014) → ", "
func StripMarkdown(text string) string {
	text = reBold.ReplaceAllString(text, "$1")
	text = reBoldUnder.ReplaceAllString(text, "$1")
	text = reItalic.ReplaceAllString(text, "$1")
	text = reItalicUnder.ReplaceAllString(text, "$1")
	text = reCode.ReplaceAllString(text, "$1")
	text = strings.ReplaceAll(text, "\u2014", ", ")
	return text
}

var (
	reBold        = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reBoldUnder   = regexp.MustCompile(`__(.+?)__`)
	reItalic      = regexp.MustCompile(`\*(.+?)\*`)
	reItalicUnder = regexp.MustCompile(`_(.+?)_`)
	reCode        = regexp.MustCompile("`(.+?)`")
)

// resolveTaskModel reads the user-configured model for taskID from the settings store,
// falling back to the task's default.  It is defined here (not in services/) to
// avoid an import cycle: services imports ai; ai must not import services.
func (s *Service) resolveTaskModel(ctx context.Context, taskID string) (string, error) {
	task := GetTask(taskID)
	if task == nil {
		return "", fmt.Errorf("unknown task id: %s", taskID)
	}

	if s.repo != nil {
		raw, ok, err := s.repo.GetSetting(ctx, "model."+taskID)
		if err != nil {
			return "", err
		}
		if ok && raw != "" {
			var obj map[string]string
			if jsonErr := json.Unmarshal([]byte(raw), &obj); jsonErr == nil {
				if modelID, exists := obj["modelId"]; exists && GetModel(modelID) != nil {
					return modelID, nil
				}
			}
		}
	}

	return task.Default, nil
}

// StripMarkdownMethod is a method wrapper so that *Service satisfies the
// services.AIService interface (which requires StripMarkdown as a method).
func (s *Service) StripMarkdown(text string) string {
	return StripMarkdown(text)
}
