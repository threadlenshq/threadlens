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
// fakeProvider – controllable test double
// ---------------------------------------------------------------------------

type fakeProvider struct {
	name      string
	available bool
	result    string
	err       error
	calls     int
}

func (f *fakeProvider) Name() string    { return f.name }
func (f *fakeProvider) Available() bool { return f.available }
func (f *fakeProvider) Generate(_ context.Context, _ string, _ string, _ string, _ time.Duration) (string, error) {
	f.calls++
	return f.result, f.err
}

// ---------------------------------------------------------------------------
// fakeSettingsGetter – minimal settings test double
// ---------------------------------------------------------------------------

type fakeSettingsGetter struct {
	values map[string]string
}

func (f fakeSettingsGetter) GetSetting(_ context.Context, key string) (string, bool, error) {
	value, ok := f.values[key]
	return value, ok, nil
}

// newTestService builds a Service with the given fake providers and a NoopMeter.
func newTestService(providers []Provider) *Service {
	return &Service{providers: providers, meter: usage.NoopMeter{}}
}

// ---------------------------------------------------------------------------
// GenerateAuto tests
// ---------------------------------------------------------------------------

func TestGenerateAuto_UsesCachedProvider(t *testing.T) {
	p := &fakeProvider{name: "copilot", available: true, result: "hello"}
	svc := newTestService([]Provider{p})
	svc.cachedProvider = p // prime the cache

	got, err := svc.GenerateAuto(context.Background(), "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	if p.calls != 1 {
		t.Errorf("expected 1 call to cached provider, got %d", p.calls)
	}
}

