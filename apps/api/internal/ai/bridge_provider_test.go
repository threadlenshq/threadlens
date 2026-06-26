package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/usage"
)

// ---------------------------------------------------------------------------
// BridgeProvider unit tests
// ---------------------------------------------------------------------------

// newBridgeProviderForTest returns a BridgeProvider wired to a test BridgeState.
func newBridgeProviderForTest(state BridgeState) *BridgeProvider {
	return &BridgeProvider{
		stateFn: func() (BridgeState, error) { return state, nil },
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func TestBridgeProvider_HealthAndGenerate(t *testing.T) {
	// Server verifies Authorization header on both endpoints.
	const token = "test-bearer-token"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"runtimes": []string{"copilot", "claude-cli"},
			})
		case "/v1/generate":
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req["provider"] != "copilot" {
				http.Error(w, "missing provider field", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"text": "  generated response  ",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	state := BridgeState{
		Enabled:  true,
		Detected: true,
		URL:      srv.URL,
		Token:    token,
		Runtimes: []string{"copilot", "claude-cli"},
	}
	p := newBridgeProviderForTest(state)

	if !p.Available() {
		t.Fatal("expected bridge to be available")
	}

	result, err := p.GenerateWithProvider(context.Background(), "copilot", "gpt-5-mini", "system prompt", "user message", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "generated response" {
		t.Errorf("expected trimmed 'generated response', got %q", result)
	}
}

func TestBridgeProvider_Name(t *testing.T) {
	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true})
	if p.Name() != "bridge" {
		t.Errorf("expected name 'bridge', got %q", p.Name())
	}
}

func TestBridgeProvider_AvailableFalseWhenNotDetected(t *testing.T) {
	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: false})
	if p.Available() {
		t.Fatal("expected bridge to be unavailable when not detected")
	}
}

func TestBridgeProvider_AvailableFalseWhenDisabled(t *testing.T) {
	p := newBridgeProviderForTest(BridgeState{Enabled: false, Detected: false})
	if p.Available() {
		t.Fatal("expected bridge to be unavailable when disabled")
	}
}

func TestBridgeProvider_RejectsNon2xxResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case "/v1/generate":
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})
	_, err := p.Generate(context.Background(), "model", "sys", "msg", 5*time.Second)
	if err == nil {
		t.Fatal("expected error on 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got %q", err.Error())
	}
}

func TestBridgeProvider_RejectsInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case "/v1/generate":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("not-json"))
		}
	}))
	defer srv.Close()

	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})
	_, err := p.Generate(context.Background(), "model", "sys", "msg", 5*time.Second)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestBridgeProvider_RejectsEmptyText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case "/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "   "})
		}
	}))
	defer srv.Close()

	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})
	_, err := p.Generate(context.Background(), "model", "sys", "msg", 5*time.Second)
	if err == nil {
		t.Fatal("expected error on empty trimmed text")
	}
}

func TestBridgeProvider_RejectsUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "wrong"})
	_, err := p.Generate(context.Background(), "model", "sys", "msg", 5*time.Second)
	if err == nil {
		t.Fatal("expected error on 401")
	}
}

func TestBridgeProvider_HealthCheckFailDoesNotPermanentlyDisable(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 && r.URL.Path == "/v1/health" {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case "/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "ok-result"})
		}
	}))
	defer srv.Close()

	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})

	// First call: health check fails on first attempt, but the retry salvages it.
	result, err1 := p.Generate(context.Background(), "model", "sys", "msg", 5*time.Second)
	if err1 != nil {
		t.Fatalf("expected retry to salvage first call after health check transient failure, got: %v", err1)
	}
	if result != "ok-result" {
		t.Errorf("expected 'ok-result', got %q", result)
	}

	// Verify the retry was attempted: the server should have received at least
	// the failing health check (call 1) + the retry health check (call 2) + the
	// generate call = 3 calls.
	if callCount < 3 {
		t.Errorf("expected at least 3 server calls (health fail + retry health + generate), got %d", callCount)
	}

	// Second call: health check succeeds immediately → should work.
	result2, err2 := p.Generate(context.Background(), "model", "sys", "msg", 5*time.Second)
	if err2 != nil {
		t.Fatalf("expected second call to succeed, got: %v", err2)
	}
	if result2 != "ok-result" {
		t.Errorf("expected 'ok-result', got %q", result2)
	}
}

// ---------------------------------------------------------------------------
// GenerateForTask bridge-aware routing tests
// ---------------------------------------------------------------------------

