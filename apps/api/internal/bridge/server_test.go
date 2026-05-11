package bridge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestServer creates a test HTTP server with a single available runtime and a
// known bearer token. MaxBodyBytes is set to 128 bytes so oversized-body tests
// are deterministic without large allocations (the valid dispatch payload is
// ~101 bytes, and the oversized-body test sends 200 bytes).
func newTestServer(t *testing.T, rt *fakeRuntime) (*httptest.Server, string) {
	t.Helper()
	const token = "test-secret"
	reg := NewRegistry(0, rt)
	srv := httptest.NewServer(NewHandler(ServerConfig{
		Registry:     reg,
		BearerToken:  token,
		MaxBodyBytes: 128,
	}))
	t.Cleanup(srv.Close)
	return srv, token
}

func doJSON(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

// TestServerHealthRequiresAuth verifies that /v1/health returns 401 without a
// valid bearer token and 200 with one.
func TestServerHealthRequiresAuth(t *testing.T) {
	rt := &fakeRuntime{
		id:     "copilot",
		status: RuntimeStatus{ID: "copilot", Available: true},
	}
	srv, token := newTestServer(t, rt)

	// No token → 401
	resp := doJSON(t, http.MethodGet, srv.URL+"/v1/health", "", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}

	// Wrong token → 401
	resp2 := doJSON(t, http.MethodGet, srv.URL+"/v1/health", "wrong", nil)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong token, got %d", resp2.StatusCode)
	}

	// Correct token → 200
	resp3 := doJSON(t, http.MethodGet, srv.URL+"/v1/health", token, nil)
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", resp3.StatusCode)
	}
}

// TestServerHealthReturnsAvailableRuntimes verifies that /v1/health returns the
// correct JSON shape and only lists available runtimes.
func TestServerHealthReturnsAvailableRuntimes(t *testing.T) {
	rt := &fakeRuntime{
		id:     "copilot",
		status: RuntimeStatus{ID: "copilot", Available: true},
	}
	srv, token := newTestServer(t, rt)

	resp := doJSON(t, http.MethodGet, srv.URL+"/v1/health", token, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if !health.OK {
		t.Error("expected health.ok to be true")
	}
	if len(health.Runtimes) != 1 {
		t.Fatalf("expected 1 runtime, got %d", len(health.Runtimes))
	}
	if health.Runtimes[0].ID != "copilot" {
		t.Errorf("expected runtime ID 'copilot', got %q", health.Runtimes[0].ID)
	}
	if !health.Runtimes[0].Available {
		t.Error("expected runtime to be available")
	}
}

// TestServerGenerateDispatchesToRegistry verifies that POST /v1/generate
// dispatches to the registry and returns trimmed text, and that the request is
// passed through verbatim to the underlying runtime.
func TestServerGenerateDispatchesToRegistry(t *testing.T) {
	rt := &fakeRuntime{
		id:     "copilot",
		status: RuntimeStatus{ID: "copilot", Available: true},
		text:   "  hello world  ",
	}
	srv, token := newTestServer(t, rt)

	payload := GenerateRequest{
		Provider:    "copilot",
		Model:       "gpt-5-mini",
		UserMessage: "hello",
		TimeoutMs:   5000,
	}
	resp := doJSON(t, http.MethodPost, srv.URL+"/v1/generate", token, payload)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var gen GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gen); err != nil {
		t.Fatalf("decode generate response: %v", err)
	}
	if gen.Text != "hello world" {
		t.Errorf("expected trimmed text 'hello world', got %q", gen.Text)
	}

	// Verify request pass-through
	if len(rt.requests) != 1 {
		t.Fatalf("expected 1 request to runtime, got %d", len(rt.requests))
	}
	got := rt.requests[0]
	if got.Provider != payload.Provider {
		t.Errorf("provider: expected %q, got %q", payload.Provider, got.Provider)
	}
	if got.Model != payload.Model {
		t.Errorf("model: expected %q, got %q", payload.Model, got.Model)
	}
	if got.UserMessage != payload.UserMessage {
		t.Errorf("userMessage: expected %q, got %q", payload.UserMessage, got.UserMessage)
	}
}

// TestServerGenerateRequiresAuth verifies that /v1/generate returns 401 without
// a valid bearer token.
func TestServerGenerateRequiresAuth(t *testing.T) {
	rt := &fakeRuntime{
		id:     "copilot",
		status: RuntimeStatus{ID: "copilot", Available: true},
		text:   "hi",
	}
	srv, _ := newTestServer(t, rt)

	resp := doJSON(t, http.MethodPost, srv.URL+"/v1/generate", "", GenerateRequest{Provider: "copilot"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// TestServerGenerateRejectsOversizedBody verifies that a request body exceeding
// MaxBodyBytes is rejected with 413.
func TestServerGenerateRejectsOversizedBody(t *testing.T) {
	rt := &fakeRuntime{
		id:     "copilot",
		status: RuntimeStatus{ID: "copilot", Available: true},
	}
	srv, token := newTestServer(t, rt)

	// Build a body that is definitely larger than MaxBodyBytes (64).
	bigBody := strings.Repeat("x", 200)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/generate", strings.NewReader(bigBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	// Allow a short timeout so the test doesn't hang if behavior is wrong.
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", resp.StatusCode)
	}
}
