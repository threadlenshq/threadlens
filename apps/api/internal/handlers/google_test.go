package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newGoogleRouter(t *testing.T) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	svc := services.NewGoogleService(repo)
	r := chi.NewRouter()
	handlers.MountGoogleRoutes(r, svc)
	return r, repo
}

// seedGoogleData inserts scout_run, google_report, keyword_summaries, and google_results.
func seedGoogleData(t *testing.T, repo *repository.Repository, projectID string) (runID, reportID int64) {
	t.Helper()

	// Insert project
	_, err := repo.DB.Exec(
		`INSERT INTO projects (id, name, mode) VALUES (?, ?, 'research')`,
		projectID, projectID+"-name",
	)
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	// Insert scout run
	res, err := repo.DB.Exec(
		`INSERT INTO scout_runs (project_id, platform, status, posts_checked, posts_found, step, error, warnings)
		 VALUES (?, 'google', 'completed', 10, 5, 'fetch', NULL, 'warn1')`,
		projectID,
	)
	if err != nil {
		t.Fatalf("seed scout_run: %v", err)
	}
	runID, _ = res.LastInsertId()

	// Insert google_report
	res, err = repo.DB.Exec(
		`INSERT INTO google_reports (run_id, project_id, executive_summary_json, keyword_summary_json, opportunities_json, risks_json, next_actions_json)
		 VALUES (?, ?, '{"title":"exec"}', '[{"kw":"foo"}]', '["opp1"]', '["risk1"]', '["act1"]')`,
		runID, projectID,
	)
	if err != nil {
		t.Fatalf("seed google_report: %v", err)
	}
	reportID, _ = res.LastInsertId()

	// Insert keyword summary
	_, err = repo.DB.Exec(
		`INSERT INTO google_keyword_summaries
		 (run_id, project_id, root_keyword, total_results, relevant_results, outreach_candidates,
		  result_types_json, content_types_json, intent_types_json, recommendation_json)
		 VALUES (?, ?, 'test keyword', 5, 3, 1, '{"blog":2}', '{"article":3}', '{"informational":3}', '{"action":"publish"}')`,
		runID, projectID,
	)
	if err != nil {
		t.Fatalf("seed keyword summary: %v", err)
	}

	// Insert google results: 4 rows with varied fit/score/outreach
	type resultSeed struct {
		relevanceFit      string
		relevanceScore    float64
		confidenceScore   float64
		outreachCandidate int
		opportunityTypes  string
		actionRec         string
	}
	seeds := []resultSeed{
		{"direct_fit", 0.9, 0.8, 1, `["seo_opportunity"]`, "Optimize for search"},
		{"direct_fit", 0.7, 0.6, 0, `["competitor_weakness"]`, "comparison analysis"},
		{"partial_fit", 0.5, 0.7, 1, `[]`, "outreach"},
		{"weak_fit", 0.3, 0.4, 0, `[]`, "ignore"},
	}
	for i, s := range seeds {
		title := fmt.Sprintf("Title %c", rune('A'+i))
		_, err = repo.DB.Exec(
			`INSERT INTO google_results
			 (run_id, project_id, root_keyword, query, title, url, display_url, snippet,
			  rank, result_type, domain, author, content_type, intent_type, relevance_fit,
			  relevance_score, confidence_score, opportunity_types, keepgoing_fit_reasons,
			  disqualifiers, summary, action_recommendation, outreach_candidate, canonical_url, content_hash)
			 VALUES (?, ?, 'kw', 'q', ?, 'http://example.com', 'example.com', 'snippet',
			  ?, 'organic', 'example.com', 'author', 'blog', 'informational', ?,
			  ?, ?, ?, '[]', '[]', 'summary', ?, ?, 'http://example.com', 'hash')`,
			runID, projectID, title,
			i+1, s.relevanceFit, s.relevanceScore, s.confidenceScore,
			s.opportunityTypes, s.actionRec, s.outreachCandidate,
		)
		if err != nil {
			t.Fatalf("seed google_result %d: %v", i, err)
		}
	}

	return runID, reportID
}

func googleReportURL(projectID string, reportID int64, suffix string) string {
	return fmt.Sprintf("/api/projects/%s/google/reports/%d%s", projectID, reportID, suffix)
}

