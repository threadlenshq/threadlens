package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
)

type providerKind int

const (
	providerKindAPI providerKind = iota
	providerKindCLI
)

type providerDef struct {
	id     string
	label  string
	kind   providerKind
	envKey string
	model  string
}

var providerRegistry = []providerDef{
	{id: "sdk", label: "Anthropic API (Claude)", kind: providerKindAPI, envKey: "ANTHROPIC_API_KEY", model: "claude-haiku-4-5-20251001"},
	{id: "gemini", label: "Google Gemini", kind: providerKindAPI, envKey: "GEMINI_API_KEY", model: "gemini-2.5-flash"},
	{id: "copilot", label: "GitHub Copilot", kind: providerKindCLI, envKey: "", model: "gpt-5-mini"},
	{id: "claude-cli", label: "Claude CLI", kind: providerKindCLI, envKey: "", model: "haiku"},
	{id: "opencode", label: "OpenCode CLI", kind: providerKindCLI, envKey: "", model: "deepseek-v4-flash"},
}

var providerRegistryByID = func() map[string]providerDef {
	m := make(map[string]providerDef, len(providerRegistry))
	for _, d := range providerRegistry {
		m[d.id] = d
	}
	m["anthropic"] = m["sdk"]
	return m
}()

func providerIDs() string {
	ids := make([]string, 0, len(providerRegistry))
	for _, d := range providerRegistry {
		ids = append(ids, d.id)
	}
	return strings.Join(ids, ", ")
}

var testAIEnvMu sync.Mutex

type testAIRequest struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
}

type testAIResponse struct {
	Connected bool   `json:"connected"`
	Provider  string `json:"provider"`
	Error     string `json:"error,omitempty"`
}

func TestAI(ctx context.Context, provider string, key string) (bool, string) {
	def, ok := providerRegistryByID[provider]
	if !ok {
		return false, fmt.Sprintf("unknown provider %q", provider)
	}
	switch def.kind {
	case providerKindAPI:
		return testAPIProvider(ctx, def, provider, key)
	case providerKindCLI:
		return testCLIProvider(ctx, def, provider)
	default:
		return false, fmt.Sprintf("unsupported provider kind for %q", provider)
	}
}

func testAPIProvider(ctx context.Context, def providerDef, provider string, key string) (bool, string) {
	resolvedKey := strings.TrimSpace(key)
	if resolvedKey == "" {
		resolvedKey = os.Getenv(def.envKey)
	}
	if resolvedKey == "" {
		return false, "no API key provided or configured"
	}
	for _, r := range resolvedKey {
		if r < 0x20 || r == 0x7F {
			return false, "invalid API key format"
		}
	}
	testAIEnvMu.Lock()
	prev := os.Getenv(def.envKey)
	os.Setenv(def.envKey, resolvedKey)
	defer func() {
		os.Setenv(def.envKey, prev)
		testAIEnvMu.Unlock()
	}()
	var p ai.Provider
	switch provider {
	case "sdk", "anthropic":
		p = &ai.AnthropicProvider{}
	case "gemini":
		p = &ai.GeminiProvider{}
	}
	if !p.Available() {
		return false, "provider not available after setting key"
	}
	text, err := p.Generate(ctx, def.model, "", "Reply with exactly the text: OK", 30*time.Second)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return false, "connection timed out — check your network and key"
		}
		errMsg := strings.ReplaceAll(err.Error(), resolvedKey, "[REDACTED]")
		return false, errMsg
	}
	_ = text
	return true, ""
}

func testCLIProvider(ctx context.Context, def providerDef, provider string) (bool, string) {
	var p ai.Provider
	switch provider {
	case "copilot":
		p = ai.NewCLIProvider("copilot")
	case "claude-cli":
		p = ai.NewCLIProvider("claude")
	case "opencode":
		p = ai.NewOpencodeProvider()
	default:
		return false, fmt.Sprintf("unsupported CLI provider %q", provider)
	}
	if !p.Available() {
		return false, fmt.Sprintf("%s CLI not found in PATH — install it first", provider)
	}
	text, err := p.Generate(ctx, def.model, "", "Reply with exactly the text: OK", 60*time.Second)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return false, "connection timed out — check your network and CLI authentication"
		}
		return false, err.Error()
	}
	_ = text
	return true, ""
}

func handleTestAI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req testAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		provider := strings.TrimSpace(req.Provider)
		if provider == "" {
			httpx.WriteError(w, http.StatusBadRequest, "provider is required")
			return
		}
		if _, ok := providerRegistryByID[provider]; !ok {
			httpx.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unknown provider %q — must be one of: %s", provider, providerIDs()))
			return
		}
		connected, errMsg := TestAI(r.Context(), provider, req.Key)
		resp := testAIResponse{
			Connected: connected,
			Provider:  provider,
		}
		if !connected {
			resp.Error = errMsg
		}
		httpx.WriteJSON(w, http.StatusOK, resp)
	}
}
