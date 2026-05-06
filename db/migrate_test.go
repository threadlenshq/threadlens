package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// SQLite migration tests
// ---------------------------------------------------------------------------

func TestMigrate_SQLite_CoreTablesExist(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer database.Close()

	if err := Migrate(context.Background(), database, DialectSQLite); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range []string{"projects", "posts", "schema_migrations"} {
		if !sqliteTableExists(t, database, table) {
			t.Errorf("expected table %q to exist after migration", table)
		}
	}

	var count int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE id = 'core/0001_core'`,
	).Scan(&count); err != nil {
		t.Fatalf("querying schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("schema_migrations rows for core/0001_core = %d, want 1", count)
	}
}

func TestMigrate_SQLite_Idempotent(t *testing.T) {
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
		`SELECT COUNT(*) FROM schema_migrations WHERE id = 'core/0001_core'`,
	).Scan(&count); err != nil {
		t.Fatalf("querying schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("schema_migrations rows for core/0001_core = %d after two runs, want 1", count)
	}
}

// ---------------------------------------------------------------------------
// Postgres migration test (optional, gated by env var)
// ---------------------------------------------------------------------------

func TestMigrate_Postgres_CoreTablesExist(t *testing.T) {
	dsn := os.Getenv("SCOUT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("SCOUT_TEST_POSTGRES_DSN not set — skipping Postgres migration test")
	}

	database, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open(pgx): %v", err)
	}
	defer database.Close()

	if err := Migrate(context.Background(), database, DialectPostgres); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range []string{"projects", "posts", "schema_migrations"} {
		if !postgresTableExists(t, database, table) {
			t.Errorf("expected table %q to exist after migration", table)
		}
	}

	var count int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE id = 'core/0001_core'`,
	).Scan(&count); err != nil {
		t.Fatalf("querying schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("schema_migrations rows for core/0001_core = %d, want 1", count)
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
