package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// coreExpectedTables is the canonical list of tables that every Migrate call
// must create, regardless of dialect.  Define it once to avoid duplication
// between the SQLite and Postgres test cases.
var coreExpectedTables = []string{
	"schema_migrations",
	"projects",
	"project_queries",
	"posts",
	"scout_runs",
	"research_reports",
	"google_results",
	"google_reports",
}

// ---------------------------------------------------------------------------
// SQLite migration tests
// ---------------------------------------------------------------------------

func TestMigrateSQLiteCreatesCoreTablesAndMetadata(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer database.Close()

	if err := Migrate(context.Background(), database, DialectSQLite); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range coreExpectedTables {
		if !sqliteTableExists(t, database, table) {
			t.Errorf("expected table %q to exist after migration", table)
		}
	}

	var count int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE scope = 'core' AND version = '0001_core'`,
	).Scan(&count); err != nil {
		t.Fatalf("querying schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("schema_migrations rows for scope='core' version='0001_core' = %d, want 1", count)
	}
}

func TestMigrateSQLiteIsIdempotent(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer database.Close()

	// First migration run
	if err := Migrate(context.Background(), database, DialectSQLite); err != nil {
		t.Fatalf("Migrate (run 1): %v", err)
	}

	var countAfterFirst int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE scope = 'core'`,
	).Scan(&countAfterFirst); err != nil {
		t.Fatalf("querying schema_migrations after first run: %v", err)
	}
	if countAfterFirst <= 0 {
		t.Errorf("schema_migrations rows for scope='core' = %d after first run, want > 0", countAfterFirst)
	}

	// Second migration run
	if err := Migrate(context.Background(), database, DialectSQLite); err != nil {
		t.Fatalf("Migrate (run 2): %v", err)
	}

	var countAfterSecond int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE scope = 'core'`,
	).Scan(&countAfterSecond); err != nil {
		t.Fatalf("querying schema_migrations after second run: %v", err)
	}

	// Verify idempotency: second run should not add or duplicate rows
	if countAfterSecond != countAfterFirst {
		t.Errorf("schema_migrations rows for scope='core': after first run = %d, after second run = %d, want equal (idempotent)", countAfterFirst, countAfterSecond)
	}
}

func TestMigrateSQLiteAllowsHackerNewsPlatform(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer database.Close()
	if _, err := database.Exec("PRAGMA foreign_keys=ON"); err != nil {
		t.Fatalf("enable fk: %v", err)
	}
	if err := Migrate(context.Background(), database, DialectSQLite); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if _, err := database.Exec(
		`INSERT INTO projects (id, name, mode) VALUES ('p1', 'P1', 'research')`); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	cases := []struct {
		name string
		stmt string
	}{
		{"project_queries", `INSERT INTO project_queries (project_id, platform, query_url, angle) VALUES ('p1','hackernews','manual invoicing','pain')`},
		{"posts", `INSERT INTO posts (id, project_id, platform) VALUES ('hn_1','p1','hackernews')`},
		{"scout_runs", `INSERT INTO scout_runs (project_id, platform) VALUES ('p1','hackernews')`},
		{"schedules", `INSERT INTO schedules (project_id, platform, cron_expr) VALUES ('p1','hackernews','0 * * * *')`},
	}
	for _, c := range cases {
		if _, err := database.Exec(c.stmt); err != nil {
			t.Errorf("%s: insert hackernews failed: %v", c.name, err)
		}
	}

	for _, idx := range []string{"idx_posts_project", "idx_posts_score", "idx_posts_filter_state", "idx_queries_project", "idx_runs_project", "idx_schedules_project"} {
		var n int
		if err := database.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&n); err != nil {
			t.Fatalf("query index %s: %v", idx, err)
		}
		if n != 1 {
			t.Errorf("index %s missing after rebuild (count=%d)", idx, n)
		}
	}
}

// ---------------------------------------------------------------------------
// Postgres migration test (optional, gated by env var)
// ---------------------------------------------------------------------------

func TestMigratePostgresCreatesCoreTablesWhenDSNProvided(t *testing.T) {
	dsn := os.Getenv("SCOUT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("SCOUT_TEST_POSTGRES_DSN not set — skipping Postgres migration test")
	}

	database, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open(pgx): %v", err)
	}
	defer database.Close()

	// Reset schema to ensure a clean state before migrating.
	if _, err := database.ExecContext(context.Background(),
		`DROP SCHEMA public CASCADE; CREATE SCHEMA public`,
	); err != nil {
		t.Fatalf("resetting postgres schema: %v", err)
	}

	if err := Migrate(context.Background(), database, DialectPostgres); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range coreExpectedTables {
		if !postgresTableExists(t, database, table) {
			t.Errorf("expected table %q to exist after migration", table)
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func sqliteTableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("sqliteTableExists(%q): %v", table, err)
	}
	return true
}

func postgresTableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)`, table,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("postgresTableExists(%q): %v", table, err)
	}
	return exists
}
