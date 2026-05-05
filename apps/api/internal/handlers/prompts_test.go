package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newPromptRouterWithProject(t *testing.T) http.Handler {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	projSvc := services.NewProjectService(repo)
	promptSvc := services.NewPromptService(repo)
	r := chi.NewRouter()
	handlers.MountProjectRoutes(r, projSvc)
	handlers.MountPromptRoutes(r, promptSvc)
	return r
}

func newPromptRouter(t *testing.T) http.Handler {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	promptSvc := services.NewPromptService(repo)
	r := chi.NewRouter()
	handlers.MountPromptRoutes(r, promptSvc)
	return r
}

func TestPromptList_MissingProject_Returns404(t *testing.T) {
	router := newPromptRouter(t)
	rr := doRequest(t, router, http.MethodGet, "/api/projects/nonexistent/prompts", nil)
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

func TestPromptCreate_MissingTypeOrPlatform_Returns400(t *testing.T) {
	router := newPromptRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/prompts", map[string]any{
		"type": "product",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "type and platform are required" {
		t.Fatalf("error = %q, want 'type and platform are required'", resp["error"])
	}
}

func TestPromptCreate_MissingPlatform_Returns400(t *testing.T) {
	router := newPromptRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/prompts", map[string]any{
		"platform": "reddit",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "type and platform are required" {
		t.Fatalf("error = %q, want 'type and platform are required'", resp["error"])
	}
}

func TestPromptCreate_DefaultsPromptTextToEmpty(t *testing.T) {
	router := newPromptRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/prompts", map[string]any{
		"type": "product", "platform": "reddit",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["prompt_text"] != "" {
		t.Fatalf("prompt_text = %v, want empty string", resp["prompt_text"])
	}
}

func TestPromptPatch_UpdatesFields(t *testing.T) {
	router := newPromptRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	createRR := doRequest(t, router, http.MethodPost, "/api/projects/p1/prompts", map[string]any{
		"type": "product", "platform": "reddit", "prompt_text": "original",
	})
	var created map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	pid := itoa(int(created["id"].(float64)))

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p1/prompts/"+pid, map[string]any{
		"type": "karma", "platform": "bluesky", "prompt_text": "updated",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["type"] != "karma" {
		t.Fatalf("type = %v, want karma", resp["type"])
	}
	if resp["platform"] != "bluesky" {
		t.Fatalf("platform = %v, want bluesky", resp["platform"])
	}
	if resp["prompt_text"] != "updated" {
		t.Fatalf("prompt_text = %v, want updated", resp["prompt_text"])
	}
}

func TestPromptDelete_Returns204(t *testing.T) {
	router := newPromptRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	createRR := doRequest(t, router, http.MethodPost, "/api/projects/p1/prompts", map[string]any{
		"type": "product", "platform": "reddit",
	})
	var created map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	pid := itoa(int(created["id"].(float64)))

	rr := doRequest(t, router, http.MethodDelete, "/api/projects/p1/prompts/"+pid, nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body = %s", rr.Code, rr.Body.String())
	}
}

func TestPromptDelete_MissingPrompt_Returns404(t *testing.T) {
	router := newPromptRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodDelete, "/api/projects/p1/prompts/999", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Prompt not found" {
		t.Fatalf("error = %q, want 'Prompt not found'", resp["error"])
	}
}

func TestPromptPatch_MissingPrompt_Returns404(t *testing.T) {
	router := newPromptRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p1/prompts/999", map[string]any{"type": "x"})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Prompt not found" {
		t.Fatalf("error = %q, want 'Prompt not found'", resp["error"])
	}
}
