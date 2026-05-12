package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testhelpers "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newModelRouter(t *testing.T) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	svc := services.NewModelService(repo, entitlements.RuntimeModeSelfHosted, resolver)
	r := chi.NewRouter()
	MountModelRoutes(r, svc)
	return r, repo
}

func TestGetModelCatalog(t *testing.T) {
	router, _ := newModelRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/catalog", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	models, ok := body["models"].([]any)
	if !ok || len(models) == 0 {
		t.Fatal("expected models array")
	}
	tasks, ok := body["tasks"].([]any)
	if !ok || len(tasks) == 0 {
		t.Fatal("expected tasks array")
	}

	// Verify first model has expected fields
	firstModel := models[0].(map[string]any)
	for _, key := range []string{"id", "provider", "model", "label", "tier", "cost"} {
		if _, exists := firstModel[key]; !exists {
			t.Fatalf("model missing key %q", key)
		}
	}
}

func TestGetModelConfig_ReturnsDefaults(t *testing.T) {
	router, _ := newModelRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/config", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	// All known task IDs should be present
	for _, taskID := range []string{"post_scoring", "query_suggestion", "report_clustering"} {
		entry, ok := body[taskID].(map[string]any)
		if !ok {
			t.Fatalf("config missing task %q", taskID)
		}
		if entry["source"] != "default" {
			t.Fatalf("task %q source = %v, want default", taskID, entry["source"])
		}
		if entry["modelId"] == nil || entry["modelId"] == "" {
			t.Fatalf("task %q modelId is empty", taskID)
		}
	}
}

func TestPutModelConfig_SetsUserModel(t *testing.T) {
	router, _ := newModelRouter(t)

	body := map[string]string{"modelId": "copilot:gpt-5-mini"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/models/config/post_scoring", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["modelId"] != "copilot:gpt-5-mini" {
		t.Fatalf("modelId = %v", resp["modelId"])
	}
	if resp["source"] != "user" {
		t.Fatalf("source = %v, want user", resp["source"])
	}
}

func TestPutModelConfig_MissingModelId(t *testing.T) {
	router, _ := newModelRouter(t)

	b := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPut, "/api/models/config/post_scoring", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "modelId is required" {
		t.Fatalf("error = %v", resp["error"])
	}
}

func TestPutModelConfig_UnknownTask(t *testing.T) {
	router, _ := newModelRouter(t)

	body := map[string]string{"modelId": "copilot:gpt-5-mini"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/models/config/nonexistent_task", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Unknown task: nonexistent_task" {
		t.Fatalf("error = %v", resp["error"])
	}
}

func TestPutModelConfig_UnknownModel(t *testing.T) {
	router, _ := newModelRouter(t)

	body := map[string]string{"modelId": "fake:model"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/models/config/post_scoring", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Unknown model: fake:model" {
		t.Fatalf("error = %v", resp["error"])
	}
}

func TestDeleteModelConfig_ResetsToDefault(t *testing.T) {
	router, _ := newModelRouter(t)

	// Set a user model first
	body := map[string]string{"modelId": "sdk:haiku"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/models/config/post_scoring", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("set failed: %d %s", rr.Code, rr.Body.String())
	}

	// Now delete it
	req2 := httptest.NewRequest(http.MethodDelete, "/api/models/config/post_scoring", nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr2.Code)
	}
	if rr2.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rr2.Body.String())
	}

	// Verify config is back to default
	req3 := httptest.NewRequest(http.MethodGet, "/api/models/config", nil)
	rr3 := httptest.NewRecorder()
	router.ServeHTTP(rr3, req3)
	var config map[string]any
	json.Unmarshal(rr3.Body.Bytes(), &config)
	entry := config["post_scoring"].(map[string]any)
	if entry["source"] != "default" {
		t.Fatalf("source after delete = %v, want default", entry["source"])
	}
}

func TestDeleteModelConfig_UnknownTask(t *testing.T) {
	router, _ := newModelRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/models/config/bad_task", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	// Express returns the raw error message: "Unknown task id: {taskId}"
	expected := "Unknown task id: bad_task"
	if resp["error"] != expected {
		t.Fatalf("error = %v, want %q", resp["error"], expected)
	}
}

