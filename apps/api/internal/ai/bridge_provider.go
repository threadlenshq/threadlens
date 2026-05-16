package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// isBridgeCompatible reports whether a catalog provider tag routes through the bridge.
// Bridge-compatible providers are "copilot" and "claude-cli".
func isBridgeCompatible(providerTag string) bool {
	return providerTag == "copilot" || providerTag == "claude-cli"
}

// BridgeProvider implements Provider using the localhost HTTP bridge.
// It calls GET /v1/health before each generation to verify the bridge is live,
// then calls POST /v1/generate with a bearer token.
//
// The bridge is not permanently disabled after a failure; each call re-checks health.
type BridgeProvider struct {
	// stateFn returns the current bridge state each time it is called.
	// Using a function allows tests to inject a static state without a real config loader.
	stateFn func() (BridgeState, error)
	// client is the HTTP client used for all bridge calls.
	client *http.Client
}

// NewBridgeProvider creates a production BridgeProvider that discovers config via LoadBridgeConfig.
// The HTTP client has no fixed timeout because request timeouts are controlled by
// the context passed to Generate (derived from the task's TimeoutMs).
func NewBridgeProvider() *BridgeProvider {
	return &BridgeProvider{
		stateFn: func() (BridgeState, error) { return LoadBridgeConfig() },
		client:  &http.Client{},
	}
}

// Name returns "bridge".
func (p *BridgeProvider) Name() string { return "bridge" }

// Available returns true if the bridge config is currently detected and enabled.
// This is a lightweight check — it does not perform a network call.
func (p *BridgeProvider) Available() bool {
	state, err := p.stateFn()
	if err != nil {
		return false
	}
	return state.Enabled && state.Detected && state.URL != "" && state.Token != ""
}

// CanServe returns true when this catalog provider is bridge-compatible and the bridge
// config is currently available. It is a lightweight check that does not perform
// network I/O; runtime availability is verified by the live /v1/health call in
// GenerateWithProvider.
func (p *BridgeProvider) CanServe(provider string) bool {
	return isBridgeCompatible(provider) && p.Available()
}

func runtimeListAllows(runtimes []string, provider string) bool {
	if len(runtimes) == 0 {
		return true
	}
	for _, runtimeID := range runtimes {
		if runtimeID == provider {
			return true
		}
	}
	return false
}

// bridgeHealthResponse is the expected shape of GET /v1/health.
type bridgeHealthResponse struct {
	OK       bool            `json:"ok"`
	Runtimes json.RawMessage `json:"runtimes"`
}

type bridgeHealthRuntime struct {
	ID string `json:"id"`
}

// bridgeGenerateRequest is the body sent to POST /v1/generate.
type bridgeGenerateRequest struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	SystemPrompt string `json:"systemPrompt"`
	UserMessage  string `json:"userMessage"`
	TimeoutMs    int64  `json:"timeoutMs"`
}

// bridgeGenerateResponse is the expected shape of POST /v1/generate.
type bridgeGenerateResponse struct {
	Text string `json:"text"`
}

// Generate calls the bridge's /v1/health then /v1/generate endpoints.
// Returns a trimmed non-empty text response, or an error.
// Failures are transient — the bridge is not marked as permanently unavailable.
// It delegates to GenerateWithProvider with an empty provider string for backward compatibility.
func (p *BridgeProvider) Generate(ctx context.Context, model string, systemPrompt string, userMessage string, timeout time.Duration) (string, error) {
	return p.GenerateWithProvider(ctx, "", model, systemPrompt, userMessage, timeout)
}

// GenerateWithProvider calls the bridge's /v1/health then /v1/generate endpoints,
// sending the explicit provider tag (e.g. "copilot", "claude-cli") in the request body.
// Returns a trimmed non-empty text response, or an error.
// Failures are transient — the bridge is not marked as permanently unavailable.
func (p *BridgeProvider) GenerateWithProvider(ctx context.Context, provider string, model string, systemPrompt string, userMessage string, timeout time.Duration) (string, error) {
	state, err := p.stateFn()
	if err != nil {
		return "", fmt.Errorf("bridge: config error: %w", err)
	}
	if !state.Enabled || !state.Detected || state.URL == "" || state.Token == "" {
		return "", fmt.Errorf("bridge: not available (enabled=%v, detected=%v)", state.Enabled, state.Detected)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Health check.
	if err := p.checkHealth(timeoutCtx, state, provider); err != nil {
		return "", fmt.Errorf("bridge: health check failed: %w", err)
	}

	// Generation.
	return p.callGenerate(timeoutCtx, state, provider, model, systemPrompt, userMessage, timeout)
}

// checkHealth calls GET /v1/health with bearer auth and returns an error if unhealthy.
func (p *BridgeProvider) checkHealth(ctx context.Context, state BridgeState, provider string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, state.URL+"/v1/health", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+state.Token)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bridge health returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var health bridgeHealthResponse
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &health); err != nil {
			return fmt.Errorf("bridge health: invalid JSON response: %w", err)
		}
	}
	if !health.OK {
		return fmt.Errorf("bridge health returned ok=false")
	}
	runtimeIDs, err := parseHealthRuntimeIDs(health.Runtimes)
	if err != nil {
		return err
	}
	if provider != "" && !runtimeListAllows(runtimeIDs, provider) {
		return fmt.Errorf("bridge: runtime unavailable for provider %q", provider)
	}
	return nil
}

func parseHealthRuntimeIDs(raw json.RawMessage) ([]string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var ids []string
	if err := json.Unmarshal(raw, &ids); err == nil {
		return ids, nil
	}

	var runtimes []bridgeHealthRuntime
	if err := json.Unmarshal(raw, &runtimes); err == nil {
		result := make([]string, 0, len(runtimes))
		for _, rt := range runtimes {
			if rt.ID != "" {
				result = append(result, rt.ID)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("bridge health: invalid runtimes format")
}

// callGenerate calls POST /v1/generate and returns the trimmed text.
func (p *BridgeProvider) callGenerate(ctx context.Context, state BridgeState, provider, model, systemPrompt, userMessage string, timeout time.Duration) (string, error) {
	reqBody, err := json.Marshal(bridgeGenerateRequest{
		Provider:     provider,
		Model:        model,
		SystemPrompt: systemPrompt,
		UserMessage:  userMessage,
		TimeoutMs:    timeout.Milliseconds(),
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, state.URL+"/v1/generate", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+state.Token)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bridge generate returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var genResp bridgeGenerateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return "", fmt.Errorf("bridge generate: invalid JSON response: %w", err)
	}

	text := strings.TrimSpace(genResp.Text)
	if text == "" {
		return "", fmt.Errorf("bridge generate: empty response text")
	}
	return text, nil
}