// bridgeCapableProvider is a Provider that records calls and returns configured response.
type bridgeCapableProvider struct {
	fakeProvider
}

// TestGenerateForTask_UsesBridgeBeforeDirectCLI verifies that a working bridge
// is tried before the direct CLI provider for a copilot:* model.
func TestGenerateForTask_UsesBridgeBeforeDirectCLI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case "/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "bridge-result"})
		}
	}))
	defer srv.Close()

	bridge := &BridgeProvider{
		stateFn: func() (BridgeState, error) {
			return BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"}, nil
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}

	directCopilot := &fakeProvider{name: "copilot", available: true, result: "direct-result"}
	directClaude := &fakeProvider{name: "claude", available: true, result: "claude-direct"}
	sdk := &fakeProvider{name: "sdk", available: true, result: "sdk-result"}
	gemini := &fakeProvider{name: "gemini", available: true, result: "gemini-result"}

	svc := NewServiceWithProviders([]Provider{bridge, directCopilot, directClaude, sdk, gemini})
	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "bridge-result" {
		t.Errorf("expected 'bridge-result', got %q", result)
	}
	if usedID != "copilot:gpt-5-mini" {
		t.Errorf("expected catalog model ID 'copilot:gpt-5-mini', got %q", usedID)
	}
	// Direct copilot should not have been called
	if directCopilot.calls != 0 {
		t.Errorf("expected direct copilot not called, but got %d calls", directCopilot.calls)
	}
}

// TestGenerateForTask_FallsBackWhenBridgeFails verifies that when the bridge
// fails, the direct CLI provider is tried next.
func TestGenerateForTask_FallsBackWhenBridgeFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bridge error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	bridge := &BridgeProvider{
		stateFn: func() (BridgeState, error) {
			return BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"}, nil
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}

	directCopilot := &fakeProvider{name: "copilot", available: true, result: "direct-result"}
	svc := NewServiceWithProviders([]Provider{bridge, directCopilot})

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("expected fallback to succeed, got: %v", err)
	}
	if result != "direct-result" {
		t.Errorf("expected 'direct-result', got %q", result)
	}
	if usedID != "copilot:gpt-5-mini" {
		t.Errorf("expected usedID='copilot:gpt-5-mini', got %q", usedID)
	}
	if directCopilot.calls != 1 {
		t.Errorf("expected direct copilot called once, got %d calls", directCopilot.calls)
	}
}

// TestGenerateForTask_BridgeCompatibleFallbackChain verifies that when bridge
// and direct CLI both fail, sdk/gemini fallbacks are tried.
func TestGenerateForTask_BridgeCompatibleFallbackChain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bridge error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	bridge := &BridgeProvider{
		stateFn: func() (BridgeState, error) {
			return BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"}, nil
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}

	directCopilot := &fakeProvider{name: "copilot", available: true, err: errors.New("copilot down")}
	directClaude := &fakeProvider{name: "claude", available: true, err: errors.New("claude down")}
	sdk := &fakeProvider{name: "sdk", available: true, result: "sdk-fallback"}
	gemini := &fakeProvider{name: "gemini", available: true, result: "gemini-fallback"}

	svc := NewServiceWithProviders([]Provider{bridge, directCopilot, directClaude, sdk, gemini})
	result, _, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "sdk-fallback" {
		t.Errorf("expected 'sdk-fallback', got %q", result)
	}
}

// TestGenerateForTask_AllProvidersFail_ReturnsFinalError verifies that when all
// providers fail, the error message includes the task ID and the attempted model IDs
// but does not leak raw provider errors or bridge startup advice.
func TestGenerateForTask_AllProvidersFail_ReturnsFinalError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bridge error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	bridge := &BridgeProvider{
		stateFn: func() (BridgeState, error) {
			return BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"}, nil
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}

	directCopilot := &fakeProvider{name: "copilot", available: true, err: errors.New("copilot down")}
	directClaude := &fakeProvider{name: "claude", available: true, err: errors.New("claude down")}
	sdk := &fakeProvider{name: "sdk", available: false}
	gemini := &fakeProvider{name: "gemini", available: false}

	svc := NewServiceWithProviders([]Provider{bridge, directCopilot, directClaude, sdk, gemini})
	_, _, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
	errStr := err.Error()
	// Must include task ID and attempted model IDs.
	for _, required := range []string{"post_scoring", "copilot:gpt-5-mini", "claude-cli:haiku"} {
		if !strings.Contains(errStr, required) {
			t.Errorf("expected error to contain %q, got: %q", required, errStr)
		}
	}
	// Must not leak raw provider errors or bridge startup advice.
	for _, forbidden := range []string{"copilot down", "claude down", "host CLI bridge is reachable", "start bridge"} {
		if strings.Contains(errStr, forbidden) {
			t.Errorf("error contains forbidden text %q: %q", forbidden, errStr)
		}
	}
}

