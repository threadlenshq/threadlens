package onboarding_test

// handlers_test.go specifies the expected HTTP behaviour for the onboarding
// endpoints before the handler implementation exists.  All tests in this file
// are intentionally failing until the handler functions (and their mount
// helper) are implemented.
//
// Endpoints covered:
//   GET  /api/onboarding/status            – returns a full config-status snapshot
//   POST /api/onboarding/save              – accepts JSON payload, delegates to Service.Save
//   POST /api/onboarding/reset             – clears the completion state (accepts optional mode body)
//   POST /api/onboarding/steps/{step}      – delegates to Service.SaveRequiredStep
//   POST /api/onboarding/exploration       – delegates to Service.UpdateExploration
//   POST /api/onboarding/starter-project   – delegates to Service.CreateStarterProject
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

	saveRequiredStepErr error
	updateExplorationErr error
	createStarterProjectErr error

	starterResult onboarding.StarterProjectResult

	// call recorders
	statusCalls int
	saveCalls   int
	resetCalls  int
	gotValues   map[string]string

	gotResetMode    onboarding.ResetMode
	gotStep         onboarding.RequiredStep
	gotStepValues   map[string]string
	gotExploration  onboarding.ExplorationUpdate
	gotStarterReq   onboarding.StarterProjectRequest

	saveRequiredStepCalls    int
	updateExplorationCalls   int
	createStarterProjectCalls int
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

func (s *stubService) Reset(_ context.Context, mode onboarding.ResetMode) error {
	s.resetCalls++
	s.gotResetMode = mode
	return s.resetErr
}

func (s *stubService) SaveRequiredStep(_ context.Context, step onboarding.RequiredStep, values map[string]string) (onboarding.Status, error) {
	s.saveRequiredStepCalls++
	s.gotStep = step
	s.gotStepValues = values
	return s.status, s.saveRequiredStepErr
}

func (s *stubService) UpdateExploration(_ context.Context, req onboarding.ExplorationUpdate) (onboarding.Status, error) {
	s.updateExplorationCalls++
	s.gotExploration = req
	return s.status, s.updateExplorationErr
}

func (s *stubService) CreateStarterProject(_ context.Context, req onboarding.StarterProjectRequest) (onboarding.StarterProjectResult, error) {
	s.createStarterProjectCalls++
	s.gotStarterReq = req
	return s.starterResult, s.createStarterProjectErr
}

// ErrOnboardingDisabled is the sentinel error the service returns when
// onboarding is disabled.  The handler maps this to HTTP 403.
// Using a sentinel keeps handler logic free of brittle string-matching.
var ErrOnboardingDisabled = onboarding.ErrDisabled

// ── helpers ───────────────────────────────────────────────────────────────────

func newOnboardingRouter(svc onboarding.ServiceIface) http.Handler {
	r := chi.NewRouter()
	onboarding.MountRoutes(r, svc, nil)
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

// TestStatusRoute_ReturnsExpandedFields asserts that the status response includes
// all the fields added in the expanded onboarding API (Tasks 5/9).
func TestStatusRoute_ReturnsExpandedFields(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{
			Enabled:               true,
			Complete:              false,
			RequiredSetupComplete: false,
			ExplorationComplete:   false,
			Phase:                 onboarding.PhaseRequiredSetup,
			CurrentRequiredStep:   onboarding.RequiredStepWelcome,
		},
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

	requiredFields := []string{
		"enabled",
		"complete",
		"requiredSetupComplete",
		"explorationComplete",
		"phase",
		"currentRequiredStep",
		"steps",
		"items",
		"capabilities",
		"appDatabase",
		"context",
	}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
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

// TestResetRoute_AcceptsExplicitMode verifies the handler passes the supplied
// mode string through to the service without rewriting it.
func TestResetRoute_AcceptsExplicitMode(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/reset", map[string]any{
		"mode": "exploration",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if svc.gotResetMode != onboarding.ResetModeExploration {
		t.Errorf("Reset called with mode %q, want %q", svc.gotResetMode, onboarding.ResetModeExploration)
	}
}

// TestResetRoute_NoBodyDefaultsToProgress verifies that a request with no body
// defaults to the "progress" reset mode.
func TestResetRoute_NoBodyDefaultsToProgress(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/reset", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if svc.gotResetMode != onboarding.ResetModeProgress {
		t.Errorf("Reset called with mode %q, want %q", svc.gotResetMode, onboarding.ResetModeProgress)
	}
}

// ── POST /api/onboarding/steps/{step} ─────────────────────────────────────────

func TestSaveRequiredStepRoute_DelegatesStepAndValues(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{Enabled: true, Phase: onboarding.PhaseRequiredSetup},
	}
	router := newOnboardingRouter(svc)

	payload := map[string]any{
		"values": map[string]string{"AI_PROVIDER": "anthropic"},
	}
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/steps/ai_provider", payload)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if svc.saveRequiredStepCalls != 1 {
		t.Errorf("SaveRequiredStep called %d times, want 1", svc.saveRequiredStepCalls)
	}
	if svc.gotStep != onboarding.RequiredStepAIProvider {
		t.Errorf("SaveRequiredStep got step %q, want %q", svc.gotStep, onboarding.RequiredStepAIProvider)
	}
	if svc.gotStepValues["AI_PROVIDER"] != "anthropic" {
		t.Errorf("SaveRequiredStep got values %v, want AI_PROVIDER=anthropic", svc.gotStepValues)
	}
}

