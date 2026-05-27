package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// queryReviewFakeAIProvider is a test-only AI provider that returns a fixed result or error.
// Use name "copilot" so the AI service model dispatcher can resolve it.
type queryReviewFakeAIProvider struct {
	name   string
	result string
	err    error
}

func (f *queryReviewFakeAIProvider) Name() string         { return f.name }
func (f *queryReviewFakeAIProvider) Available() bool      { return true }
func (f *queryReviewFakeAIProvider) Generate(_ context.Context, _, _, _ string, _ time.Duration) (string, error) {
	return f.result, f.err
}

// newQueryReviewRouter builds a test HTTP router with all query review job routes mounted.
// It returns both the handler and the repository for direct DB assertions.
// aiResult is returned by the fake "copilot" provider; use "" with nil err for an empty
// but valid response (the service wraps it in a SuggestResponse).
func newQueryReviewRouter(t *testing.T, aiResult string, aiErr error) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	// Name must be "copilot" so the AI service can match it to the model catalog.
	aiSvc := ai.NewServiceWithProviders([]ai.Provider{&queryReviewFakeAIProvider{name: "copilot", result: aiResult, err: aiErr}})
	querySvc := services.NewQueryService(repo, aiSvc)
	projectSvc := services.NewProjectService(repo, entitlements.RuntimeModeSelfHosted, entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil))
	r := chi.NewRouter()
	handlers.MountProjectRoutes(r, projectSvc)
	handlers.MountQueryRoutes(r, querySvc)
	handlers.MountQueryReviewJobRoutes(r, repo, querySvc)
	return r, repo
}

// waitForQueryReviewStatus polls the GET single-job endpoint until the job reaches the
// expected status or a 2-second deadline elapses.
func waitForQueryReviewStatus(t *testing.T, router http.Handler, projectID string, jobID int, want string) map[string]any {
	t.Helper()
	path := "/api/projects/" + projectID + "/query-review-jobs/" + itoa(jobID)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rr := doRequest(t, router, http.MethodGet, path, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("get job status = %d body = %s", rr.Code, rr.Body.String())
		}
		var job map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
			t.Fatal(err)
		}
		if job["status"] == want {
			return job
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %d did not reach status %s within deadline", jobID, want)
	return nil
}

// TestQueryReviewJob_CreateSuggestCompletesAndPersistsResult verifies that POSTing a
// suggest job returns 202, the job starts as "running", and the background goroutine
// completes it and persists the AI result.
func TestQueryReviewJob_CreateSuggestCompletesAndPersistsResult(t *testing.T) {
	aiJSON := `[{"platform":"google","query_url":"founder burnout","angle":"burnout"}]`
	// The QueryService.Suggest wraps the raw AI output in a SuggestResponse.
	// We rely on the service to parse and re-encode — our fake provider returns raw array
	// which the service wraps under "suggestions".
	router, _ := newQueryReviewRouter(t, aiJSON, nil)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/query-review-jobs", map[string]any{
		"kind":       "suggest",
		"refinement": "focus google",
	})
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body = %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created["status"] != "running" {
		t.Fatalf("initial status = %v, want running", created["status"])
	}
	if created["kind"] != "suggest" {
		t.Fatalf("kind = %v, want suggest", created["kind"])
	}

	job := waitForQueryReviewStatus(t, router, "p1", int(created["id"].(float64)), "completed")
	if job["result"] == nil {
		t.Fatalf("completed job has no result: %v", job)
	}
}

// TestQueryReviewJob_CreateRefineCompletesAndPersistsResult verifies that a refine job
// starts running, the background goroutine completes it, and the result is persisted.
func TestQueryReviewJob_CreateRefineCompletesAndPersistsResult(t *testing.T) {
	// The QueryService.Refine expects the AI to return a JSON object with "summary" and
	// "recommendations". Feed it valid JSON so the service can parse it.
	aiJSON := `{"summary":"Improve set","recommendations":[{"type":"add","reason":"more specific","sources":["current_queries"],"query":{"platform":"bluesky","query_url":"founder burnout","angle":"burnout"}}]}`
	router, _ := newQueryReviewRouter(t, aiJSON, nil)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})
	// Add at least one query so Refine has context to work with.
	doRequest(t, router, http.MethodPost, "/api/projects/p1/queries", map[string]any{
		"platform":  "google",
		"query_url": "startup burnout",
		"angle":     "burnout",
	})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/query-review-jobs", map[string]any{"kind": "refine"})
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body = %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	job := waitForQueryReviewStatus(t, router, "p1", int(created["id"].(float64)), "completed")
	if job["result"] == nil {
		t.Fatalf("completed refine job has no result: %v", job)
	}
}