func TestGetModelCatalog_ExternalRuntimeShape(t *testing.T) {
	router, _ := newModelRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/catalog", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	ext, ok := body["externalRuntime"].(map[string]any)
	if !ok {
		t.Fatalf("externalRuntime missing or wrong type; body keys: %v", keysOf(body))
	}

	for _, key := range []string{"type", "detected", "availableRuntimes", "source", "autoLaunchAttempted", "message", "scope", "description"} {
		if _, exists := ext[key]; !exists {
			t.Fatalf("externalRuntime missing key %q", key)
		}
	}

	if ext["scope"] != "optional-local" {
		t.Fatalf("externalRuntime.scope = %v, want optional-local", ext["scope"])
	}

	// Must not expose secrets or paths
	rawJSON := rr.Body.String()
	for _, forbidden := range []string{"token", "tokenFile"} {
		if containsKey(rawJSON, forbidden) {
			t.Fatalf("response JSON must not contain key %q", forbidden)
		}
	}
}

func TestGetModelCatalog_ExternalRuntimeNoSecretLeak(t *testing.T) {
	// Even with bridge fully disabled the response must have the externalRuntime key
	// and must never expose token values or file paths.
	t.Setenv("SCOUT_AI_BRIDGE_DISABLE", "1")

	router, _ := newModelRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/catalog", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	ext, ok := body["externalRuntime"].(map[string]any)
	if !ok {
		t.Fatalf("externalRuntime missing or wrong type when disabled")
	}
	if ext["detected"] != false {
		t.Fatalf("detected should be false when disabled, got %v", ext["detected"])
	}

	rawJSON := rr.Body.String()
	for _, forbidden := range []string{`"token"`, `"tokenFile"`} {
		if strings.Contains(rawJSON, forbidden) {
			t.Fatalf("response JSON must not contain %q", forbidden)
		}
	}
}

func TestGetModelCatalog_ExternalRuntimeProductionLikeNoBridgeIsNotDegraded(t *testing.T) {
	// Simulate a production environment where no bridge env vars are set.
	// The catalog must still return 200 and externalRuntime.detected == false
	// with source == "policy", confirming the bridge is optional.
	t.Setenv("SCOUT_AI_BRIDGE_URL", "")
	t.Setenv("SCOUT_AI_BRIDGE_TOKEN_FILE", "")
	t.Setenv("SCOUT_AI_BRIDGE_DISABLE", "")
	t.Setenv("SCOUT_AI_BRIDGE_MODE", "")

	router, _ := newModelRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/catalog", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	ext, ok := body["externalRuntime"].(map[string]any)
	if !ok {
		t.Fatalf("externalRuntime missing or wrong type in production-like no-bridge scenario")
	}
	if ext["detected"] != false {
		t.Fatalf("externalRuntime.detected = %v, want false in no-bridge scenario", ext["detected"])
	}
	if ext["source"] != "policy" {
		t.Fatalf("externalRuntime.source = %v, want policy in no-bridge scenario", ext["source"])
	}
}

func TestGetModelCatalog_ExternalRuntimeNoURLOrPathLeak(t *testing.T) {
	// Point the bridge at a known URL and token file path. Neither should
	// appear anywhere in the catalog response, even in field values.
	const knownURL = "http://127.0.0.1:19999"
	const knownToken = "supersecrettoken"
	const knownPath = "/tmp/scout-test-token-file-leak-check.txt"

	// Write a token file so the bridge loader considers it valid.
	if err := os.WriteFile(knownPath, []byte(knownToken), 0o600); err != nil {
		t.Fatalf("failed to write temp token file: %v", err)
	}
	t.Cleanup(func() { os.Remove(knownPath) })

	t.Setenv("SCOUT_AI_BRIDGE_URL", knownURL)
	t.Setenv("SCOUT_AI_BRIDGE_TOKEN_FILE", knownPath)

	router, _ := newModelRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/catalog", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	rawJSON := rr.Body.String()
	for _, secret := range []string{knownToken, knownPath, knownURL} {
		if strings.Contains(rawJSON, secret) {
			t.Fatalf("response JSON must not contain secret/path value %q", secret)
		}
	}
}

// keysOf returns the key names of a map[string]any for diagnostic messages.
func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// containsKey reports whether the raw JSON string contains the given key name as a JSON key.
func containsKey(rawJSON, key string) bool {
	return strings.Contains(rawJSON, `"`+key+`"`)
}

func TestConfigPersists_UserOverridesReflectedInConfig(t *testing.T) {
	router, _ := newModelRouter(t)

	// Set override
	body := map[string]string{"modelId": "gemini:2.5-flash"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/models/config/query_suggestion", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Read back config
	req2 := httptest.NewRequest(http.MethodGet, "/api/models/config", nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	var config map[string]any
	json.Unmarshal(rr2.Body.Bytes(), &config)
	entry := config["query_suggestion"].(map[string]any)
	if entry["modelId"] != "gemini:2.5-flash" {
		t.Fatalf("modelId = %v, want gemini:2.5-flash", entry["modelId"])
	}
	if entry["source"] != "user" {
		t.Fatalf("source = %v, want user", entry["source"])
	}
}
