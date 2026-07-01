package repository_test

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
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

func TestGeneralQuery_CRUDAndMaterialize(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	if _, err := repo.CreateProject(ctx, domain.Project{ID: "p1", Name: "Proj", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	gq, err := repo.CreateGeneralQuery(ctx, "p1", "manual invoicing", "pain", []string{"reddit", "google"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if gq.ID == 0 || len(gq.Platforms) != 2 {
		t.Fatalf("unexpected general query: %+v", gq)
	}

	if _, err := repo.InsertMaterializedQuery(ctx, "p1", gq.ID, "reddit", "https://www.reddit.com/search.json?q=x&sort=new&t=month&limit=100", "pain", true); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.InsertMaterializedQuery(ctx, "p1", gq.ID, "google", "manual invoicing", "pain", true); err != nil {
		t.Fatal(err)
	}

	all, err := repo.ListAllQueries(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	owned := 0
	for _, q := range all {
		if q.GeneralQueryID != nil && *q.GeneralQueryID == gq.ID {
			owned++
		}
	}
	if owned != 2 {
		t.Fatalf("want 2 materialized rows, got %d", owned)
	}

	// DeleteMaterializedQueries clears rows but keeps the general query.
	if err := repo.DeleteMaterializedQueries(ctx, gq.ID); err != nil {
		t.Fatal(err)
	}
	all, _ = repo.ListAllQueries(ctx, "p1")
	for _, q := range all {
		if q.GeneralQueryID != nil {
			t.Fatal("expected no materialized rows after delete")
		}
	}

	// Deleting the general query cascades (re-add a row first).
	if _, err := repo.InsertMaterializedQuery(ctx, "p1", gq.ID, "google", "manual invoicing", "pain", true); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteGeneralQuery(ctx, "p1", gq.ID); err != nil {
		t.Fatal(err)
	}
	gqs, _ := repo.ListGeneralQueries(ctx, "p1")
	if len(gqs) != 0 {
		t.Fatalf("want 0 general queries, got %d", len(gqs))
	}
	all, _ = repo.ListAllQueries(ctx, "p1")
	if len(all) != 0 {
		t.Fatalf("cascade failed: %d project_queries rows remain", len(all))
	}
}
