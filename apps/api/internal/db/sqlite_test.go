//go:build ignore

// This file is excluded from compilation. The tests it contained have been
// migrated to wire_test.go which uses the shared open-core/db module.

package db

import (
	"database/sql"
	"testing"
)

func TestOpenInitializesCoreTables(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, table := range []string{"projects", "project_queries", "posts", "scout_runs", "research_reports", "google_results", "google_reports"} {
		if !tableExists(t, db, table) {
			t.Fatalf("table %s does not exist", table)
		}
	}
}

func TestOpenEnablesForeignKeys(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var enabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 {
		t.Fatalf("foreign_keys = %d, want 1", enabled)
	}
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var found string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&found)
	return err == nil && found == name
}
