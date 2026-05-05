package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// seedPerfDB inserts 1 project, 500 posts, 20 research reports, and 10 google reports.
func seedPerfDB(t *testing.T, db *sql.DB) {
	t.Helper()

	mustExec := func(query string, args ...any) sql.Result {
		t.Helper()
		res, err := db.Exec(query, args...)
		if err != nil {
			t.Fatalf("seedPerfDB exec error: %v\nquery: %s", err, query)
		}
		return res
	}

	mustExec(`INSERT INTO projects (id, name, mode) VALUES ('p1', 'Perf Project', 'research')`)

	for i := 1; i <= 500; i++ {
		mustExec(
			`INSERT INTO posts (id, project_id, platform, title, body, author, url, post_score, final_score, engagement_type, status, found_at, scouted_at)
			 VALUES (?, 'p1', 'reddit', ?, 'body', 'user', 'http://example.com', 5.0, 5.0, 'karma', 'new', datetime('now'), datetime('now'))`,
			fmt.Sprintf("post%d", i),
			fmt.Sprintf("Post %d", i),
		)
	}

	for i := 1; i <= 20; i++ {
		mustExec(
			`INSERT INTO research_reports (project_id, title, status, post_count, clusters, assessment, model_used)
			 VALUES ('p1', ?, 'completed', 5, '[]', 'assessment', 'test-model')`,
			fmt.Sprintf("Report %d", i),
		)
	}

	for i := 1; i <= 10; i++ {
		res := mustExec(
			`INSERT INTO scout_runs (project_id, platform, status, posts_checked, posts_found)
			 VALUES ('p1', 'google', 'completed', 10, 5)`,
		)
		runID, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("seedPerfDB: last insert id: %v", err)
		}
		mustExec(
			`INSERT INTO google_reports (run_id, project_id, executive_summary_json, keyword_summary_json, opportunities_json, risks_json, next_actions_json)
			 VALUES (?, 'p1', '{}', '[]', '[]', '[]', '[]')`,
			runID,
		)
	}
}

// TestPerformanceConcurrentRequests runs 50 concurrent GET /api/projects/p1/posts?page=1&limit=20
// and 50 concurrent GET /api/projects/p1/google/reports against an in-process httptest server.
// Each request must complete within 250 ms and return HTTP 200; no SQLite busy errors may occur.
func TestPerformanceConcurrentRequests(t *testing.T) {
	database := testhelpers.OpenTestDB(t)
	seedPerfDB(t, database)

	cfg := LoadConfig()
	a := New(cfg, database)
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)

	type result struct {
		duration time.Duration
		status   int
		err      error
	}

	runConcurrent := func(path string, n int) []result {
		results := make([]result, n)
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				start := time.Now()
				resp, err := http.Get(srv.URL + path) //nolint:noctx
				dur := time.Since(start)
				if err != nil {
					results[idx] = result{duration: dur, err: err}
					return
				}
				resp.Body.Close()
				results[idx] = result{duration: dur, status: resp.StatusCode}
			}(i)
		}
		wg.Wait()
		return results
	}

	const maxDuration = 250 * time.Millisecond
	const concurrency = 50

	// 50 concurrent posts requests
	for i, r := range runConcurrent("/api/projects/p1/posts?page=1&limit=20", concurrency) {
		if r.err != nil {
			t.Errorf("posts[%d] error: %v", i, r.err)
			continue
		}
		if r.status != http.StatusOK {
			t.Errorf("posts[%d] status = %d, want 200", i, r.status)
		}
		if r.duration > maxDuration {
			t.Errorf("posts[%d] duration = %v, want <= %v", i, r.duration, maxDuration)
		}
	}

	// 50 concurrent google reports requests
	for i, r := range runConcurrent("/api/projects/p1/google/reports", concurrency) {
		if r.err != nil {
			t.Errorf("google/reports[%d] error: %v", i, r.err)
			continue
		}
		if r.status != http.StatusOK {
			t.Errorf("google/reports[%d] status = %d, want 200", i, r.status)
		}
		if r.duration > maxDuration {
			t.Errorf("google/reports[%d] duration = %v, want <= %v", i, r.duration, maxDuration)
		}
	}
}
