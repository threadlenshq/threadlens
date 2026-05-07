package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
