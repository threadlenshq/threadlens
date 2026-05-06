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

	for i := range 2 {
		if err := Migrate(context.Background(), database, DialectSQLite); err != nil {
			t.Fatalf("Migrate (run %d): %v", i+1, err)
		}
	}

	var count int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE scope = 'core'`,
	).Scan(&count); err != nil {
		t.Fatalf("querying schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("schema_migrations rows for scope='core' = %d after two runs, want 1", count)
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
