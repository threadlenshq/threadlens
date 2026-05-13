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
// a real database.  It records all calls so tests can verify delegation.
type stubService struct {
	status    onboarding.Status
	statusErr error
	saveErr   error
	resetErr  error

	// call recorders
	statusCalls int
	saveCalls   int
	resetCalls  int
	gotValues   map[string]string
}

func (s *stubService) GetStatus(_ context.Context) (onboarding.Status, error) {
	s.statusCalls++
	return s.status, s.statusErr
}

func (s *stubService) Save(_ context.Context, values map[string]string) error {
	s.saveCalls++
	s.gotValues = values
	return s.saveErr
}

func (s *stubService) Reset(_ context.Context, _ onboarding.ResetMode) error {
	s.resetCalls++
	return s.resetErr
}

func (s *stubService) SaveRequiredStep(_ context.Context, _ onboarding.RequiredStep, _ map[string]string) (onboarding.Status, error) {
	return s.status, nil
}

func (s *stubService) UpdateExploration(_ context.Context, _ onboarding.ExplorationUpdate) (onboarding.Status, error) {
	return s.status, nil
}

func (s *stubService) CreateStarterProject(_ context.Context, _ onboarding.StarterProjectRequest) (onboarding.StarterProjectResult, error) {
	return onboarding.StarterProjectResult{}, nil
}

// ErrOnboardingDisabled is the sentinel error the service returns when
// onboarding is disabled.  The handler maps this to HTTP 403.
// Using a sentinel keeps handler logic free of brittle string-matching.
var ErrOnboardingDisabled = onboarding.ErrDisabled

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

// assertErrorBody verifies the response body is a JSON object with an "error"
// key and that the body does not contain any of the forbidden strings.
func assertErrorBody(t *testing.T, rr *httptest.ResponseRecorder, forbidden ...string) {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	if _, ok := resp["error"]; !ok {
		t.Error("error response missing 'error' key")
	}
	body := rr.Body.String()
	for _, s := range forbidden {
		if bytes.Contains([]byte(body), []byte(s)) {
			t.Errorf("response body must not contain %q", s)
		}
	}
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
	// Verify the handler delegated to the service exactly once.
	if svc.statusCalls != 1 {
		t.Errorf("GetStatus called %d times, want 1", svc.statusCalls)
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
	// Must surface an "error" key without leaking the raw internal error text.
	assertErrorBody(t, rr, "db gone")
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
	// Verify the handler delegated to the service exactly once with the right values.
	if svc.saveCalls != 1 {
		t.Errorf("Save called %d times, want 1", svc.saveCalls)
	}
	if svc.gotValues["ANTHROPIC_API_KEY"] != "sk-test" {
		t.Errorf("Save received values %v, want ANTHROPIC_API_KEY=sk-test", svc.gotValues)
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
	assertErrorBody(t, rr)
	// Handler must reject before calling service.
	if svc.saveCalls != 0 {
		t.Errorf("Save called %d times, want 0 on bad request", svc.saveCalls)
	}
}

func TestSaveRoute_EmptyValuesMapReturns400(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	// "values" key present but empty map.
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/save", map[string]any{
		"values": map[string]string{},
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr)
	if svc.saveCalls != 0 {
		t.Errorf("Save called %d times, want 0 on bad request", svc.saveCalls)
	}
}

func TestSaveRoute_EmptyStringValueReturns400(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	// A key present but with an empty/blank value should be rejected at the
	// handler layer before the service is called.
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/save", map[string]any{
		"values": map[string]string{"ANTHROPIC_API_KEY": ""},
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr)
	if svc.saveCalls != 0 {
		t.Errorf("Save called %d times, want 0 on bad request", svc.saveCalls)
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
	// Use the sentinel error so the handler can map it to 403 without
	// string-matching internal error text.
	svc := &stubService{saveErr: onboarding.ErrDisabled}
	router := newOnboardingRouter(svc)

	payload := map[string]any{
		"values": map[string]string{"ANTHROPIC_API_KEY": "sk-test"},
	}
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/save", payload)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr)
}

func TestSaveRoute_ServiceErrorDoesNotLeakRawValue(t *testing.T) {
	internalErrText := "open /secret/internal/path: permission denied"
	svc := &stubService{saveErr: errors.New(internalErrText)}
	router := newOnboardingRouter(svc)

	payload := map[string]any{
		"values": map[string]string{"ANTHROPIC_API_KEY": "sk-ant-supersecret"},
	}
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/save", payload)

	// Must be a 500 (not 200/403).
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rr.Code, rr.Body.String())
	}
	// Must surface an "error" key without leaking the raw API key or internal path.
	assertErrorBody(t, rr, "sk-ant-supersecret", internalErrText)
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
	assertErrorBody(t, rr)
}

// ── POST /api/onboarding/reset ────────────────────────────────────────────────

func TestResetRoute_ClearsCompletionState(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/reset", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	// Verify the handler delegated to the service exactly once.
	if svc.resetCalls != 1 {
		t.Errorf("Reset called %d times, want 1", svc.resetCalls)
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
	if svc.resetCalls != 2 {
		t.Errorf("Reset called %d times, want 2 across two requests", svc.resetCalls)
	}
}

func TestResetRoute_ServiceErrorReturns500(t *testing.T) {
	svc := &stubService{resetErr: errors.New("db gone")}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/reset", nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rr.Code, rr.Body.String())
	}
	// Must surface an "error" key without leaking the raw internal error text.
	assertErrorBody(t, rr, "db gone")
}