// TestGenerateForTask_BridgeModelIDStaysUnchanged verifies that catalog model IDs
// are preserved even when routed through the bridge.
func TestGenerateForTask_BridgeModelIDStaysUnchanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case "/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "bridge-output"})
		}
	}))
	defer srv.Close()

	bridge := &BridgeProvider{
		stateFn: func() (BridgeState, error) {
			return BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"}, nil
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}

	svc := NewServiceWithProviders([]Provider{bridge})

	// Test copilot:* model
	_, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usedID != "copilot:gpt-5-mini" {
		t.Errorf("expected catalog model ID 'copilot:gpt-5-mini', got %q", usedID)
	}
}

// TestGenerateForTask_ClaudeCLIModelRoutedThroughBridge verifies that claude-cli:*
// models are also routed through bridge when bridge is available.
func TestGenerateForTask_ClaudeCLIModelRoutedThroughBridge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case "/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "bridge-claude-result"})
		}
	}))
	defer srv.Close()

	bridge := &BridgeProvider{
		stateFn: func() (BridgeState, error) {
			return BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"}, nil
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}

	directClaude := &fakeProvider{name: "claude", available: true, result: "direct-claude"}
	svc := NewServiceWithProviders([]Provider{bridge, directClaude})

	// draft_generation default is claude-cli:sonnet
	result, usedID, err := svc.GenerateForTask(context.Background(), "draft_generation", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "bridge-claude-result" {
		t.Errorf("expected 'bridge-claude-result', got %q", result)
	}
	if usedID != "claude-cli:sonnet" {
		t.Errorf("expected catalog model ID 'claude-cli:sonnet', got %q", usedID)
	}
	if directClaude.calls != 0 {
		t.Errorf("expected direct claude not called, got %d calls", directClaude.calls)
	}
}

func TestBridgeProvider_RuntimeMismatchSkipsGenerate(t *testing.T) {
	generateCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "runtimes": []string{"claude-cli"}})
		case "/v1/generate":
			generateCalled = true
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "should-not-run"})
		}
	}))
	defer srv.Close()

	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})
	_, err := p.GenerateWithProvider(context.Background(), "copilot", "gpt-5-mini", "sys", "msg", 5*time.Second)
	if err == nil {
		t.Fatal("expected runtime mismatch error")
	}
	if !strings.Contains(err.Error(), "runtime unavailable") {
		t.Fatalf("error = %q, want runtime unavailable", err.Error())
	}
	if generateCalled {
		t.Fatal("generate endpoint should not be called when health runtimes exclude provider")
	}
}

func TestBridgeProvider_HealthObjectRuntimesAllowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"runtimes": []map[string]any{
					{"id": "copilot", "available": true, "message": "ok"},
					{"id": "claude-cli", "available": true, "message": "ok"},
				},
			})
		case "/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "bridge-object-runtimes-ok"})
		}
	}))
	defer srv.Close()

	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})
	text, err := p.GenerateWithProvider(context.Background(), "copilot", "gpt-5-mini", "sys", "msg", 5*time.Second)
	if err != nil {
		t.Fatalf("expected success for object runtimes health shape, got: %v", err)
	}
	if text != "bridge-object-runtimes-ok" {
		t.Fatalf("result = %q, want bridge-object-runtimes-ok", text)
	}
}

func TestBridgeProvider_StateRuntimeMismatchSkipsHealth(t *testing.T) {
	// BridgeState.Runtimes is advisory only and must NOT prevent the health call.
	// If the live bridge actually supports the provider (health says ok with matching runtime
	// list), generation should succeed even if the local config runtime list is stale/different.
	healthCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			healthCalled = true
			// Live bridge reports it supports copilot — overrides stale config list.
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "runtimes": []string{"copilot", "claude-cli"}})
		case "/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "bridge-ok"})
		}
	}))
	defer srv.Close()

	// Config state has a stale runtime list that only lists claude-cli.
	// The live health response lists copilot, so the call should succeed.
	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok", Runtimes: []string{"claude-cli"}})
	result, err := p.GenerateWithProvider(context.Background(), "copilot", "gpt-5-mini", "sys", "msg", 5*time.Second)
	if err != nil {
		t.Fatalf("expected success when live health confirms runtime, got: %v", err)
	}
	if result != "bridge-ok" {
		t.Fatalf("result = %q, want bridge-ok", result)
	}
	if !healthCalled {
		t.Fatal("health endpoint must be called to get authoritative runtime list")
	}
}

