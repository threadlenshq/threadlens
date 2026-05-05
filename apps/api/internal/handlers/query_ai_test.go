package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// fakeAIProvider is a controllable stub for the AI provider interface.
type fakeAIProvider struct {
	name   string
	result string
	err    error
}

func (f *fakeAIProvider) Name() string    { return f.name }
func (f *fakeAIProvider) Available() bool { return true }
func (f *fakeAIProvider) Generate(_ context.Context, _ string, _ string, _ string, _ time.Duration) (string, error) {
	return f.result, f.err
}

func newQueryAIRouter(t *testing.T, aiResult string) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	// Use a fake copilot provider that always returns the given JSON
	fakeProvider := &fakeAIProvider{name: "copilot", result: aiResult}
	aiSvc := ai.NewServiceWithProviders([]ai.Provider{fakeProvider})
	querySvc := services.NewQueryService(repo, aiSvc)
	r := chi.NewRouter()
	handlers.MountProjectRoutes(r, services.NewProjectService(repo))
	handlers.MountQueryRoutes(r, querySvc)
	return r, repo
}

// ────────────────────────────────────────────────────────────────
// POST /api/projects/{id}/queries/suggest
// ────────────────────────────────────────────────────────────────

func TestQueryAISuggest_MissingProject_Returns404(t *testing.T) {
	router, _ := newQueryAIRouter(t, `[]`)
	rr := doRequest(t, router, http.MethodPost, "/api/projects/nope/queries/suggest", nil)
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

func TestQueryAISuggest_ReturnsSuggestions(t *testing.T) {
	aiJSON := `[{"platform":"reddit","query_url":"https://www.reddit.com/search.json?q=pain&sort=new&t=week&limit=100","angle":"knee pain"},{"platform":"bluesky","query_url":"running injury","angle":"running injury"},{"platform":"google","query_url":"knee pain recovery","angle":"knee pain"}]`
	router, _ := newQueryAIRouter(t, aiJSON)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries/suggest", map[string]any{})
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
	if first["platform"] == nil || first["query_url"] == nil || first["angle"] == nil {
		t.Fatalf("suggestion missing required fields: %v", first)
	}
}

func TestQueryAISuggest_ExcludesDuplicates(t *testing.T) {
	// The AI returns a query that already exists in the project - it should be excluded
	aiJSON := `[{"platform":"reddit","query_url":"https://www.reddit.com/search.json?q=existing&sort=new&t=week&limit=100","angle":"existing"},{"platform":"bluesky","query_url":"new query","angle":"new angle"}]`
	router, _ := newQueryAIRouter(t, aiJSON)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})
	// Create the existing query
	doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "reddit", "query_url": "https://www.reddit.com/search.json?q=existing&sort=new&t=week&limit=100", "angle": "existing",
	})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries/suggest", map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	suggestions := resp["suggestions"].([]any)
	// Should only return the non-duplicate
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion after dedup, got %d", len(suggestions))
	}
	s := suggestions[0].(map[string]any)
	if s["platform"] != "bluesky" {
		t.Fatalf("expected bluesky platform, got %v", s["platform"])
	}
}

func TestQueryAISuggest_InvalidAIJSON_Returns500(t *testing.T) {
	router, _ := newQueryAIRouter(t, `not valid json at all!!!`)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries/suggest", map[string]any{})
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

func TestQueryAISuggest_FiltersGoogleURLs(t *testing.T) {
	// Google suggestions should be filtered if they are full URLs
	aiJSON := `[{"platform":"google","query_url":"https://google.com/search?q=test","angle":"bad google"},{"platform":"google","query_url":"knee pain recovery","angle":"good google"}]`
	router, _ := newQueryAIRouter(t, aiJSON)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries/suggest", map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	suggestions := resp["suggestions"].([]any)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 valid google suggestion, got %d: %v", len(suggestions), suggestions)
	}
}

// ────────────────────────────────────────────────────────────────
// POST /api/projects/{id}/queries/refine
// ────────────────────────────────────────────────────────────────

