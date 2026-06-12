package handlers_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newQueryRouter(t *testing.T) http.Handler {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	svc := services.NewQueryService(repo, ai.New())
	r := chi.NewRouter()
	handlers.MountQueryRoutes(r, svc, nil)
	return r
}

func newQueryRouterWithProject(t *testing.T) http.Handler {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	// Create a project for testing queries
	projSvc := services.NewProjectService(repo, entitlements.RuntimeModeSelfHosted, entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil))
	querySvc := services.NewQueryService(repo, ai.New())
	r := chi.NewRouter()
	handlers.MountProjectRoutes(r, projSvc)
	handlers.MountQueryRoutes(r, querySvc, nil)
	return r
}

func TestQueryList_MissingProject_Returns404(t *testing.T) {
	router := newQueryRouter(t)
	rr := doRequest(t, router, http.MethodGet, "/api/projects/nonexistent/queries", nil)
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

func TestQueryList_ReturnsEmptyArray(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodGet, "/api/projects/p1/queries", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var result []any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty array, got %v", result)
	}
}

func TestQueryCreate_MissingFields_Returns400(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "reddit",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "platform, query_url, and angle are required" {
		t.Fatalf("error = %q, want 'platform, query_url, and angle are required'", resp["error"])
	}
}

func TestQueryCreate_InvalidPlatform_Returns400(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "twitter", "query_url": "test", "angle": "test",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "platform must be reddit, bluesky, or google" {
		t.Fatalf("error = %q", resp["error"])
	}
}

func TestQueryCreate_Success_Reddit(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform":  "reddit",
		"query_url": "https://www.reddit.com/search.json?q=test",
		"angle":     "test angle",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["platform"] != "reddit" {
		t.Fatalf("platform = %v, want reddit", resp["platform"])
	}
}

func TestQueryCreate_Success_Bluesky(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform":  "bluesky",
		"query_url": "knee pain running",
		"angle":     "knee pain",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
}

func TestQueryCreate_Success_Google(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform":  "google",
		"query_url": "resume coding project",
		"angle":     "resume coding",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
}

func TestQueryCreate_MissingProject_Returns404(t *testing.T) {
	router := newQueryRouter(t)
	rr := doRequest(t, router, http.MethodPost, "/api/projects/nonexistent/queries", map[string]any{
		"platform": "reddit", "query_url": "test", "angle": "test",
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

func TestQueryPatch_Updates(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	createRR := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "reddit", "query_url": "https://reddit.com/search?q=test", "angle": "test angle",
	})
	var created map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id := created["id"].(float64)

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p1/queries/"+itoa(int(id)), map[string]any{
		"angle": "updated angle",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["angle"] != "updated angle" {
		t.Fatalf("angle = %v, want 'updated angle'", resp["angle"])
	}
}

func TestQueryPatch_MissingQuery_Returns404(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p1/queries/999", map[string]any{"angle": "x"})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Query not found" {
		t.Fatalf("error = %q, want 'Query not found'", resp["error"])
	}
}

func TestQueryDelete_Returns204(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	createRR := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "reddit", "query_url": "https://reddit.com/search?q=test", "angle": "test",
	})
	var created map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id := created["id"].(float64)

	rr := doRequest(t, router, http.MethodDelete, "/api/projects/p1/queries/"+itoa(int(id)), nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body = %s", rr.Code, rr.Body.String())
	}
}

func TestQueryDelete_MissingQuery_Returns404(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodDelete, "/api/projects/p1/queries/999", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Query not found" {
		t.Fatalf("error = %q, want 'Query not found'", resp["error"])
	}
}

func TestQueryList_OrderedByCreatedAt(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "reddit", "query_url": "https://reddit.com/search?q=first", "angle": "first",
	})
	doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform": "bluesky", "query_url": "second query", "angle": "second",
	})
	rr := doRequest(t, router, http.MethodGet, "/api/projects/p1/queries", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var result []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(result))
	}
}

func TestQueryCreate_EnabledFalse_HonorsEnabled(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform":  "reddit",
		"query_url": "https://www.reddit.com/search.json?q=test",
		"angle":     "test angle",
		"enabled":   false,
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["enabled"] != float64(0) {
		t.Fatalf("enabled = %v, want 0 (false)", resp["enabled"])
	}
}

func TestQueryCreate_EnabledDefault_IsTrue(t *testing.T) {
	router := newQueryRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform":  "reddit",
		"query_url": "https://www.reddit.com/search.json?q=default",
		"angle":     "default angle",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["enabled"] != float64(1) {
		t.Fatalf("enabled = %v, want 1 (true)", resp["enabled"])
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
