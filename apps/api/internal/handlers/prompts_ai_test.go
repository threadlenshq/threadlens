package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newPromptAIRouter(t *testing.T, aiResult string, aiErr error) http.Handler {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	fakeProvider := &fakeAIProvider{name: "copilot", result: aiResult, err: aiErr}
	aiSvc := ai.NewServiceWithProviders([]ai.Provider{fakeProvider})
	promptSvc := services.NewPromptService(repo, aiSvc)
	projSvc := services.NewProjectService(repo, entitlements.RuntimeModeSelfHosted, entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil))
	r := chi.NewRouter()
	handlers.MountProjectRoutes(r, projSvc)
	handlers.MountPromptRoutes(r, promptSvc)
	return r
}

func validPromptSuggestionJSON() string {
	return `[
		{"text":"Write a Reddit post about ergonomic chairs for back pain relief.","label":"Back pain angle"},
		{"text":"Draft a Reddit post about standing desk benefits for remote workers.","label":"Standing desk"},
		{"text":"Create a Reddit post about the hidden costs of cheap office chairs.","label":"Quality matters"}
	]`
}

// Test 1: Missing project returns 404
func TestPromptAISuggest_MissingProject_Returns404(t *testing.T) {
	router := newPromptAIRouter(t, validPromptSuggestionJSON(), nil)
	rr := doRequest(t, router, http.MethodPost, "/api/projects/nonexistent/prompts/suggest", map[string]any{
		"platform": "reddit", "type": "product",
	})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Project not found" {
		t.Fatalf("error = %q, want 'Project not found'", resp["error"])
	}
}

// Test 2: Valid request returns 200 with suggestions
func TestPromptAISuggest_ValidRequest_Returns200(t *testing.T) {
	router := newPromptAIRouter(t, validPromptSuggestionJSON(), nil)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "ErgoChair", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/prompts/suggest", map[string]any{
		"platform": "reddit", "type": "product",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	suggestions, ok := resp["suggestions"].([]any)
	if !ok {
		t.Fatalf("suggestions is not array: %T", resp["suggestions"])
	}
	if len(suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(suggestions))
	}
	first := suggestions[0].(map[string]any)
	if first["text"] == nil || first["text"] == "" {
		t.Fatalf("suggestion missing text: %v", first)
	}
	if first["label"] == nil || first["label"] == "" {
		t.Fatalf("suggestion missing label: %v", first)
	}
}

// Test 3: AI unavailable returns 200 with empty list and notice
func TestPromptAISuggest_AIUnavailable_ReturnsNotice(t *testing.T) {
	router := newPromptAIRouter(t, "", errors.New("all AI providers failed: no provider"))
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/prompts/suggest", map[string]any{
		"platform": "reddit", "type": "product",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	suggestions, ok := resp["suggestions"].([]any)
	if !ok {
		t.Fatal("missing suggestions array")
	}
	if len(suggestions) != 0 {
		t.Fatalf("expected 0 suggestions, got %d", len(suggestions))
	}
	notice, _ := resp["notice"].(string)
	if notice == "" {
		t.Fatal("expected non-empty notice")
	}
}

// Test 4: Malformed JSON from AI returns 500
func TestPromptAISuggest_MalformedJSON_Returns500(t *testing.T) {
	router := newPromptAIRouter(t, "not valid json!!!", nil)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/prompts/suggest", map[string]any{
		"platform": "reddit", "type": "product",
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %s", rr.Body.String())
	}
	if resp["error"] == "" {
		t.Fatalf("expected error field in response, got: %v", resp)
	}
}