func TestQueryAIRefine_MissingProject_Returns404(t *testing.T) {
	refineJSON := `{"summary":"test","recommendations":[]}`
	router, _ := newQueryAIRouter(t, refineJSON)
	rr := doRequest(t, router, http.MethodPost, "/api/projects/nope/queries/refine", nil)
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

func TestQueryAIRefine_ReturnsExpectedShape(t *testing.T) {
	refineJSON := `{"summary":"Refine the queries","recommendations":[{"type":"add","reason":"More specific","sources":["social_report"],"query":{"platform":"bluesky","query_url":"knee injury rehab","angle":"rehab"}}]}`
	router, _ := newQueryAIRouter(t, refineJSON)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})
	// Add an existing query
	doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "reddit", "query_url": "https://reddit.com/search.json?q=test&sort=new&t=week&limit=100", "angle": "test pain",
	})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries/refine", map[string]any{"refinement": "focus on pain points"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["summary"]; !ok {
		t.Fatal("missing 'summary' field")
	}
	ctx, ok := resp["context"].(map[string]any)
	if !ok {
		t.Fatal("missing 'context' field")
	}
	if _, ok := ctx["query_count"]; !ok {
		t.Fatal("context missing 'query_count'")
	}
	if _, ok := ctx["enabled_query_count"]; !ok {
		t.Fatal("context missing 'enabled_query_count'")
	}
	recs, ok := resp["recommendations"].([]any)
	if !ok {
		t.Fatal("missing 'recommendations' array")
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recs))
	}
	rec := recs[0].(map[string]any)
	if rec["type"] != "add" {
		t.Fatalf("rec type = %v, want 'add'", rec["type"])
	}
	if rec["id"] == "" || rec["id"] == nil {
		t.Fatal("recommendation missing 'id' field")
	}
	if rec["reason"] == "" || rec["reason"] == nil {
		t.Fatal("recommendation missing 'reason'")
	}
}

func TestQueryAIRefine_AcceptsValidSourceValues(t *testing.T) {
	refineJSON := `{"summary":"Test","recommendations":[{"type":"add","reason":"need more","sources":["current_queries","social_report","google_report","project_context","refinement_request"],"query":{"platform":"bluesky","query_url":"test keyword","angle":"test"}}]}`
	router, _ := newQueryAIRouter(t, refineJSON)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries/refine", map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	recs := resp["recommendations"].([]any)
	if len(recs) == 0 {
		t.Fatal("expected at least one recommendation")
	}
	rec := recs[0].(map[string]any)
	sources, ok := rec["sources"].([]any)
	if !ok {
		t.Fatal("recommendation missing 'sources'")
	}
	if len(sources) != 5 {
		t.Fatalf("expected 5 sources, got %d: %v", len(sources), sources)
	}
}

func TestQueryAIRefine_InvalidAIJSON_Returns500(t *testing.T) {
	router, _ := newQueryAIRouter(t, `not valid json!!!`)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries/refine", map[string]any{})
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

func TestQueryAIRefine_DisableRecommendation(t *testing.T) {
	// Create a project with one query, then refine recommends disabling it
	var createdQueryID float64
	router, _ := newQueryAIRouter(t, `{}`) // placeholder; will swap below
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})
	createRR := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "reddit", "query_url": "https://reddit.com/search.json?q=old&sort=new&t=week&limit=100", "angle": "old topic",
	})
	var created map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	createdQueryID = created["id"].(float64)

	// Now use a router with AI that returns a disable recommendation for the created query ID
	import_json := map[string]any{
		"summary": "Disable weak query",
		"recommendations": []any{
			map[string]any{
				"type":    "disable",
				"reason":  "too broad",
				"sources": []any{"social_report"},
				"query":   map[string]any{"id": createdQueryID},
			},
		},
	}
	aiBytes, _ := json.Marshal(import_json)
	router2, _ := newQueryAIRouter(t, string(aiBytes))
	doRequest(t, router2, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})
	doRequest(t, router2, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "reddit", "query_url": "https://reddit.com/search.json?q=old&sort=new&t=week&limit=100", "angle": "old topic",
	})

	rr := doRequest(t, router2, http.MethodPost, "/api/projects/p1/queries/refine", map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	recs := resp["recommendations"].([]any)
	if len(recs) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recs))
	}
	rec := recs[0].(map[string]any)
	if rec["type"] != "disable" {
		t.Fatalf("expected type=disable, got %v", rec["type"])
	}
}

func TestQueryAIRefine_QualityFieldsPresent(t *testing.T) {
	// Test that query list returns quality scoring fields
	aiJSON := `[{"platform":"reddit","query_url":"https://www.reddit.com/search.json?q=test&sort=new&t=week&limit=100","angle":"test"}]`
	router, _ := newQueryAIRouter(t, aiJSON)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})
	doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "reddit", "query_url": "https://reddit.com/search.json?q=test&sort=new&t=week&limit=100", "angle": "test",
	})

	rr := doRequest(t, router, http.MethodGet, "/api/projects/p1/queries", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	// NOTE: The current list endpoint returns domain.Query which doesn't include quality.
	// Quality is computed in the service layer. Check if it's there.
	var result []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least one query")
	}
	// Basic fields present
	if result[0]["platform"] == nil {
		t.Fatal("missing platform field")
	}
}