func TestSaveRequiredStepRoute_ReturnsStatusShape(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{
			Enabled:               true,
			Phase:                 onboarding.PhaseRequiredSetup,
			CurrentRequiredStep:   onboarding.RequiredStepAIProvider,
			RequiredSetupComplete: false,
		},
	}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/steps/welcome", map[string]any{
		"values": map[string]string{},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, field := range []string{"enabled", "complete", "requiredSetupComplete", "explorationComplete", "phase", "currentRequiredStep", "steps", "items"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestSaveRequiredStepRoute_UnknownStepReturns400(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/steps/not_a_real_step", map[string]any{
		"values": map[string]string{},
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr)
	if svc.saveRequiredStepCalls != 0 {
		t.Errorf("SaveRequiredStep called %d times, want 0 on bad request", svc.saveRequiredStepCalls)
	}
}

func TestSaveRequiredStepRoute_ServiceDisabledReturns403(t *testing.T) {
	svc := &stubService{saveRequiredStepErr: onboarding.ErrDisabled}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/steps/welcome", map[string]any{
		"values": map[string]string{},
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr)
}

func TestSaveRequiredStepRoute_ServiceErrorReturns500(t *testing.T) {
	svc := &stubService{saveRequiredStepErr: errors.New("db gone")}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/steps/welcome", map[string]any{
		"values": map[string]string{},
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr, "db gone")
}

// ── POST /api/onboarding/exploration ──────────────────────────────────────────

func TestUpdateExplorationRoute_DelegatesItemAndState(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{Enabled: true, Phase: onboarding.PhaseExploration},
	}
	router := newOnboardingRouter(svc)

	payload := map[string]any{
		"item":  "starter_project",
		"state": "completed",
	}
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/exploration", payload)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if svc.updateExplorationCalls != 1 {
		t.Errorf("UpdateExploration called %d times, want 1", svc.updateExplorationCalls)
	}
	if svc.gotExploration.Item != onboarding.ExplorationItemStarterProject {
		t.Errorf("UpdateExploration got item %q, want %q", svc.gotExploration.Item, onboarding.ExplorationItemStarterProject)
	}
	if svc.gotExploration.State != onboarding.ItemStateCompleted {
		t.Errorf("UpdateExploration got state %q, want %q", svc.gotExploration.State, onboarding.ItemStateCompleted)
	}
}

func TestUpdateExplorationRoute_DelegateDismiss(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{Enabled: true},
	}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/exploration", map[string]any{
		"dismiss": true,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if !svc.gotExploration.Dismiss {
		t.Error("UpdateExploration got Dismiss=false, want true")
	}
}

func TestUpdateExplorationRoute_ReturnsStatusShape(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{Enabled: true, Phase: onboarding.PhaseExploration},
	}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/exploration", map[string]any{
		"dismiss": true,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, field := range []string{"enabled", "complete", "explorationComplete", "phase", "items"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestUpdateExplorationRoute_ServiceDisabledReturns403(t *testing.T) {
	svc := &stubService{updateExplorationErr: onboarding.ErrDisabled}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/exploration", map[string]any{
		"dismiss": true,
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr)
}

func TestUpdateExplorationRoute_ServiceErrorReturns500(t *testing.T) {
	svc := &stubService{updateExplorationErr: errors.New("db error")}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/exploration", map[string]any{
		"item":  "starter_project",
		"state": "completed",
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr, "db error")
}

// ── POST /api/onboarding/starter-project ──────────────────────────────────────