func TestGenerateAuto_ClearsCache_WhenCachedProviderFails(t *testing.T) {
	bad := &fakeProvider{name: "copilot", available: true, err: errors.New("boom")}
	good := &fakeProvider{name: "claude", available: true, result: "ok"}
	svc := newTestService([]Provider{bad, good})
	svc.cachedProvider = bad // prime with provider that will fail

	got, err := svc.GenerateAuto(context.Background(), "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" {
		t.Errorf("expected 'ok', got %q", got)
	}
	// Cache should now point to 'claude' (fallback that worked)
	if svc.cachedProvider == nil || svc.cachedProvider.Name() != "claude" {
		t.Errorf("expected cache to be 'claude', got %v", svc.cachedProvider)
	}
}

func TestGenerateAuto_FallbackOrder(t *testing.T) {
	// All unavailable except sdk:haiku
	copilot := &fakeProvider{name: "copilot", available: false}
	claude := &fakeProvider{name: "claude", available: false}
	sdk := &fakeProvider{name: "sdk", available: true, result: "sdk-result"}
	gemini := &fakeProvider{name: "gemini", available: true, result: "gemini-result"}

	svc := newTestService([]Provider{copilot, claude, sdk, gemini})

	got, err := svc.GenerateAuto(context.Background(), "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sdk-result" {
		t.Errorf("expected 'sdk-result', got %q", got)
	}
	// sdk should now be cached
	if svc.cachedProvider == nil || svc.cachedProvider.Name() != "sdk" {
		t.Errorf("expected cache='sdk', got %v", svc.cachedProvider)
	}
}

func TestGenerateAuto_FallbackOrderComplete(t *testing.T) {
	// Verify the exact order the fallbacks are tried by recording call sequence.
	var order []string
	makeProvider := func(name string, available bool) *fakeProvider {
		return &fakeProvider{
			name:      name,
			available: available,
			err:       errors.New("fail"),
		}
	}
	_ = order

	copilot := makeProvider("copilot", true)
	claude := makeProvider("claude", true)
	sdk := makeProvider("sdk", true)
	gemini := &fakeProvider{name: "gemini", available: true, result: "last"}

	svc := newTestService([]Provider{copilot, claude, sdk, gemini})
	got, err := svc.GenerateAuto(context.Background(), "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "last" {
		t.Errorf("expected 'last' from gemini, got %q", got)
	}
	// Each of copilot, claude, sdk should have been tried exactly once.
	for _, p := range []*fakeProvider{copilot, claude, sdk} {
		if p.calls != 1 {
			t.Errorf("provider %q: expected 1 call, got %d", p.name, p.calls)
		}
	}
}

// ---------------------------------------------------------------------------
// GenerateForTask tests
// ---------------------------------------------------------------------------

func TestGenerateForTask_TriesPrimaryFirst(t *testing.T) {
	// post_scoring default is copilot:gpt-5-mini
	copilot := &fakeProvider{name: "copilot", available: true, result: "scored"}
	svc := newTestService([]Provider{copilot})

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "scored" {
		t.Errorf("expected 'scored', got %q", result)
	}
	if usedID != "copilot:gpt-5-mini" {
		t.Errorf("expected usedID='copilot:gpt-5-mini', got %q", usedID)
	}
}

func TestGenerateForTask_FallsBackWhenPrimaryFails(t *testing.T) {
	copilot := &fakeProvider{name: "copilot", available: true, err: errors.New("copilot down")}
	claude := &fakeProvider{name: "claude", available: true, result: "fallback-result"}

	svc := newTestService([]Provider{copilot, claude})
	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "fallback-result" {
		t.Errorf("expected 'fallback-result', got %q", result)
	}
	if usedID != "claude-cli:haiku" {
		t.Errorf("expected usedID='claude-cli:haiku', got %q", usedID)
	}
}

func TestGenerateForTask_SkipsUnavailableFallbacks(t *testing.T) {
	copilot := &fakeProvider{name: "copilot", available: true, err: errors.New("down")}
	claude := &fakeProvider{name: "claude", available: false} // unavailable
	sdk := &fakeProvider{name: "sdk", available: false}       // unavailable
	gemini := &fakeProvider{name: "gemini", available: true, result: "gemini-ok"}

	svc := newTestService([]Provider{copilot, claude, sdk, gemini})
	result, _, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "gemini-ok" {
		t.Errorf("expected 'gemini-ok', got %q", result)
	}
}

func TestGenerateForTask_UnknownTask(t *testing.T) {
	svc := newTestService(nil)
	_, _, err := svc.GenerateForTask(context.Background(), "nonexistent_task", "sys", "msg")
	if err == nil {
		t.Fatal("expected error for unknown task")
	}
}

func TestFallbackOrderExactCatalogModelIDs(t *testing.T) {
	want := []string{
		"copilot:gpt-5-mini",
		"opencode-go:deepseek-v4-flash",
		"claude-cli:haiku",
		"sdk:haiku",
		"gemini:2.5-flash",
	}
	if len(fallbackOrder) != len(want) {
		t.Fatalf("fallbackOrder length = %d, want %d", len(fallbackOrder), len(want))
	}
	for i := range want {
		if fallbackOrder[i] != want[i] {
			t.Fatalf("fallbackOrder[%d] = %q, want %q", i, fallbackOrder[i], want[i])
		}
	}
}

func TestGenerateForTask_AttemptsUnavailablePrimaryOnce(t *testing.T) {
	copilot := &fakeProvider{name: "copilot", available: false, result: "primary-still-tried"}
	sdk := &fakeProvider{name: "sdk", available: true, result: "sdk-fallback"}
	svc := newTestService([]Provider{copilot, sdk})

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "primary-still-tried" {
		t.Fatalf("result = %q, want primary-still-tried", result)
	}
	if usedID != "copilot:gpt-5-mini" {
		t.Fatalf("usedID = %q, want copilot:gpt-5-mini", usedID)
	}
	if copilot.calls != 1 {
		t.Fatalf("copilot calls = %d, want 1", copilot.calls)
	}
}

func TestGenerateForTask_SDKPrimaryNeverUsesBridge(t *testing.T) {
	// Wire a real *BridgeProvider to a test server that records any call — there should be none.
	// After Task 2, bridge is stored in s.bridge; these tests construct Service literals with the
	// bridge still in providers (s.bridge == nil) so bridgeProvider() returns nil and bridge is
	// never consulted, which is the correct invariant: sdk must never route through bridge.
	bridgeCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bridgeCalled = true
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "text": "bridge-result"})
	}))
	defer srv.Close()
	realBridge := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})

	sdk := &fakeProvider{name: "sdk", available: true, result: "sdk-result"}
	svc := &Service{
		repo: fakeSettingsGetter{values: map[string]string{
			"model.post_scoring": `{"modelId":"sdk:haiku"}`,
		}},
		providers: []Provider{realBridge, sdk},
		meter:     usage.NoopMeter{},
	}

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "sdk-result" {
		t.Fatalf("result = %q, want sdk-result", result)
	}
	if usedID != "sdk:haiku" {
		t.Fatalf("usedID = %q, want sdk:haiku", usedID)
	}
	if bridgeCalled {
		t.Fatal("bridge server was called for sdk primary — sdk must never route through bridge")
	}
}