// TestGenerateForTask_NonBridgeModelNotRouted verifies that sdk:* models are NOT
// sent through the bridge, they use their direct provider.
func TestGenerateForTask_NonBridgeModelNotRouted(t *testing.T) {
	bridgeCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bridgeCalled = true
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "runtimes": []string{"copilot", "claude-cli"}})
		case "/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "bridge-result"})
		}
	}))
	defer srv.Close()

	bridge := &BridgeProvider{
		stateFn: func() (BridgeState, error) {
			return BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"}, nil
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}
	sdk := &fakeProvider{name: "sdk", available: true, result: "sdk-direct-result"}

	// Service where sdk:haiku is the configured primary model — bridge must not be called.
	svc := &Service{
		repo: fakeSettingsGetter{values: map[string]string{
			"model.post_scoring": `{"modelId":"sdk:haiku"}`,
		}},
		providers: []Provider{bridge, sdk},
		bridge:    bridge,
		meter:     nil,
	}

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "sdk-direct-result" {
		t.Errorf("expected sdk-direct-result, got %q", result)
	}
	if usedID != "sdk:haiku" {
		t.Errorf("expected usedID=sdk:haiku, got %q", usedID)
	}
	if bridgeCalled {
		t.Error("bridge must not be called for sdk:* models")
	}
}

func TestIsBridgeCompatible_Opencode(t *testing.T) {
	if !isBridgeCompatible("opencode") {
		t.Fatal("expected opencode to be bridge-compatible")
	}
	if !isBridgeCompatible("opencode-go") {
		t.Fatal("expected opencode-go to be bridge-compatible")
	}
	if !isBridgeCompatible("copilot") {
		t.Fatal("expected copilot to be bridge-compatible")
	}
	if !isBridgeCompatible("claude-cli") {
		t.Fatal("expected claude-cli to be bridge-compatible")
	}
	if isBridgeCompatible("sdk") {
		t.Fatal("expected sdk to NOT be bridge-compatible")
	}
	if isBridgeCompatible("gemini") {
		t.Fatal("expected gemini to NOT be bridge-compatible")
	}
}

func TestBridgeProvider_OpencodeGoSendsOpencodeProvider(t *testing.T) {
	// Verify that when the catalog tag is opencode-go, the bridge request body
	// uses provider:"opencode" (the collapsed runtime ID).
	var capturedProvider string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"runtimes": []string{"copilot", "claude-cli", "opencode"},
			})
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if p, ok := body["provider"].(string); ok {
			capturedProvider = p
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"text": "opencode-result"})
	}))
	defer srv.Close()

	realBridge := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})
	opencode := &fakeProvider{name: "opencode", available: true, result: "direct-result"}
	svc := &Service{
		repo: fakeSettingsGetter{values: map[string]string{
			"model.post_scoring": `{"modelId":"opencode-go:deepseek-v4-flash"}`,
		}},
		bridge:    realBridge,
		providers: []Provider{opencode},
		meter:     usage.NoopMeter{},
	}

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "opencode-result" {
		t.Fatalf("result = %q, want opencode-result", result)
	}
	if usedID != "opencode-go:deepseek-v4-flash" {
		t.Fatalf("usedID = %q, want opencode-go:deepseek-v4-flash", usedID)
	}
	if capturedProvider != "opencode" {
		t.Fatalf("bridge received provider = %q, want opencode (collapsed from opencode-go)", capturedProvider)
	}
}

func TestBridgeProvider_HealthAcceptsOpencodeRuntime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"runtimes": []map[string]any{
					{"id": "copilot", "available": true, "message": "ok"},
					{"id": "claude-cli", "available": true, "message": "ok"},
					{"id": "opencode", "available": true, "message": "ok"},
				},
			})
		case "/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]any{"text": "opencode-health-ok"})
		}
	}))
	defer srv.Close()

	p := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})
	text, err := p.GenerateWithProvider(context.Background(), "opencode", "big-pickle", "sys", "msg", 5*time.Second)
	if err != nil {
		t.Fatalf("expected success with opencode in health runtimes, got: %v", err)
	}
	if text != "opencode-health-ok" {
		t.Fatalf("result = %q, want opencode-health-ok", text)
	}
}
