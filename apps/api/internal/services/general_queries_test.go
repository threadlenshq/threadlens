package services_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newGQDeps(t *testing.T) (*services.GeneralQueryService, *repository.Repository) {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Proj", Mode: "research"}); err != nil {
		t.Fatal(err)
	}
	return services.NewGeneralQueryService(repo), repo
}

func TestGeneralQueryService_CreateMaterializesRows(t *testing.T) {
	svc, repo := newGQDeps(t)
	en := true
	gq, status, msg := svc.Create(context.Background(), "p1", services.GeneralQueryRequest{
		QueryText: "manual invoicing", Angle: "pain", Platforms: []string{"reddit", "google"}, Enabled: &en,
	})
	if msg != "" || status != http.StatusCreated {
		t.Fatalf("create failed: %d %q", status, msg)
	}
	all, _ := repo.ListAllQueries(context.Background(), "p1")
	if len(all) != 2 {
		t.Fatalf("want 2 materialized rows, got %d", len(all))
	}
	for _, q := range all {
		if q.GeneralQueryID == nil || *q.GeneralQueryID != gq.ID {
			t.Fatalf("row not linked to general query: %+v", q)
		}
		if q.Platform == "google" && q.QueryURL != "manual invoicing" {
			t.Fatalf("google row url: %q", q.QueryURL)
		}
	}
}

func TestGeneralQueryService_CollisionBlocked(t *testing.T) {
	svc, repo := newGQDeps(t)
	// Pre-existing standalone google query with the same normalized URL.
	if _, err := repo.CreateQuery(context.Background(), "p1", "google", "manual invoicing", "pain", true); err != nil {
		t.Fatal(err)
	}
	en := true
	_, status, msg := svc.Create(context.Background(), "p1", services.GeneralQueryRequest{
		QueryText: "Manual   Invoicing", Angle: "pain", Platforms: []string{"google"}, Enabled: &en,
	})
	if status != http.StatusConflict {
		t.Fatalf("want 409 conflict, got %d (%q)", status, msg)
	}
}

func TestGeneralQueryService_UpdateResyncsRows(t *testing.T) {
	svc, repo := newGQDeps(t)
	en := true
	gq, _, _ := svc.Create(context.Background(), "p1", services.GeneralQueryRequest{
		QueryText: "invoicing", Angle: "pain", Platforms: []string{"reddit", "google"}, Enabled: &en,
	})
	// Drop google, keep reddit.
	_, status, msg := svc.Update(context.Background(), "p1", gq.ID, services.GeneralQueryRequest{
		QueryText: "invoicing", Angle: "pain", Platforms: []string{"reddit"}, Enabled: &en,
	})
	if msg != "" || status != http.StatusOK {
		t.Fatalf("update failed: %d %q", status, msg)
	}
	all, _ := repo.ListAllQueries(context.Background(), "p1")
	if len(all) != 1 || all[0].Platform != "reddit" {
		t.Fatalf("want 1 reddit row after resync, got %+v", all)
	}
}

