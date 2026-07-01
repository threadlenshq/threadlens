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
