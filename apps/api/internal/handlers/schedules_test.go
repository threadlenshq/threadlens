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
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/scheduler"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newScheduleRouter(t *testing.T) http.Handler {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	aiSvc := ai.New()
	runner := pipeline.NewRunner(repo, aiSvc)
	sched := scheduler.New(repo, runner, nil)
	svc := services.NewScheduleService(repo, sched)
	r := chi.NewRouter()
	handlers.MountScheduleRoutes(r, svc)
	return r
}

func newScheduleRouterWithProject(t *testing.T) http.Handler {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	aiSvc := ai.New()
	runner := pipeline.NewRunner(repo, aiSvc)
	sched := scheduler.New(repo, runner, nil)
	projSvc := services.NewProjectService(repo, entitlements.RuntimeModeSelfHosted, entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil))
	scheduleSvc := services.NewScheduleService(repo, sched)
	r := chi.NewRouter()
	handlers.MountProjectRoutes(r, projSvc)
	handlers.MountScheduleRoutes(r, scheduleSvc)
	return r
}

func TestScheduleList_MissingProject_Returns404(t *testing.T) {
	router := newScheduleRouter(t)
	rr := doRequest(t, router, http.MethodGet, "/api/projects/nonexistent/schedules", nil)
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

func TestScheduleList_ReturnsEmptyArray(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodGet, "/api/projects/p1/schedules", nil)
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

func TestScheduleList_OrderedByCreatedAt(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	doRequest(t, router, http.MethodPost, "/api/projects/p1/schedules", map[string]any{
		"platform": "reddit", "cron_expr": "0 9 * * *",
	})
	doRequest(t, router, http.MethodPost, "/api/projects/p1/schedules", map[string]any{
		"platform": "bluesky", "cron_expr": "0 10 * * *",
	})
	rr := doRequest(t, router, http.MethodGet, "/api/projects/p1/schedules", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var result []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(result))
	}
}

func TestScheduleCreate_MissingFields_Returns400(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/schedules", map[string]any{
		"platform": "reddit",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "platform and cron_expr are required" {
		t.Fatalf("error = %q, want 'platform and cron_expr are required'", resp["error"])
	}
}

func TestScheduleCreate_InvalidPlatform_Returns400(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/schedules", map[string]any{
		"platform": "twitter", "cron_expr": "0 9 * * *",
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

func TestScheduleCreate_MissingProject_Returns404(t *testing.T) {
	router := newScheduleRouter(t)
	rr := doRequest(t, router, http.MethodPost, "/api/projects/nonexistent/schedules", map[string]any{
		"platform": "reddit", "cron_expr": "0 9 * * *",
	})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
}

func TestScheduleCreate_Success(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/schedules", map[string]any{
		"platform": "reddit", "cron_expr": "0 9 * * *",
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
	if resp["enabled"] != float64(1) {
		t.Fatalf("enabled = %v, want 1", resp["enabled"])
	}
}

func TestSchedulePatch_NoOp_ReturnsExisting(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	createRR := doRequest(t, router, http.MethodPost, "/api/projects/p1/schedules", map[string]any{
		"platform": "reddit", "cron_expr": "0 9 * * *",
	})
	var created map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	sid := int(created["id"].(float64))

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p1/schedules/"+strconv.Itoa(sid), map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["cron_expr"] != "0 9 * * *" {
		t.Fatalf("cron_expr = %v, want '0 9 * * *'", resp["cron_expr"])
	}
}

func TestSchedulePatch_DisableUnregisters(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	createRR := doRequest(t, router, http.MethodPost, "/api/projects/p1/schedules", map[string]any{
		"platform": "reddit", "cron_expr": "0 9 * * *",
	})
	var created map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	sid := int(created["id"].(float64))

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p1/schedules/"+strconv.Itoa(sid), map[string]any{
		"enabled": false,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["enabled"] != float64(0) {
		t.Fatalf("enabled = %v, want 0", resp["enabled"])
	}
}

func TestSchedulePatch_MissingSchedule_Returns404(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p1/schedules/999", map[string]any{"cron_expr": "0 8 * * *"})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Schedule not found" {
		t.Fatalf("error = %q, want 'Schedule not found'", resp["error"])
	}
}

func TestScheduleDelete_Returns204(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	createRR := doRequest(t, router, http.MethodPost, "/api/projects/p1/schedules", map[string]any{
		"platform": "reddit", "cron_expr": "0 9 * * *",
	})
	var created map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	sid := int(created["id"].(float64))

	rr := doRequest(t, router, http.MethodDelete, "/api/projects/p1/schedules/"+strconv.Itoa(sid), nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body = %s", rr.Code, rr.Body.String())
	}
}

func TestScheduleDelete_MissingSchedule_Returns404(t *testing.T) {
	router := newScheduleRouterWithProject(t)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"id": "p1", "name": "Test", "mode": "research",
	})
	rr := doRequest(t, router, http.MethodDelete, "/api/projects/p1/schedules/999", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Schedule not found" {
		t.Fatalf("error = %q, want 'Schedule not found'", resp["error"])
	}
}
