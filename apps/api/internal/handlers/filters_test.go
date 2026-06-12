package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newFilterRouter(t *testing.T) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	classifier := pipeline.NewFilterClassifier(repo, nil)
	r := chi.NewRouter()
	handlers.MountFilterRoutes(r, repo, classifier, nil)
	return r, repo
}

func seedFilteredPost(t *testing.T, repo *repository.Repository, projectID, postID string) {
	t.Helper()
	_, err := repo.DB.Exec(
		`INSERT INTO posts (id, project_id, platform, title, body, author, url, post_score, final_score,
			engagement_type, status, filter_state, filter_reason, filter_reasons_json, filter_explanation,
			filter_source, filtered_at, source_identity_json, found_at, scouted_at)
		 VALUES (?, ?, 'reddit', 'Spam Post', 'body', 'spammer', 'http://example.com', 3.0, 3.0,
		 	'karma', 'new', 'filtered', 'spam', '["spam"]', 'promotional language',
		 	'rules', datetime('now'), '{"reddit_author":"spammer"}', datetime('now'), datetime('now'))`,
		postID, projectID,
	)
	if err != nil {
		t.Fatalf("seed filtered post: %v", err)
	}
}

func TestFiltersListReturnsFilteredFindings(t *testing.T) {
	router, repo := newFilterRouter(t)
	seedProject(t, repo, "proj1")
	seedFilteredPost(t, repo, "proj1", "post-filtered-1")

	rr := doRequest(t, router, http.MethodGet, "/api/projects/proj1/filters/findings", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body = %s", rr.Code, rr.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	items, ok := result["items"].([]any)
	if !ok {
		t.Fatalf("expected items array; body = %s", rr.Body.String())
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 filtered finding, got %d", len(items))
	}
}

func TestFiltersRecoverRestoreOnlyPreservesPostStatusAndScore(t *testing.T) {
	router, repo := newFilterRouter(t)
	seedProject(t, repo, "proj2")
	seedFilteredPost(t, repo, "proj2", "post-to-restore")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj2/filters/findings/recover", map[string]any{
		"finding_type": "post",
		"id":           "post-to-restore",
		"mode":         "restore_visibility",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body = %s", rr.Code, rr.Body.String())
	}

	// Verify post score and status are preserved, filter_state is visible
	post, err := repo.GetPost(t.Context(), "proj2", "post-to-restore")
	if err != nil {
		t.Fatal(err)
	}
	if post.FilterState != "visible" {
		t.Fatalf("filter_state = %q, want visible", post.FilterState)
	}
	if post.Status != "new" {
		t.Fatalf("status = %q, want new (unchanged)", post.Status)
	}
	if post.FinalScore != 3.0 {
		t.Fatalf("final_score = %v, want 3.0 (unchanged)", post.FinalScore)
	}
}

func TestFiltersRecoverRestoreAndTrustCreatesTrustRecord(t *testing.T) {
	router, repo := newFilterRouter(t)
	seedProject(t, repo, "proj3")
	seedFilteredPost(t, repo, "proj3", "post-trust-restore")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj3/filters/findings/recover", map[string]any{
		"finding_type": "post",
		"id":           "post-trust-restore",
		"mode":         "restore_and_trust",
		"trust": map[string]any{
			"platform":    "reddit",
			"trust_type":  "source",
			"source_kind": "reddit_author",
			"source_key":  "spammer",
		},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d want 200; body = %s", rr.Code, rr.Body.String())
	}

	// Verify trust record was created
	records, err := repo.ListTrustRecords(t.Context(), "proj3")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 trust record, got %d", len(records))
	}
	if records[0].SourceKey != "spammer" {
		t.Fatalf("source_key = %q, want spammer", records[0].SourceKey)
	}
}

func TestFiltersRecoverRejectsCrossProjectFinding(t *testing.T) {
	router, repo := newFilterRouter(t)
	seedProject(t, repo, "projA")
	seedProject(t, repo, "projB")
	seedFilteredPost(t, repo, "projA", "post-projA")

	// Attempt to recover projA's post via projB's route
	rr := doRequest(t, router, http.MethodPost, "/api/projects/projB/filters/findings/recover", map[string]any{
		"finding_type": "post",
		"id":           "post-projA",
		"mode":         "restore_visibility",
	})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d want 404; body = %s", rr.Code, rr.Body.String())
	}
}

func TestFiltersCreateJobReturnsAccepted(t *testing.T) {
	router, repo := newFilterRouter(t)
	seedProject(t, repo, "proj4")
	seedPost(t, repo, "proj4", "visible-post-1")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj4/filters/jobs", map[string]any{
		"requested_scope": "selected_visible_posts",
		"targets": []map[string]any{
			{"finding_type": "post", "id": "visible-post-1"},
		},
	})
	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d want 202; body = %s", rr.Code, rr.Body.String())
	}
	var job map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if job["status"] != "running" {
		t.Fatalf("status = %v, want running", job["status"])
	}
}
