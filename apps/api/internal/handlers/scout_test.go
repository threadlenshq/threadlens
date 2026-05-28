package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newScoutRouter(t *testing.T) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	aiSvc := ai.NewService(repo)
	runner := pipeline.NewRunner(repo, aiSvc)
	r := chi.NewRouter()
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	handlers.MountProjectRoutes(r, services.NewProjectService(repo, entitlements.RuntimeModeSelfHosted, resolver))
	handlers.MountScoutRoutes(r, services.NewScoutService(repo, runner, entitlements.RuntimeModeSelfHosted, resolver))
	return r, repo
}

func TestScout_InvalidPlatform(t *testing.T) {
	r, _ := newScoutRouter(t)
	doRequest(t, r, http.MethodPost, "/api/projects", map[string]any{"id": "sp1", "name": "Test", "mode": "research"})

	rr := doRequest(t, r, http.MethodPost, "/api/projects/sp1/scout?platform=invalid", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	want := `platform must be "reddit", "bluesky", or "google"`
	if resp["error"] != want {
		t.Fatalf("error = %q, want %q", resp["error"], want)
	}
}

func TestScout_MissingProject(t *testing.T) {
	r, _ := newScoutRouter(t)

	rr := doRequest(t, r, http.MethodPost, "/api/projects/noproject/scout?platform=reddit", nil)
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

func TestScout_StartAsync_ReturnsRunning(t *testing.T) {
	r, _ := newScoutRouter(t)
	doRequest(t, r, http.MethodPost, "/api/projects", map[string]any{"id": "sp2", "name": "Test", "mode": "research"})

	rr := doRequest(t, r, http.MethodPost, "/api/projects/sp2/scout?platform=reddit", nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "running" {
		t.Fatalf("status = %v, want 'running'", resp["status"])
	}
	if resp["runId"] == nil {
		t.Fatal("expected runId in response")
	}
}

func TestScout_GoogleWithoutParallelKeyDeniedBeforeRunCreation(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "")
	r, repo := newScoutRouter(t)
	doRequest(t, r, http.MethodPost, "/api/projects", map[string]any{"id": "sp-google-locked", "name": "Test", "mode": "research"})

	rr := doRequest(t, r, http.MethodPost, "/api/projects/sp-google-locked/scout?platform=google", nil)
	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	want := "capability denied: core.scout.run.google: capability_not_granted"
	if resp["error"] != want {
		t.Fatalf("error = %q, want %q", resp["error"], want)
	}

	// Confirm zero rows in scout_runs for this project/platform.
	var count int
	err := repo.DB.QueryRow(
		`SELECT COUNT(*) FROM scout_runs WHERE project_id = 'sp-google-locked' AND platform = 'google'`,
	).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 scout_runs rows, got %d", count)
	}
}

func TestScout_ListRuns_ReturnsLatest20(t *testing.T) {
	r, _ := newScoutRouter(t)
	doRequest(t, r, http.MethodPost, "/api/projects", map[string]any{"id": "sp3", "name": "Test", "mode": "research"})

	// Start a run.
	doRequest(t, r, http.MethodPost, "/api/projects/sp3/scout?platform=reddit", nil)

	rr := doRequest(t, r, http.MethodGet, "/api/projects/sp3/scout/runs", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var runs []any
	if err := json.Unmarshal(rr.Body.Bytes(), &runs); err != nil {
		t.Fatal(err)
	}
	if len(runs) == 0 {
		t.Fatal("expected at least one run")
	}
}

func TestScout_GetRun_NotFound(t *testing.T) {
	r, _ := newScoutRouter(t)

	rr := doRequest(t, r, http.MethodGet, "/api/projects/noproject/scout/runs/9999", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Run not found" {
		t.Fatalf("error = %q, want 'Run not found'", resp["error"])
	}
}

func TestScout_Cancel_UntrackedRunningRow_MarksFailedWithCancelled(t *testing.T) {
	_, repo := newScoutRouter(t)
	db := testingpkg.OpenTestDB(t)
	repo2 := repository.New(db)
	aiSvc := ai.NewService(repo2)
	runner := pipeline.NewRunner(repo2, aiSvc)
	r2 := chi.NewRouter()
	resolver2 := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	handlers.MountProjectRoutes(r2, services.NewProjectService(repo2, entitlements.RuntimeModeSelfHosted, resolver2))
	handlers.MountScoutRoutes(r2, services.NewScoutService(repo2, runner, entitlements.RuntimeModeSelfHosted, resolver2))
	_ = repo // unused; repo2 is the test target

	doRequest(t, r2, http.MethodPost, "/api/projects", map[string]any{"id": "sp4", "name": "Test", "mode": "research"})

	// Insert a running row directly (not tracked by runner).
	runID, err := repo2.CreateScoutRun(t.Context(), "sp4", "reddit")
	if err != nil {
		t.Fatal(err)
	}

	rr := doRequest(t, r2, http.MethodPost, "/api/projects/sp4/scout/runs/"+intStr(runID)+"/cancel", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true {
		t.Fatalf("ok = %v, want true", resp["ok"])
	}

	// Verify the DB row was updated.
	run, err := repo2.GetScoutRun(t.Context(), "sp4", runID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "failed" {
		t.Fatalf("status = %q, want 'failed'", run.Status)
	}
	if run.Error == nil || *run.Error != "Cancelled" {
		t.Fatalf("error = %v, want 'Cancelled'", run.Error)
	}
}

func intStr(i int64) string {
	return fmt.Sprintf("%d", i)
}