// TestQueryService_MaterializedRowsAreReadOnly verifies that the per-platform
// QueryService Patch and Delete endpoints refuse to touch rows that are owned
// by a general query (general_query_id IS NOT NULL), returning 409, while
// standalone rows remain fully editable.
func TestQueryService_MaterializedRowsAreReadOnly(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()
	if _, err := repo.CreateProject(ctx, domain.Project{ID: "p1", Name: "Proj", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	gqSvc := services.NewGeneralQueryService(repo)
	qSvc := services.NewQueryService(repo, nil)
	en := true

	// Create a general query for google - materializes one row.
	gq, status, msg := gqSvc.Create(ctx, "p1", services.GeneralQueryRequest{
		QueryText: "manual invoicing", Angle: "pain", Platforms: []string{"google"}, Enabled: &en,
	})
	if msg != "" || status != http.StatusCreated {
		t.Fatalf("create general query failed: %d %q", status, msg)
	}

	// Find the materialized row.
	all, err := repo.ListAllQueries(ctx, "p1")
	if err != nil || len(all) != 1 {
		t.Fatalf("expected 1 materialized row, got %d (err: %v)", len(all), err)
	}
	rowID := all[0].ID
	if all[0].GeneralQueryID == nil || *all[0].GeneralQueryID != gq.ID {
		t.Fatalf("row not linked to general query: %+v", all[0])
	}

	// Delete must return 409 and the row must still exist.
	delStatus, delMsg := qSvc.Delete(ctx, "p1", rowID)
	if delStatus != http.StatusConflict {
		t.Errorf("Delete materialized row: want 409, got %d (%q)", delStatus, delMsg)
	}
	after, _ := repo.ListAllQueries(ctx, "p1")
	if len(after) != 1 {
		t.Errorf("Delete materialized row: row was removed despite 409; rows after: %d", len(after))
	}

	// Patch must return 409 and the row must be unchanged.
	_, patchStatus, patchMsg := qSvc.Patch(ctx, "p1", rowID, map[string]any{"angle": "x"})
	if patchStatus != http.StatusConflict {
		t.Errorf("Patch materialized row: want 409, got %d (%q)", patchStatus, patchMsg)
	}
	after2, _ := repo.ListAllQueries(ctx, "p1")
	if len(after2) == 0 {
		t.Errorf("Patch materialized row: row no longer exists")
	} else if after2[0].Angle != "pain" {
		t.Errorf("Patch materialized row: angle was changed to %q", after2[0].Angle)
	}

	// Control case: standalone query can be patched and deleted normally.
	standalone, err := repo.CreateQuery(ctx, "p1", "reddit", "/r/selfhosted", "test angle", true)
	if err != nil {
		t.Fatal(err)
	}
	_, ctlPatchStatus, ctlPatchMsg := qSvc.Patch(ctx, "p1", standalone.ID, map[string]any{"angle": "updated"})
	if ctlPatchStatus != http.StatusOK {
		t.Errorf("Patch standalone row: want 200, got %d (%q)", ctlPatchStatus, ctlPatchMsg)
	}
	ctlDelStatus, ctlDelMsg := qSvc.Delete(ctx, "p1", standalone.ID)
	if ctlDelStatus != http.StatusNoContent {
		t.Errorf("Delete standalone row: want 204, got %d (%q)", ctlDelStatus, ctlDelMsg)
	}
}

// TestGeneralQueryService_UpdateCollision409PreservesRows verifies that a 409
// collision on Update leaves the general query's original materialized rows intact
// (regression: previously the rows were deleted before the collision check).
func TestGeneralQueryService_UpdateCollision409PreservesRows(t *testing.T) {
	svc, repo := newGQDeps(t)
	en := true

	// Create general query G for google with "invoicing" - materializes one google row.
	gq, status, msg := svc.Create(context.Background(), "p1", services.GeneralQueryRequest{
		QueryText: "invoicing", Angle: "pain", Platforms: []string{"google"}, Enabled: &en,
	})
	if msg != "" || status != http.StatusCreated {
		t.Fatalf("create G failed: %d %q", status, msg)
	}

	// Create a standalone google query with "different keyword" - this will collide
	// when we try to update G to use the same text.
	if _, err := repo.CreateQuery(context.Background(), "p1", "google", "different keyword", "pain", true); err != nil {
		t.Fatal(err)
	}

	// Attempt Update on G changing to "different keyword" - should get 409 because
	// the standalone row already uses that normalized URL.
	_, status, msg = svc.Update(context.Background(), "p1", gq.ID, services.GeneralQueryRequest{
		QueryText: "different keyword", Angle: "pain", Platforms: []string{"google"}, Enabled: &en,
	})
	if status != http.StatusConflict {
		t.Fatalf("want 409 conflict, got %d (%q)", status, msg)
	}

	// After the failed update, G's original materialized row must still exist.
	all, err := repo.ListAllQueries(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	var gqRows []domain.Query
	for _, q := range all {
		if q.GeneralQueryID != nil && *q.GeneralQueryID == gq.ID {
			gqRows = append(gqRows, q)
		}
	}
	if len(gqRows) != 1 {
		t.Fatalf("want 1 materialized row for G after 409, got %d (rows deleted on rejected update)", len(gqRows))
	}
	if services.NormalizeQueryURL("google", gqRows[0].QueryURL) != services.NormalizeQueryURL("google", "invoicing") {
		t.Fatalf("original row URL changed: %q", gqRows[0].QueryURL)
	}
}
