package db_test

import (
	"context"
	"testing"

	shareddb "github.com/kyle/scout/open-core/db"
)

// TestSharedDBOpenWiring verifies that the shared open-core/db module
// opens an in-memory SQLite database and produces the expected core tables.
func TestSharedDBOpenWiring(t *testing.T) {
	database, err := shareddb.Open(context.Background(), shareddb.Config{
		Dialect:    shareddb.DialectSQLite,
		SQLitePath: ":memory:",
	})
	if err != nil {
		t.Fatalf("shareddb.Open: %v", err)
	}
	defer database.Close()

	for _, table := range []string{
		"projects", "project_queries", "posts", "scout_runs",
		"research_reports", "google_results", "google_reports",
	} {
		var found string
		err := database.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&found)
		if err != nil || found != table {
			t.Errorf("expected table %q to exist after shared db.Open", table)
		}
	}
}
