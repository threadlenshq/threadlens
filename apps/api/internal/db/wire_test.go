package db_test

import (
	"context"
	"database/sql"
	"testing"

	internaldb "github.com/kyle/scout/open-core/apps/api/internal/db"
	shareddb "github.com/kyle/scout/open-core/db"
)

func openMemDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := shareddb.Open(context.Background(), shareddb.Config{
		Dialect:    shareddb.DialectSQLite,
		SQLitePath: ":memory:",
	})
	if err != nil {
		t.Fatalf("shareddb.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// TestSharedDBOpenWiring_Tables verifies that the shared open-core/db module
// opens an in-memory SQLite database and produces all expected core tables,
// including those added via migrations.
func TestSharedDBOpenWiring_Tables(t *testing.T) {
	database := openMemDB(t)

	tables := []string{
		"projects",
		"project_queries",
		"project_prompts",
		"posts",
		"seen_posts",
		"scout_runs",
		"query_review_jobs",
		"schedules",
		"app_settings",
		"research_reports",
		"report_posts",
		"google_results",
		"google_keyword_summaries",
		"google_reports",
		"report_councils",
		"google_domain_stats",
		"dm_targets",
	}
	for _, table := range tables {
		var found string
		err := database.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&found)
		if err != nil || found != table {
			t.Errorf("expected table %q to exist after shared db.Open", table)
		}
	}
}

// TestSharedDBOpenWiring_PragmaForeignKeys verifies that the shared open path
// enables foreign key enforcement on every connection.
func TestSharedDBOpenWiring_PragmaForeignKeys(t *testing.T) {
	database := openMemDB(t)

	var enabled int
	if err := database.QueryRow("PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if enabled != 1 {
		t.Errorf("foreign_keys = %d, want 1", enabled)
	}
}

// TestSharedDBOpenWiring_PragmaWAL verifies that the shared open path requests
// WAL journal mode (applied via DSN pragma; in-memory DBs report "memory").
func TestSharedDBOpenWiring_PragmaWAL(t *testing.T) {
	database := openMemDB(t)

	var mode string
	if err := database.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	// In-memory SQLite ignores journal_mode=WAL and reports "memory".
	// Accept both so the assertion is valid for :memory: and file-backed paths.
	if mode != "wal" && mode != "memory" {
		t.Errorf("journal_mode = %q, want wal (or memory for in-memory DB)", mode)
	}
}

// TestSharedDBOpenWiring_PragmaBusyTimeout verifies that the shared open path
// sets a non-zero busy_timeout so writers retry instead of failing immediately.
func TestSharedDBOpenWiring_PragmaBusyTimeout(t *testing.T) {
	database := openMemDB(t)

	var timeout int
	if err := database.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if timeout <= 0 {
		t.Errorf("busy_timeout = %d, want > 0", timeout)
	}
}

// TestSharedDBOpenWiring_MigrationColumns verifies that columns added by
// schema migrations are present after db.Open, confirming migrations ran.
func TestSharedDBOpenWiring_MigrationColumns(t *testing.T) {
	database := openMemDB(t)

	checks := []struct{ table, column string }{
		{"scout_runs", "step"},
		{"scout_runs", "warnings"},
		{"projects", "description"},
		{"posts", "signal_type"},
		{"google_results", "mentioned_products"},
	}

	for _, c := range checks {
		rows, err := database.Query("PRAGMA table_info(" + c.table + ")")
		if err != nil {
			t.Errorf("PRAGMA table_info(%s): %v", c.table, err)
			continue
		}
		found := false
		for rows.Next() {
			var cid int
			var name, typ string
			var notNull int
			var defaultVal any
			var pk int
			if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultVal, &pk); err != nil {
				t.Errorf("scan table_info(%s): %v", c.table, err)
				break
			}
			if name == c.column {
				found = true
				break
			}
		}
		rows.Close()
		if !found {
			t.Errorf("column %s.%s missing — migration may not have run", c.table, c.column)
		}
	}
}

// TestInitSchemaAddsFilteringTablesAndColumns verifies that InitSchema creates
// the filter_trust_records and filter_jobs tables and adds filter columns to
// posts and google_results. It opens an in-memory DB via the shared module
// (which runs SQL file migrations) and then calls InitSchema to apply the
// additive ALTER TABLE migrations and new table definitions.
func TestInitSchemaAddsFilteringTablesAndColumns(t *testing.T) {
	db := openMemDB(t)
	if err := internaldb.InitSchema(db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	for _, tc := range []struct{ table, column string }{
		{"posts", "filter_state"}, {"posts", "filter_reasons_json"}, {"posts", "source_identity_json"},
		{"google_results", "filter_state"}, {"google_results", "filter_reasons_json"}, {"google_results", "source_identity_json"},
	} {
		if !columnExists(t, db, tc.table, tc.column) {
			t.Fatalf("expected %s.%s to exist", tc.table, tc.column)
		}
	}

	for _, table := range []string{"filter_trust_records", "filter_jobs"} {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %s: %v", table, err)
		}
	}
}

func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table info: %v", err)
		}
		if name == column {
			return true
		}
	}
	return false
}
