package repository_test

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func TestMigration_GeneralQueriesSchema(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	ctx := context.Background()

	// general_queries table exists
	var name string
	err := db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='general_queries'").Scan(&name)
	if err != nil {
		t.Fatalf("general_queries table missing: %v", err)
	}

	// project_queries.general_query_id column exists
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(project_queries)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var colName, colType string
		var notNull, pk int
		var dflt any
		if err := rows.Scan(&cid, &colName, &colType, &notNull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		if colName == "general_query_id" {
			found = true
		}
	}
	if !found {
		t.Fatal("project_queries.general_query_id column missing")
	}
}