func TestGenerateForTask_GeminiPrimaryNeverUsesBridge(t *testing.T) {
	// Wire a real *BridgeProvider to a test server that records any call — there should be none.
	// gemini must never route through bridge regardless of bridge configuration.
	bridgeCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bridgeCalled = true
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "text": "bridge-result"})
	}))
	defer srv.Close()
	realBridge := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})

	gemini := &fakeProvider{name: "gemini", available: true, result: "gemini-result"}
	svc := &Service{
		repo: fakeSettingsGetter{values: map[string]string{
			"model.post_scoring": `{"modelId":"gemini:2.5-flash"}`,
		}},
		providers: []Provider{realBridge, gemini},
		meter:     usage.NoopMeter{},
	}

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "gemini-result" {
		t.Fatalf("result = %q, want gemini-result", result)
	}
	if usedID != "gemini:2.5-flash" {
		t.Fatalf("usedID = %q, want gemini:2.5-flash", usedID)
	}
	if bridgeCalled {
		t.Fatal("bridge server was called for gemini primary — gemini must never route through bridge")
	}
}

func TestGenerateForTask_OpencodeGoFallbackWhenCopilotFails(t *testing.T) {
	copilot := &fakeProvider{name: "copilot", available: true, err: errors.New("copilot down")}
	opencode := &fakeProvider{name: "opencode", available: true, result: "opencode-fallback"}
	claude := &fakeProvider{name: "claude", available: true, result: "claude-result"}
	sdk := &fakeProvider{name: "sdk", available: true, result: "sdk-result"}
	gemini := &fakeProvider{name: "gemini", available: true, result: "gemini-result"}

	svc := newTestService([]Provider{copilot, opencode, claude, sdk, gemini})
	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "opencode-fallback" {
		t.Errorf("expected 'opencode-fallback', got %q", result)
	}
	if usedID != "opencode-go:deepseek-v4-flash" {
		t.Errorf("expected usedID='opencode-go:deepseek-v4-flash', got %q", usedID)
	}
}

func TestGenerateForTask_OpencodeGoPrimaryUsesBridge(t *testing.T) {
	// opencode-go is bridge-compatible, so the bridge should be tried first.
	bridgeCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bridgeCalled = true
		if r.URL.Path == "/v1/health" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":       true,
				"runtimes": []string{"copilot", "claude-cli", "opencode"},
			})
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		// Verify the bridge receives "opencode" (not "opencode-go") as the provider.
		if p, ok := body["provider"].(string); ok && p != "opencode" {
			t.Fatalf("bridge received provider = %q, want opencode", p)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"text": "bridge-result"})
	}))
	defer srv.Close()
	realBridge := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})

	opencode := &fakeProvider{name: "opencode", available: true, result: "opencode-direct"}
	svc := &Service{
		repo: fakeSettingsGetter{values: map[string]string{
			"model.post_scoring": `{"modelId":"opencode-go:deepseek-v4-flash"}`,
		}},
		providers: []Provider{opencode},
		bridge:    realBridge,
		meter:     usage.NoopMeter{},
	}

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "bridge-result" {
		t.Fatalf("result = %q, want bridge-result (opencode-go is bridge-compatible)", result)
	}
	if usedID != "opencode-go:deepseek-v4-flash" {
		t.Fatalf("usedID = %q, want opencode-go:deepseek-v4-flash", usedID)
	}
	if !bridgeCalled {
		t.Fatal("bridge should be called for opencode-go primary")
	}
}

