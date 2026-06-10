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

// providerSecretKeys maps provider names to their environment variable keys.
var providerSecretKeys = map[string]string{
	"anthropic": "ANTHROPIC_API_KEY",
	"gemini":    "GEMINI_API_KEY",
}

// providerModels maps provider names to the cheapest model used for connectivity tests.
var providerModels = map[string]string{
	"anthropic": "claude-haiku-4-5-20251001",
	"gemini":    "gemini-2.5-flash",
}

// testAIEnvMu serializes env-var mutations in TestAI because os.Setenv is not
// safe for concurrent use.
var testAIEnvMu sync.Mutex

// testAIRequest is the JSON body for POST /api/onboarding/test-ai.
type testAIRequest struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
}

// testAIResponse is the JSON response for POST /api/onboarding/test-ai.
type testAIResponse struct {
	Connected bool   `json:"connected"`
	Provider  string `json:"provider"`
	Error     string `json:"error,omitempty"`
}

// TestAI attempts a minimal live call to the specified AI provider. It returns
// whether the connection succeeded and an error message on failure. The
// submitted key is used only for the duration of this call and is never
// persisted or logged.
func TestAI(ctx context.Context, provider string, key string) (bool, string) {
	envKey, ok := providerSecretKeys[provider]
	if !ok {
		return false, fmt.Sprintf("unknown provider %q", provider)
	}

	// Resolve the key: prefer the submitted key, fall back to the env value.
	resolvedKey := strings.TrimSpace(key)
	if resolvedKey == "" {
		resolvedKey = os.Getenv(envKey)
	}
	if resolvedKey == "" {
		return false, "no API key provided or configured"
	}

	// Reject keys with control characters (defensive).
	for _, r := range resolvedKey {
		if r < 0x20 || r == 0x7F {
			return false, "invalid API key format"
		}
	}

	// Temporarily set the env var so the provider's Available() and Generate()
	// pick it up, then restore the previous value. We lock because os.Setenv
	// is not safe for concurrent use.
	testAIEnvMu.Lock()
	prev := os.Getenv(envKey)
	os.Setenv(envKey, resolvedKey)
	defer func() {
		os.Setenv(envKey, prev)
		testAIEnvMu.Unlock()
	}()

	// Build the provider instance.
	var p ai.Provider
	switch provider {
	case "anthropic":
		p = &ai.AnthropicProvider{}
	case "gemini":
		p = &ai.GeminiProvider{}
	default:
		return false, fmt.Sprintf("unsupported provider %q", provider)
	}

	if !p.Available() {
		return false, "provider not available after setting key"
	}

	model := providerModels[provider]
	timeout := 10 * time.Second

	// Anthropic uses max_tokens in the request body; we pass a small value via
	// the Generate call. The Generate method uses a hardcoded max_tokens of
	// 4096, but for a connectivity test the response is tiny regardless.
	text, err := p.Generate(ctx, model, "", "Reply with exactly the text: OK", timeout)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return false, "connection timed out — check your network and key"
		}
		// Strip any key-like content from the error message before returning.
		errMsg := strings.ReplaceAll(err.Error(), resolvedKey, "[REDACTED]")
		return false, errMsg
	}

	// Success: the response text should contain "OK" (case-insensitive).
	if strings.Contains(strings.ToUpper(strings.TrimSpace(text)), "OK") {
		return true, ""
	}
	// Even if the text doesn't contain OK, a 200 response means the key works.
	return true, ""
}

// handleTestAI is the HTTP handler for POST /api/onboarding/test-ai.
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
		if _, ok := providerSecretKeys[provider]; !ok {
			httpx.WriteError(w, http.StatusBadRequest, fmt.Sprintf("unknown provider %q — must be one of: anthropic, gemini", provider))
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
