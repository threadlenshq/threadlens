package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newProjectRouter(t *testing.T) http.Handler {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	svc := services.NewProjectService(repo)
	r := chi.NewRouter()
	handlers.MountProjectRoutes(r, svc)
	return r
}

func doRequest(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
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

func TestProjectList_Empty(t *testing.T) {
	router := newProjectRouter(t)
	rr := doRequest(t, router, http.MethodGet, "/api/projects", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result []any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty array, got %v", result)
	}
}

func TestProjectCreate_MissingFields_Returns400(t *testing.T) {
	router := newProjectRouter(t)
	rr := doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "id, name, and mode are required" {
		t.Fatalf("error = %q, want 'id, name, and mode are required'", resp["error"])
	}
}

func TestProjectCreate_Success_Returns201(t *testing.T) {
	router := newProjectRouter(t)
	rr := doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id":   "p1",
		"name": "Test Project",
		"mode": "research",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["id"] != "p1" {
		t.Fatalf("id = %q, want p1", resp["id"])
	}
}

func TestProjectGet_ReturnsStats(t *testing.T) {
	router := newProjectRouter(t)
	// Create first
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id":   "p2",
		"name": "Stats Project",
		"mode": "research",
	})
	rr := doRequest(t, router, http.MethodGet, "/api/projects/p2", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	stats, ok := resp["stats"].(map[string]any)
	if !ok {
		t.Fatalf("expected stats object, got %T: %v", resp["stats"], resp["stats"])
	}
	if _, ok := stats["total_posts"]; !ok {
		t.Fatalf("stats missing total_posts: %v", stats)
	}
}

func TestProjectGet_NotFound_Returns404(t *testing.T) {
	router := newProjectRouter(t)
	rr := doRequest(t, router, http.MethodGet, "/api/projects/nonexistent", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestProjectPatch_UpdatesDescription(t *testing.T) {
	router := newProjectRouter(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p3", "name": "Patch Project", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p3", map[string]any{"description": "updated desc"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["description"] != "updated desc" {
		t.Fatalf("description = %v, want 'updated desc'", resp["description"])
	}
}

func TestProjectDelete_Returns204EmptyBody(t *testing.T) {
	router := newProjectRouter(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p4", "name": "Delete Project", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodDelete, "/api/projects/p4", nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body = %s", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("expected empty body, got: %s", rr.Body.String())
	}
}

func TestProjectClone_Returns201(t *testing.T) {
	router := newProjectRouter(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p5", "name": "Source", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p5/clone", map[string]any{
		"id": "p6", "name": "Clone",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["id"] != "p6" {
		t.Fatalf("id = %q, want p6", resp["id"])
	}
}

func TestProjectSelectAngle_MissingFields_Returns400(t *testing.T) {
	router := newProjectRouter(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p7", "name": "Angle Project", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p7/select-angle", map[string]any{})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "report_id and cluster_index are required" {
		t.Fatalf("error = %q", resp["error"])
	}
}

func TestProjectGraduate_WithoutAngle_Returns400(t *testing.T) {
	router := newProjectRouter(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p8", "name": "Graduate Project", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p8/graduate", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Select a product angle before graduating to marketing mode" {
		t.Fatalf("error = %q", resp["error"])
	}
}

func TestProjectGraduate_AlreadyMarketing_Returns400(t *testing.T) {
	router := newProjectRouter(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p9", "name": "Marketing Project", "mode": "marketing",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p9/graduate", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Project is already in marketing mode" {
		t.Fatalf("error = %q", resp["error"])
	}
}