func TestGoogleReports_List(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "")
	r, repo := newGoogleRouter(t)
	_, _ = seedGoogleData(t, repo, "gp1")

	rr := doRequest(t, r, http.MethodGet, "/api/projects/gp1/google/reports", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var result []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 report, got %d", len(result))
	}
	// JSON columns should be parsed (not raw strings)
	if _, ok := result[0]["executive_summary"].(map[string]any); !ok {
		t.Fatalf("executive_summary should be object, got %T", result[0]["executive_summary"])
	}
	if _, ok := result[0]["opportunities"].([]any); !ok {
		t.Fatalf("opportunities should be array, got %T", result[0]["opportunities"])
	}
	if _, ok := result[0]["run"].(map[string]any); !ok {
		t.Fatalf("run should be object, got %T", result[0]["run"])
	}
}

func TestGoogleReports_Latest_NotFound(t *testing.T) {
	r, _ := newGoogleRouter(t)
	rr := doRequest(t, r, http.MethodGet, "/api/projects/noproj/google/reports/latest", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rr.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "Google report not found" {
		t.Fatalf("error=%q want 'Google report not found'", resp["error"])
	}
}

func TestGoogleReports_GetByID_NotFound(t *testing.T) {
	r, _ := newGoogleRouter(t)
	rr := doRequest(t, r, http.MethodGet, "/api/projects/noproj/google/reports/999", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rr.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "Google report not found" {
		t.Fatalf("error=%q want 'Google report not found'", resp["error"])
	}
}

func TestGoogleReports_GetByID_EmbedRunFields(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "")
	r, repo := newGoogleRouter(t)
	_, reportID := seedGoogleData(t, repo, "gp2")

	rr := doRequest(t, r, http.MethodGet, googleReportURL("gp2", reportID, ""), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	runObj, ok := result["run"].(map[string]any)
	if !ok {
		t.Fatalf("run should be object, got %T", result["run"])
	}
	// step, error, warnings must be present
	for _, field := range []string{"step", "error", "warnings"} {
		if _, has := runObj[field]; !has {
			t.Fatalf("run missing %q field", field)
		}
	}
}

func TestGoogleReports_Keywords_ParsesJSON(t *testing.T) {
	r, repo := newGoogleRouter(t)
	_, reportID := seedGoogleData(t, repo, "gp3")

	rr := doRequest(t, r, http.MethodGet, googleReportURL("gp3", reportID, "/keywords"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var result []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("expected at least 1 keyword summary")
	}
	// JSON columns should be parsed objects
	for _, field := range []string{"result_types", "content_types", "intent_types", "recommendation"} {
		if _, ok := result[0][field].(map[string]any); !ok {
			t.Fatalf("%s should be object, got %T: %v", field, result[0][field], result[0][field])
		}
	}
}

func TestGoogleReports_Results_InvalidMode(t *testing.T) {
	r, repo := newGoogleRouter(t)
	_, reportID := seedGoogleData(t, repo, "gp4")

	rr := doRequest(t, r, http.MethodGet, googleReportURL("gp4", reportID, "/results?mode=invalid"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
	var resp map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["error"] != "mode must be one of seo, messaging, competitor, outreach" {
		t.Fatalf("error=%q", resp["error"])
	}
}

func TestGoogleReports_Results_SEO(t *testing.T) {
	r, repo := newGoogleRouter(t)
	_, reportID := seedGoogleData(t, repo, "gp5")

	rr := doRequest(t, r, http.MethodGet, googleReportURL("gp5", reportID, "/results?mode=seo&limit=10"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["mode"] != "seo" {
		t.Fatalf("mode=%v want seo", resp["mode"])
	}
	results, ok := resp["results"].([]any)
	if !ok {
		t.Fatalf("results should be array, got %T", resp["results"])
	}
	// Only direct_fit results: 2 of 4 seeded
	if len(results) != 2 {
		t.Fatalf("expected 2 direct_fit results, got %d", len(results))
	}
}

func TestGoogleReports_Results_Messaging(t *testing.T) {
	r, repo := newGoogleRouter(t)
	_, reportID := seedGoogleData(t, repo, "gp6")

	rr := doRequest(t, r, http.MethodGet, googleReportURL("gp6", reportID, "/results?mode=messaging"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	results := resp["results"].([]any)
	// weak_fit excluded -> 3 results
	if len(results) != 3 {
		t.Fatalf("expected 3 messaging results, got %d", len(results))
	}
}

func TestGoogleReports_Results_Outreach(t *testing.T) {
	r, repo := newGoogleRouter(t)
	_, reportID := seedGoogleData(t, repo, "gp7")

	rr := doRequest(t, r, http.MethodGet, googleReportURL("gp7", reportID, "/results?mode=outreach"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var resp map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	results := resp["results"].([]any)
	// outreach_candidate=1: 2 results
	if len(results) != 2 {
		t.Fatalf("expected 2 outreach results, got %d", len(results))
	}
}

func TestGoogleResultsExcludeFilteredRows(t *testing.T) {
	r, repo := newGoogleRouter(t)
	projectID := "gp_filter_test"

	// Insert project
	_, err := repo.DB.Exec(
		`INSERT INTO projects (id, name, mode) VALUES (?, ?, 'research')`,
		projectID, projectID+"-name",
	)
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	// Insert scout run
	res, err := repo.DB.Exec(
		`INSERT INTO scout_runs (project_id, platform, status, posts_checked, posts_found, step, error, warnings)
		 VALUES (?, 'google', 'completed', 2, 2, 'fetch', NULL, NULL)`,
		projectID,
	)
	if err != nil {
		t.Fatalf("seed scout_run: %v", err)
	}
	runID, _ := res.LastInsertId()

	// Insert google_report
	res, err = repo.DB.Exec(
		`INSERT INTO google_reports (run_id, project_id, executive_summary_json, keyword_summary_json, opportunities_json, risks_json, next_actions_json)
		 VALUES (?, ?, '{}', '[]', '[]', '[]', '[]')`,
		runID, projectID,
	)
	if err != nil {
		t.Fatalf("seed google_report: %v", err)
	}
	reportID, _ := res.LastInsertId()

	// Insert visible result
	_, err = repo.DB.Exec(
		`INSERT INTO google_results
		 (run_id, project_id, root_keyword, query, title, url, display_url, snippet,
		  rank, result_type, domain, author, content_type, intent_type, relevance_fit,
		  relevance_score, confidence_score, opportunity_types, keepgoing_fit_reasons,
		  disqualifiers, summary, action_recommendation, outreach_candidate, canonical_url, content_hash,
		  filter_state, filter_source, filter_reasons_json, source_identity_json)
		 VALUES (?, ?, 'kw', 'q', 'Visible Title', 'http://example.com/v', 'example.com', 'snippet',
		  1, 'organic', 'example.com', 'author', 'blog', 'informational', 'direct_fit',
		  0.9, 0.8, '[]', '[]', '[]', 'summary', 'action', 1, 'http://example.com/v', 'hash1',
		  'visible', 'none', '[]', '{}')`,
		runID, projectID,
	)
	if err != nil {
		t.Fatalf("seed visible google_result: %v", err)
	}

	// Insert filtered result
	_, err = repo.DB.Exec(
		`INSERT INTO google_results
		 (run_id, project_id, root_keyword, query, title, url, display_url, snippet,
		  rank, result_type, domain, author, content_type, intent_type, relevance_fit,
		  relevance_score, confidence_score, opportunity_types, keepgoing_fit_reasons,
		  disqualifiers, summary, action_recommendation, outreach_candidate, canonical_url, content_hash,
		  filter_state, filter_source, filter_reasons_json, source_identity_json)
		 VALUES (?, ?, 'kw', 'q', 'Filtered Title', 'http://example.com/f', 'example.com', 'snippet',
		  2, 'organic', 'example.com', 'author', 'blog', 'informational', 'direct_fit',
		  0.8, 0.7, '[]', '[]', '[]', 'summary', 'action', 1, 'http://example.com/f', 'hash2',
		  'filtered', 'rules', '["spam"]', '{}')`,
		runID, projectID,
	)
	if err != nil {
		t.Fatalf("seed filtered google_result: %v", err)
	}

	rr := doRequest(t, r, http.MethodGet, googleReportURL(projectID, reportID, "/results?mode=seo"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	results, ok := resp["results"].([]any)
	if !ok {
		t.Fatalf("results should be array, got %T", resp["results"])
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 visible result, got %d", len(results))
	}
	first := results[0].(map[string]any)
	if first["title"] != "Visible Title" {
		t.Fatalf("expected title 'Visible Title', got %v", first["title"])
	}
}