// TestQueryReviewJob_FailedGenerationPersistsError verifies that when a job is marked
// failed (e.g. by the background worker or startup cleanup), the error field is persisted
// and the job is visible in the GET single-job endpoint.
func TestQueryReviewJob_FailedGenerationPersistsError(t *testing.T) {
	router, repo := newQueryReviewRouter(t, `[]`, nil)
	ctx := context.Background()

	// Set up project and a running job via HTTP.
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})
	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/query-review-jobs", map[string]any{"kind": "suggest"})
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body = %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	jobID := int64(created["id"].(float64))

	// Wait for the background goroutine to settle (it completes with notice in test mode).
	waitForQueryReviewStatus(t, router, "p1", int(jobID), "completed")

	// Simulate a failure being persisted (e.g. by a subsequent goroutine overwrite or
	// by directly exercising FailQueryReviewJob on a fresh running job).
	_, err := repo.DB.ExecContext(ctx,
		`INSERT INTO query_review_jobs (project_id, kind, status, started_at) VALUES ('p1', 'suggest', 'running', datetime('now'))`,
	)
	if err != nil {
		t.Fatal(err)
	}
	var failedID int64
	if err := repo.DB.QueryRowContext(ctx, `SELECT id FROM query_review_jobs WHERE status='running' AND project_id='p1'`).Scan(&failedID); err != nil {
		t.Fatal(err)
	}

	failed, err := repo.FailQueryReviewJob(ctx, "p1", failedID, "AI provider exploded during processing")
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != "failed" {
		t.Fatalf("status = %q, want failed", failed.Status)
	}
	if failed.Error == nil || *failed.Error == "" {
		t.Fatalf("expected error to be set, got nil/empty")
	}

	// Verify via HTTP GET that the error is persisted.
	getRR := doRequest(t, router, http.MethodGet, "/api/projects/p1/query-review-jobs/"+itoa(int(failedID)), nil)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get status = %d body = %s", getRR.Code, getRR.Body.String())
	}
	var job map[string]any
	if err := json.Unmarshal(getRR.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if job["status"] != "failed" {
		t.Fatalf("job status = %v, want failed", job["status"])
	}
	if job["error"] == nil || job["error"] == "" {
		t.Fatalf("expected error field on failed job, got %v", job["error"])
	}
}

// TestQueryReviewJob_ListAndReviewedResolution checks:
//  1. A completed job appears in the project list.
//  2. After marking it reviewed (denied), it no longer appears in the list.
func TestQueryReviewJob_ListAndReviewedResolution(t *testing.T) {
	router, _ := newQueryReviewRouter(t, `[]`, nil)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "Test", "mode": "research"})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/query-review-jobs", map[string]any{"kind": "suggest"})
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body = %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	jobID := int(created["id"].(float64))

	// Wait for completion so the job is reviewable.
	waitForQueryReviewStatus(t, router, "p1", jobID, "completed")

	// List should contain the completed, unreviewed job.
	listRR := doRequest(t, router, http.MethodGet, "/api/projects/p1/query-review-jobs", nil)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRR.Code, listRR.Body.String())
	}
	var list []map[string]any
	if err := json.Unmarshal(listRR.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}

	// Mark as reviewed with resolution=denied.
	reviewRR := doRequest(t, router, http.MethodPost,
		"/api/projects/p1/query-review-jobs/"+itoa(jobID)+"/reviewed",
		map[string]any{"resolution": "denied"},
	)
	if reviewRR.Code != http.StatusOK {
		t.Fatalf("review status = %d body = %s", reviewRR.Code, reviewRR.Body.String())
	}

	// List should now be empty because the job has been reviewed.
	listRR = doRequest(t, router, http.MethodGet, "/api/projects/p1/query-review-jobs", nil)
	if err := json.Unmarshal(listRR.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list len after reviewed = %d, want 0", len(list))
	}
}

// TestQueryReviewJob_ProjectIsolation verifies that a job created under project p1
// cannot be fetched or reviewed via project p2's routes.
func TestQueryReviewJob_ProjectIsolation(t *testing.T) {
	router, _ := newQueryReviewRouter(t, `[]`, nil)
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p1", "name": "One", "mode": "research"})
	doRequest(t, router, http.MethodPost, "/api/projects", map[string]any{"id": "p2", "name": "Two", "mode": "research"})

	rr := doRequest(t, router, http.MethodPost, "/api/projects/p1/query-review-jobs", map[string]any{"kind": "suggest"})
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body = %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	jobID := int(created["id"].(float64))

	// GET /api/projects/p2/query-review-jobs/:jobId should return 404.
	wrongGet := doRequest(t, router, http.MethodGet, "/api/projects/p2/query-review-jobs/"+itoa(jobID), nil)
	if wrongGet.Code != http.StatusNotFound {
		t.Fatalf("wrong-project GET status = %d, want 404", wrongGet.Code)
	}

	// Wait for job to complete so we can attempt the reviewed endpoint.
	waitForQueryReviewStatus(t, router, "p1", jobID, "completed")

	// POST /api/projects/p2/query-review-jobs/:jobId/reviewed should return 404.
	wrongReview := doRequest(t, router, http.MethodPost,
		"/api/projects/p2/query-review-jobs/"+itoa(jobID)+"/reviewed",
		map[string]any{"resolution": "denied"},
	)
	if wrongReview.Code != http.StatusNotFound {
		t.Fatalf("wrong-project review status = %d, want 404", wrongReview.Code)
	}
}

// TestQueryReviewJob_StaleCleanup verifies that MarkStaleQueryReviewJobsFailed transitions
// jobs whose started_at is older than 15 minutes to "failed" status.
func TestQueryReviewJob_StaleCleanup(t *testing.T) {
	_, repo := newQueryReviewRouter(t, `[]`, nil)
	ctx := context.Background()

	// Insert a project and a stale running job directly via the DB.
	_, err := repo.DB.ExecContext(ctx, `INSERT INTO projects (id, name, mode) VALUES ('p1', 'Test', 'research')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.DB.ExecContext(ctx,
		`INSERT INTO query_review_jobs (project_id, kind, status, started_at)
		 VALUES ('p1', 'suggest', 'running', datetime('now', '-20 minutes'))`,
	)
	if err != nil {
		t.Fatal(err)
	}

	n, err := repo.MarkStaleQueryReviewJobsFailed(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("rows affected = %d, want 1", n)
	}

	jobs, err := repo.ListQueryReviewJobs(ctx, "p1", 20)
	if err != nil {
		t.Fatal(err)
	}
	// The failed job has no reviewed_at, so it should still appear in the list.
	if len(jobs) != 1 {
		t.Fatalf("jobs len = %d, want 1", len(jobs))
	}
	if jobs[0].Status != "failed" {
		t.Fatalf("job status = %q, want failed", jobs[0].Status)
	}
}