func TestCreateStarterProjectRoute_DelegatesRequest(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	payload := map[string]any{
		"projectId":   "proj-1",
		"projectName": "My Project",
		"query":       "saas pain points",
		"platform":    "reddit",
	}
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/starter-project", payload)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if svc.createStarterProjectCalls != 1 {
		t.Errorf("CreateStarterProject called %d times, want 1", svc.createStarterProjectCalls)
	}
	if svc.gotStarterReq.ProjectID != "proj-1" {
		t.Errorf("CreateStarterProject got ProjectID %q, want %q", svc.gotStarterReq.ProjectID, "proj-1")
	}
	if svc.gotStarterReq.ProjectName != "My Project" {
		t.Errorf("CreateStarterProject got ProjectName %q, want %q", svc.gotStarterReq.ProjectName, "My Project")
	}
	if svc.gotStarterReq.Query != "saas pain points" {
		t.Errorf("CreateStarterProject got Query %q, want %q", svc.gotStarterReq.Query, "saas pain points")
	}
	if svc.gotStarterReq.Platform != "reddit" {
		t.Errorf("CreateStarterProject got Platform %q, want %q", svc.gotStarterReq.Platform, "reddit")
	}
}

func TestCreateStarterProjectRoute_ReturnsResultShape(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/starter-project", map[string]any{
		"projectId":   "proj-1",
		"projectName": "My Project",
		"query":       "test query",
		"platform":    "reddit",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, field := range []string{"project", "query", "createdProject", "createdQuery"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestCreateStarterProjectRoute_MissingRequiredFieldsReturns400(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)

	// Missing projectId
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/starter-project", map[string]any{
		"projectName": "My Project",
		"query":       "test query",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr)
	if svc.createStarterProjectCalls != 0 {
		t.Errorf("CreateStarterProject called %d times, want 0 on bad request", svc.createStarterProjectCalls)
	}
}

func TestCreateStarterProjectRoute_ServiceDisabledReturns403(t *testing.T) {
	svc := &stubService{createStarterProjectErr: onboarding.ErrDisabled}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/starter-project", map[string]any{
		"projectId":   "proj-1",
		"projectName": "My Project",
		"query":       "test query",
		"platform":    "reddit",
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr)
}

func TestCreateStarterProjectRoute_ServiceErrorReturns500(t *testing.T) {
	svc := &stubService{createStarterProjectErr: errors.New("db error")}
	router := newOnboardingRouter(svc)

	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/starter-project", map[string]any{
		"projectId":   "proj-1",
		"projectName": "My Project",
		"query":       "test query",
		"platform":    "reddit",
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rr.Code, rr.Body.String())
	}
	assertErrorBody(t, rr, "db error")
}

// ── POST /api/onboarding/required-step (body-encoded step) ───────────────────

func TestRequiredStepRouteDelegates(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{Phase: onboarding.PhaseRequiredSetup, CurrentRequiredStep: onboarding.RequiredStepAppDatabase},
	}
	router := newOnboardingRouter(svc)
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/required-step", map[string]any{
		"step":   "ai_provider",
		"values": map[string]string{"AI_PROVIDER": "anthropic"},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if svc.saveRequiredStepCalls != 1 {
		t.Fatalf("SaveRequiredStep calls = %d, want 1", svc.saveRequiredStepCalls)
	}
	if svc.gotStep != onboarding.RequiredStepAIProvider {
		t.Fatalf("got step %q, want ai_provider", svc.gotStep)
	}
}

func TestExplorationRouteDelegates(t *testing.T) {
	svc := &stubService{
		status: onboarding.Status{Phase: onboarding.PhaseComplete, ExplorationComplete: true},
	}
	router := newOnboardingRouter(svc)
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/exploration", map[string]any{
		"item":  "reports_intro",
		"state": "completed",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if svc.updateExplorationCalls != 1 {
		t.Fatalf("UpdateExploration calls = %d, want 1", svc.updateExplorationCalls)
	}
}

func TestStarterProjectRouteDoesNotLeakSecret(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/starter-project", map[string]any{
		"projectId":   "ai-notes",
		"projectName": "AI Notes",
		"query":       "meeting notes too time consuming",
		"platform":    "reddit",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if bytes.Contains(rr.Body.Bytes(), []byte("sk-ant")) {
		t.Fatalf("starter response leaked secret-like text: %s", rr.Body.String())
	}
}

func TestResetRouteAcceptsMode(t *testing.T) {
	svc := &stubService{}
	router := newOnboardingRouter(svc)
	rr := doOnboardingRequest(t, router, http.MethodPost, "/api/onboarding/reset", map[string]any{
		"mode": "exploration",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	if svc.gotResetMode != onboarding.ResetModeExploration {
		t.Fatalf("resetMode = %q, want exploration", svc.gotResetMode)
	}
}
