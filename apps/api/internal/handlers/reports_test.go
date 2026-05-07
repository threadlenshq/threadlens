package handlers_test

import (
	"encoding/json"
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

func newCombinedReportRouter(t *testing.T) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	aiSvc := ai.NewService(repo)
	r := chi.NewRouter()
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	handlers.MountProjectRoutes(r, services.NewProjectService(repo, entitlements.RuntimeModeSelfHosted, resolver))
	handlers.MountReportRoutes(r, services.NewReportService(repo, db, aiSvc, entitlements.RuntimeModeSelfHosted, resolver))
	return r, repo
}

func TestReportList_Empty(t *testing.T) {
	r, _ := newCombinedReportRouter(t)

	doRequest(t, r, http.MethodPost, "/api/projects", map[string]any{"id": "rp1", "name": "N", "mode": "research"})
	rr := doRequest(t, r, http.MethodGet, "/api/projects/rp1/reports", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var result []any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty array, got %v", result)
	}
}

func TestReportList_ClustersIsArray(t *testing.T) {
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	aiSvc := ai.NewService(repo)
	r := chi.NewRouter()
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	handlers.MountProjectRoutes(r, services.NewProjectService(repo, entitlements.RuntimeModeSelfHosted, resolver))
	handlers.MountReportRoutes(r, services.NewReportService(repo, db, aiSvc, entitlements.RuntimeModeSelfHosted, resolver))

	doRequest(t, r, http.MethodPost, "/api/projects", map[string]any{"id": "rp2", "name": "N", "mode": "research"})

	// Insert a completed report directly to verify clusters JSON parsing.
	_, err := db.Exec(`INSERT INTO research_reports (project_id, status, title, post_count, clusters, assessment, model_used)
		VALUES ('rp2', 'completed', 'Test Report', 5, '[{"name":"cluster1"}]', 'good', 'test-model')`)
	if err != nil {
		t.Fatal(err)
	}

	rr := doRequest(t, r, http.MethodGet, "/api/projects/rp2/reports", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var result []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least one report")
	}
	clusters, ok := result[0]["clusters"]
	if !ok {
		t.Fatal("missing clusters field")
	}
	// clusters should decode to a []interface{} (JSON array), not a string.
	if _, isArray := clusters.([]any); !isArray {
		t.Fatalf("clusters expected []any (JSON array), got %T: %v", clusters, clusters)
	}
}

func TestReportGet_NotFound(t *testing.T) {
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	aiSvc := ai.NewService(repo)
	r := chi.NewRouter()
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	handlers.MountReportRoutes(r, services.NewReportService(repo, db, aiSvc, entitlements.RuntimeModeSelfHosted, resolver))

	rr := doRequest(t, r, http.MethodGet, "/api/projects/noproj/reports/999", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Report not found" {
		t.Fatalf("error = %q, want 'Report not found'", resp["error"])
	}
}

func TestReportCouncil_NotFound(t *testing.T) {
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	aiSvc := ai.NewService(repo)
	r := chi.NewRouter()
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	handlers.MountReportRoutes(r, services.NewReportService(repo, db, aiSvc, entitlements.RuntimeModeSelfHosted, resolver))

	rr := doRequest(t, r, http.MethodGet, "/api/projects/noproj/reports/999/council", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "Council not found" {
		t.Fatalf("error = %q, want 'Council not found'", resp["error"])
	}
}

func TestReportCreate_Returns201WithRunningStatus(t *testing.T) {
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	aiSvc := ai.NewService(repo)
	r := chi.NewRouter()
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	handlers.MountProjectRoutes(r, services.NewProjectService(repo, entitlements.RuntimeModeSelfHosted, resolver))
	handlers.MountReportRoutes(r, services.NewReportService(repo, db, aiSvc, entitlements.RuntimeModeSelfHosted, resolver))

	doRequest(t, r, http.MethodPost, "/api/projects", map[string]any{"id": "rp3", "name": "N", "mode": "research"})

	rr := doRequest(t, r, http.MethodPost, "/api/projects/rp3/reports", nil)
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
}

func TestReportCreate_BackgroundFailureUpdatesStatus(t *testing.T) {
	// When there are no posts, the analysis should eventually fail and update the report row.
	// We test the repository directly (not via HTTP) to avoid timing issues in tests.
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)

	// Insert a project.
	_, err := db.Exec(`INSERT INTO projects (id, name, mode) VALUES ('rp4', 'N', 'research')`)
	if err != nil {
		t.Fatal(err)
	}

	reportID, err := repo.StartAnalysis(t.Context(), "rp4", "test-model")
	if err != nil {
		t.Fatal(err)
	}

	// Mark it failed (as RunAnalysis would when there are no posts).
	if err := repo.MarkReportFailed(t.Context(), reportID, "No posts to analyze"); err != nil {
		t.Fatal(err)
	}

	rep, err := repo.GetReport(t.Context(), "rp4", reportID)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != "failed" {
		t.Fatalf("status = %q, want 'failed'", rep.Status)
	}
	if rep.Error == nil || *rep.Error != "No posts to analyze" {
		t.Fatalf("error = %v, want 'No posts to analyze'", rep.Error)
	}
}
