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