func TestGenerateForTask_AllProvidersFailErrorOmitsBridgeStartupAdviceAndSecrets(t *testing.T) {
	copilot := &fakeProvider{name: "copilot", available: true, err: errors.New("copilot token /tmp/secret-token unavailable")}
	opencode := &fakeProvider{name: "opencode", available: true, err: errors.New("opencode not authenticated")}
	claude := &fakeProvider{name: "claude", available: true, err: errors.New("claude credentials missing")}
	sdk := &fakeProvider{name: "sdk", available: true, err: errors.New("ANTHROPIC_API_KEY missing")}
	gemini := &fakeProvider{name: "gemini", available: true, err: errors.New("GEMINI_API_KEY missing")}
	svc := newTestService([]Provider{copilot, opencode, claude, sdk, gemini})

	_, _, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err == nil {
		t.Fatal("expected all providers failure")
	}
	errText := err.Error()
	for _, required := range []string{"post_scoring", "copilot:gpt-5-mini", "opencode-go:deepseek-v4-flash", "claude-cli:haiku", "sdk:haiku", "gemini:2.5-flash"} {
		if !strings.Contains(errText, required) {
			t.Fatalf("error %q missing %q", errText, required)
		}
	}
	for _, forbidden := range []string{"/tmp/secret-token", "host CLI bridge is reachable", "start bridge"} {
		if strings.Contains(errText, forbidden) {
			t.Fatalf("error %q contains forbidden text %q", errText, forbidden)
		}
	}
}

func TestGenerateForTask_BridgeReceivesCatalogProviderTag(t *testing.T) {
	// Verify that invokeModelWithBridge forwards m.Provider to the bridge so it
	// can route to the correct host runtime (e.g. "copilot" vs "claude-cli").
	var capturedProvider string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if p, ok := body["provider"].(string); ok {
			capturedProvider = p
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"text": "bridge-result"})
	}))
	defer srv.Close()

	realBridge := newBridgeProviderForTest(BridgeState{Enabled: true, Detected: true, URL: srv.URL, Token: "tok"})
	svc := &Service{
		bridge: realBridge,
		providers: []Provider{
			&fakeProvider{name: "copilot", available: true, result: "direct-result"},
		},
		meter: usage.NoopMeter{},
	}

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "bridge-result" {
		t.Fatalf("result = %q, want bridge-result", result)
	}
	if usedID != "copilot:gpt-5-mini" {
		t.Fatalf("usedID = %q, want copilot:gpt-5-mini", usedID)
	}
	if capturedProvider != "copilot" {
		t.Fatalf("bridge received provider = %q, want copilot", capturedProvider)
	}
}

// ---------------------------------------------------------------------------
// StripMarkdown tests
// ---------------------------------------------------------------------------

func TestStripMarkdown_Bold(t *testing.T) {
	got := StripMarkdown("**hello**")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_BoldUnderscore(t *testing.T) {
	got := StripMarkdown("__hello__")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_Italic(t *testing.T) {
	got := StripMarkdown("*hello*")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_ItalicUnderscore(t *testing.T) {
	got := StripMarkdown("_hello_")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_Code(t *testing.T) {
	got := StripMarkdown("`hello`")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_EmDash(t *testing.T) {
	got := StripMarkdown("foo\u2014bar")
	if got != "foo, bar" {
		t.Errorf("expected 'foo, bar', got %q", got)
	}
}

func TestStripMarkdown_Mixed(t *testing.T) {
	input := "**Bold** and _italic_ and `code` \u2014 done"
	// Express replaces \u2014 with ", " (comma+space), so surrounding spaces are preserved.
	want := "Bold and italic and code ,  done"
	got := StripMarkdown(input)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

// ---------------------------------------------------------------------------
// GenerateForTask usage metering tests
// ---------------------------------------------------------------------------

func TestGenerateForTaskRecordsUsageEvent(t *testing.T) {
	meter := usage.NewMemoryMeter()
	copilot := &fakeProvider{name: "copilot", available: true, result: "scored"}
	svc := &Service{providers: []Provider{copilot}, meter: meter}

	_, _, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := meter.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 usage event, got %d", len(events))
	}
	ev := events[0]
	if ev.TaskID != "post_scoring" {
		t.Errorf("expected TaskID='post_scoring', got %q", ev.TaskID)
	}
	if ev.ModelID == "" {
		t.Errorf("expected non-empty ModelID")
	}
	if ev.Operation != "ai.generate_for_task" {
		t.Errorf("expected Operation='ai.generate_for_task', got %q", ev.Operation)
	}
	if !ev.Success {
		t.Errorf("expected Success=true")
	}
}
