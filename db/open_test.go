package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenAppliesSQLitePragmasAndMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scout.db")
	database, err := Open(context.Background(), Config{
		Dialect:    DialectSQLite,
		SQLitePath: path,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	// Verify foreign_keys pragma is enabled.
	var fkEnabled int
	if err := database.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("PRAGMA foreign_keys = %d, want 1", fkEnabled)
	}

	// Verify WAL journal mode is active.
	var journalMode string
	if err := database.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("PRAGMA journal_mode = %q, want \"wal\"", journalMode)
	}

	// Verify busy_timeout is set to the expected value (5000 ms).
	var busyTimeout int
	if err := database.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("PRAGMA busy_timeout = %d, want 5000", busyTimeout)
	}

	// Verify migrations ran and the projects table exists.
	if !sqliteOpenTableExists(t, database, "projects") {
		t.Error("expected table \"projects\" to exist after Open")
	}
}

func TestOpenRejectsMissingSQLitePath(t *testing.T) {
	_, err := Open(context.Background(), Config{Dialect: DialectSQLite})
	if err == nil {
		t.Fatal("Open with empty SQLitePath: expected error, got nil")
	}
}

func TestOpenRejectsMissingDatabaseURL(t *testing.T) {
	_, err := Open(context.Background(), Config{Dialect: DialectPostgres})
	if err == nil {
		t.Fatal("Open with empty DatabaseURL: expected error, got nil")
	}
}

// sqliteOpenTableExists reports whether the named table exists in the SQLite DB.
func sqliteOpenTableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatalf("sqliteOpenTableExists(%q): %v", table, err)
	}
	return true
}
