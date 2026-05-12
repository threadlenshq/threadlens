package onboarding_test

// handlers_test.go specifies the expected HTTP behaviour for the onboarding
// endpoints before the handler implementation exists.  All tests in this file
// are intentionally failing until the handler functions (and their mount
// helper) are implemented.
//
// Endpoints covered:
//   GET  /api/onboarding/status   – returns a config-status snapshot
//   POST /api/onboarding/save     – accepts JSON payload, delegates to Service.Save
//   POST /api/onboarding/reset    – clears the completion state
//
// Design constraints kept here:
//   - Tests are isolated: a stubService drives HTTP responses without touching I/O.
//   - Status codes and JSON shapes are asserted, not raw env values.
//   - Disabled-mode requests return HTTP 403 (Forbidden).
//   - Errors never leak raw secret values.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
)

// ── stub service ──────────────────────────────────────────────────────────────

// stubService implements onboarding.ServiceIface so tests never touch disk or
// a real database.
type stubService struct {
	status    onboarding.Status
	statusErr error
	saveErr   error
	resetErr  error
}

func (s *stubService) GetStatus(_ context.Context) (onboarding.Status, error) {
	return s.status, s.statusErr
}

func (s *stubService) Save(_ context.Context, values map[string]string) error {
	return s.saveErr
}

func (s *stubService) Reset(_ context.Context) error {
	return s.resetErr
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newOnboardingRouter(svc onboarding.ServiceIface) http.Handler {
	r := chi.NewRouter()
	onboarding.MountRoutes(r, svc)
	return r
}

func doOnboardingRequest(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// ── GET /api/onboarding/status ────────────────────────────────────────────────

func TestStatusRoute_ReturnsSnapshot(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{Enabled: true, Complete: false, EnvFilePath: ""},
	}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodGet, "/api/onboarding/status", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["enabled"]; !ok {
		t.Error("response missing 'enabled' field")
	}
	if _, ok := resp["complete"]; !ok {
		t.Error("response missing 'complete' field")
	}
}

func TestStatusRoute_ReflectsCompleteTrue(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{Enabled: true, Complete: true},
	}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodGet, "/api/onboarding/status", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["complete"] != true {
		t.Errorf("complete = %v, want true", resp["complete"])
	}
}

func TestStatusRoute_ReturnsEnabledFalseWhenDisabled(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{Enabled: false, Complete: false},
	}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodGet, "/api/onboarding/status", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["enabled"] != false {
		t.Errorf("enabled = %v, want false", resp["enabled"])
	}
}

func TestStatusRoute_ServiceErrorReturns500(t *testing.T) {
	svc := &stubService{statusErr: errors.New("db gone")}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodGet, "/api/onboarding/status", nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rr.Code, rr.Body.String())
	}

	// Error response must not leak the raw error string that could contain
	// secrets.  It must contain an "error" key.
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("error response missing 'error' key")
	}
}

// ── POST /api/onboarding/save ─────────────────────────────────────────────────

func TestSaveRoute_AcceptsValidPayload(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	payload := map[string]any{
		"values": map[string]string{"ANTHROPIC_API_KEY": "sk-test"},
	}
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/save", payload)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
}

func TestSaveRoute_MissingValuesFieldReturns400(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	// Payload lacks the required "values" key entirely.
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/save", map[string]any{})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("error response missing 'error' key")
	}
}

func TestSaveRoute_EmptyBodyReturns400(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/onboarding/save", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
}

func TestSaveRoute_ServiceDisabledReturns403(t *testing.T) {
	svc := &stubService{
		saveErr: errors.New("onboarding: save rejected — onboarding is disabled"),
	}
	router := newOnboardingRouter(svc)

	payload := map[string]any{
		"values": map[string]string{"ANTHROPIC_API_KEY": "sk-test"},
	}
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/save", payload)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("error response missing 'error' key")
	}
}

func TestSaveRoute_ServiceErrorDoesNotLeakRawValue(t *testing.T) {
	secretErrMsg := "onboarding: writing env file: open /secret/path: permission denied"
	svc := &stubService{saveErr: errors.New(secretErrMsg)}
	router := newOnboardingRouter(svc)

	payload := map[string]any{
		"values": map[string]string{"ANTHROPIC_API_KEY": "sk-ant-supersecret"},
	}
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/save", payload)

	body := rr.Body.String()
	// The raw API key value must never appear in the response body.
	if bytes.Contains([]byte(body), []byte("sk-ant-supersecret")) {
		t.Error("response body must not contain the raw API key value")
	}
}

func TestSaveRoute_ServiceOtherErrorReturns500(t *testing.T) {
	svc := &stubService{saveErr: errors.New("unexpected db error")}
	router := newOnboardingRouter(svc)

	payload := map[string]any{
		"values": map[string]string{"ANTHROPIC_API_KEY": "sk-test"},
	}
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/save", payload)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rr.Code, rr.Body.String())
	}
}

// ── POST /api/onboarding/reset ────────────────────────────────────────────────

func TestResetRoute_ClearsCompletionState(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/reset", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
}

func TestResetRoute_IsIdempotent(t *testing.T) {
	svc := &stubService{} // resetErr stays nil
	router := newOnboardingRouter(svc)

	for i := range 2 {
		rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/reset", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("call %d: status = %d, want 200; body = %s", i+1, rr.Code, rr.Body.String())
		}
	}
}

func TestResetRoute_ServiceErrorReturns500(t *testing.T) {
	svc := &stubService{resetErr: errors.New("db gone")}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/reset", nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("error response missing 'error' key")
	}
}

func TestResetRoute_DisabledModeStillAllowsReset(t *testing.T) {
	// Reset should work even when the onboarding flow is disabled — it is an
	// administrative operation, not a user-facing flow gate.
	svc := &stubService{
		status: onboarding.Status{Enabled: false},
		// resetErr stays nil — service accepts the call
	}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/reset", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even in disabled mode; body = %s", rr.Code, rr.Body.String())
	}
}
