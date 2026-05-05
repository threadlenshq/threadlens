package services_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// insertRunningRun inserts a scout_run with status='running' started at startedAt.
func insertRunningRun(t *testing.T, db *sql.DB, projectID string, startedAt time.Time) int64 {
	t.Helper()
	// Ensure a project exists.
	_, err := db.Exec(
		`INSERT OR IGNORE INTO projects (id, name, mode) VALUES (?, 'Test', 'research')`,
		projectID,
	)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	res, err := db.Exec(
		`INSERT INTO scout_runs (project_id, platform, started_at, status)
		 VALUES (?, 'reddit', ?, 'running')`,
		projectID, startedAt.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// TestStartupReconciliation_OrphanedRunsFailed verifies that running scout_runs
// older than 5 minutes are marked failed with "Server restarted".
func TestStartupReconciliation_OrphanedRunsFailed(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	staleTime := time.Now().Add(-10 * time.Minute)
	oldID := insertRunningRun(t, db, "proj-orphan", staleTime)

	if err := services.RunStartupTasks(ctx, repo, nil); err != nil {
		t.Fatalf("RunStartupTasks: %v", err)
	}

	var status, errMsg sql.NullString
	row := db.QueryRow(`SELECT status, error FROM scout_runs WHERE id = ?`, oldID)
	if err := row.Scan(&status, &errMsg); err != nil {
		t.Fatalf("scan run: %v", err)
	}

	if status.String != "failed" {
		t.Errorf("expected status=failed, got %q", status.String)
	}
	if errMsg.String != "Server restarted" {
		t.Errorf("expected error='Server restarted', got %q", errMsg.String)
	}
}

// TestStartupReconciliation_RecentRunsUntouched verifies that runs started within
// the last 5 minutes are NOT marked failed.
func TestStartupReconciliation_RecentRunsUntouched(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	recentTime := time.Now().Add(-1 * time.Minute)
	recentID := insertRunningRun(t, db, "proj-recent", recentTime)

	if err := services.RunStartupTasks(ctx, repo, nil); err != nil {
		t.Fatalf("RunStartupTasks: %v", err)
	}

	var status string
	row := db.QueryRow(`SELECT status FROM scout_runs WHERE id = ?`, recentID)
	if err := row.Scan(&status); err != nil {
		t.Fatalf("scan run: %v", err)
	}

	if status != "running" {
		t.Errorf("expected status=running for recent run, got %q", status)
	}
}

// TestStartupReconciliation_CouncilReconciliationRuns verifies that completed
// reports without a council row trigger a council start (report_councils row created).
// We don't actually run AI; StartCouncil is enough to verify reconciliation fires.
func TestStartupReconciliation_CouncilReconciliationRuns(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	// Insert a project + completed report with no council row.
	_, err := db.Exec(
		`INSERT INTO projects (id, name, mode) VALUES ('proj-council', 'Council Test', 'research')`,
	)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	res, err := db.Exec(
		`INSERT INTO research_reports (project_id, title, status, post_count, clusters, assessment, model_used)
		 VALUES ('proj-council', 'Test Report', 'completed', 0, '[]', 'ok', 'test')`,
	)
	if err != nil {
		t.Fatalf("insert report: %v", err)
	}
	reportID, _ := res.LastInsertId()

	// Passing nil AI service; reconciliation marks stale councils but StartCouncil
	// with nil AI will insert the row and then fail in the goroutine – that's fine for this test.
	_ = services.RunStartupTasks(ctx, repo, nil)

	// Give the background goroutine a moment to insert the council row.
	time.Sleep(50 * time.Millisecond)

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM report_councils WHERE report_id = ?`, reportID).Scan(&count)
	if count < 1 {
		t.Errorf("expected a council row to be created for orphaned report %d, got %d", reportID, count)
	}
}

// TestDemoSeed_SeededOnce verifies that seedDemoData inserts the demo project on first run.
func TestDemoSeed_SeededOnce(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	t.Setenv("SCOUT_INIT_DEMO", "1")

	if err := services.RunStartupTasks(ctx, repo, nil); err != nil {
		t.Fatalf("RunStartupTasks: %v", err)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM projects WHERE id = 'demo-project'`).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 demo project, got %d", count)
	}
}

// TestDemoSeed_NotSeededWithoutEnv verifies that demo data is NOT inserted when
// SCOUT_INIT_DEMO is unset.
func TestDemoSeed_NotSeededWithoutEnv(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	os.Unsetenv("SCOUT_INIT_DEMO")

	if err := services.RunStartupTasks(ctx, repo, nil); err != nil {
		t.Fatalf("RunStartupTasks: %v", err)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM projects WHERE id = 'demo-project'`).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 demo projects without env, got %d", count)
	}
}

// TestDemoSeed_IdempotentOnSecondRun verifies that calling RunStartupTasks twice
// with SCOUT_INIT_DEMO=1 does not error and produces exactly one project.
func TestDemoSeed_IdempotentOnSecondRun(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	t.Setenv("SCOUT_INIT_DEMO", "1")

	if err := services.RunStartupTasks(ctx, repo, nil); err != nil {
		t.Fatalf("first RunStartupTasks: %v", err)
	}
	if err := services.RunStartupTasks(ctx, repo, nil); err != nil {
		t.Fatalf("second RunStartupTasks: %v", err)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM projects WHERE id = 'demo-project'`).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 demo project after idempotent seed, got %d", count)
	}
}
