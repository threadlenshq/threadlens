package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func TestListProjects_Empty(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	projects, err := repo.ListProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 0 {
		t.Fatalf("want 0 projects, got %d", len(projects))
	}
}

func TestCreateProject_Success(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	p := domain.Project{ID: "p1", Name: "Test", Mode: "research"}
	created, err := repo.CreateProject(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "p1" || created.Name != "Test" || created.Mode != "research" {
		t.Fatalf("unexpected project: %+v", created)
	}
}

func TestCreateProject_MissingFields(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	_, err := repo.CreateProject(context.Background(), domain.Project{Name: "X", Mode: "research"})
	if !errors.Is(err, repository.ErrValidation) {
		t.Fatalf("want ErrValidation for missing id, got %v", err)
	}

	_, err = repo.CreateProject(context.Background(), domain.Project{ID: "p1", Mode: "research"})
	if !errors.Is(err, repository.ErrValidation) {
		t.Fatalf("want ErrValidation for missing name, got %v", err)
	}

	_, err = repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "X"})
	if !errors.Is(err, repository.ErrValidation) {
		t.Fatalf("want ErrValidation for missing mode, got %v", err)
	}
}

func TestCreateProject_InvalidMode(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	_, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "X", Mode: "invalid"})
	if !errors.Is(err, repository.ErrValidation) {
		t.Fatalf("want ErrValidation for bad mode, got %v", err)
	}
}

func TestCreateProject_DuplicateConflict(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	p := domain.Project{ID: "p1", Name: "Test", Mode: "research"}
	if _, err := repo.CreateProject(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	_, err := repo.CreateProject(context.Background(), p)
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("want ErrConflict for duplicate, got %v", err)
	}
}

func TestListProjects_OrderedByCreatedAtDesc(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	for _, id := range []string{"p1", "p2", "p3"} {
		if _, err := repo.CreateProject(context.Background(), domain.Project{ID: id, Name: id, Mode: "research"}); err != nil {
			t.Fatal(err)
		}
	}
	projects, err := repo.ListProjects(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 3 {
		t.Fatalf("want 3 projects, got %d", len(projects))
	}
}

func TestGetProject_NotFound(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	_, err := repo.GetProject(context.Background(), "nonexistent")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestGetProjectWithStats(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	pws, err := repo.GetProjectWithStats(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if pws.ID != "p1" {
		t.Fatalf("want id p1, got %q", pws.ID)
	}
	if pws.Stats.TotalPosts != 0 || pws.Stats.NewPosts != 0 || pws.Stats.TotalQueries != 0 {
		t.Fatalf("unexpected stats: %+v", pws.Stats)
	}
	if pws.Stats.LastRun != nil {
		t.Fatalf("expected nil last_run, got %v", pws.Stats.LastRun)
	}
}

func TestGetProjectWithStats_NotFound(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	_, err := repo.GetProjectWithStats(context.Background(), "missing")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestPatchProject_Fields(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Before", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	updated, err := repo.PatchProject(context.Background(), "p1", map[string]any{"name": "After", "description": "desc"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "After" {
		t.Fatalf("want name After, got %q", updated.Name)
	}
	if updated.Description == nil || *updated.Description != "desc" {
		t.Fatalf("want description desc, got %v", updated.Description)
	}
}

func TestPatchProject_NoUpdates_ReturnsCurrent(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	updated, err := repo.PatchProject(context.Background(), "p1", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Test" {
		t.Fatalf("want unchanged name, got %q", updated.Name)
	}
}

func TestPatchProject_NotFound(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	_, err := repo.PatchProject(context.Background(), "missing", map[string]any{"name": "X"})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteProject_Cascade(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteProject(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}

	_, err := repo.GetProject(context.Background(), "p1")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	err := repo.DeleteProject(context.Background(), "missing")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCloneProject_Success(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	desc := "original desc"
	src := domain.Project{ID: "src", Name: "Source", Mode: "research", Description: &desc}
	if _, err := repo.CreateProject(context.Background(), src); err != nil {
		t.Fatal(err)
	}

	cloned, err := repo.CloneProject(context.Background(), "src", "clone1", "Clone")
	if err != nil {
		t.Fatal(err)
	}
	if cloned.ID != "clone1" || cloned.Name != "Clone" || cloned.Mode != "research" {
		t.Fatalf("unexpected clone: %+v", cloned)
	}
}

func TestCloneProject_SourceNotFound(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	_, err := repo.CloneProject(context.Background(), "missing", "c1", "Clone")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCloneProject_MissingFields(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "src", Name: "Source", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	_, err := repo.CloneProject(context.Background(), "src", "", "Clone")
	if !errors.Is(err, repository.ErrValidation) {
		t.Fatalf("want ErrValidation for missing newID, got %v", err)
	}

	_, err = repo.CloneProject(context.Background(), "src", "c1", "")
	if !errors.Is(err, repository.ErrValidation) {
		t.Fatalf("want ErrValidation for missing name, got %v", err)
	}
}

func TestCloneProject_DuplicateConflict(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "src", Name: "Source", Mode: "research"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "existing", Name: "Existing", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	_, err := repo.CloneProject(context.Background(), "src", "existing", "Clone")
	if !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("want ErrConflict for duplicate clone id, got %v", err)
	}
}

func TestSelectAngle_Success(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	updated, err := repo.SelectAngle(context.Background(), "p1", 42, 1)
	if err != nil {
		t.Fatal(err)
	}
	if updated.SelectedReportID == nil || *updated.SelectedReportID != 42 {
		t.Fatalf("want selected_report_id 42, got %v", updated.SelectedReportID)
	}
	if updated.SelectedClusterIndex == nil || *updated.SelectedClusterIndex != 1 {
		t.Fatalf("want selected_cluster_index 1, got %v", updated.SelectedClusterIndex)
	}
}

func TestSelectAngle_NotFound(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	_, err := repo.SelectAngle(context.Background(), "missing", 1, 0)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestGraduateProject_Success(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"}); err != nil {
		t.Fatal(err)
	}
	// Set a selected_report_id first
	if _, err := repo.SelectAngle(context.Background(), "p1", 10, 0); err != nil {
		t.Fatal(err)
	}

	updated, err := repo.GraduateProject(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Mode != "marketing" {
		t.Fatalf("want mode marketing, got %q", updated.Mode)
	}
}

func TestGraduateProject_AlreadyMarketing(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "marketing"}); err != nil {
		t.Fatal(err)
	}

	_, err := repo.GraduateProject(context.Background(), "p1")
	if !errors.Is(err, repository.ErrValidation) {
		t.Fatalf("want ErrValidation for already marketing, got %v", err)
	}
}

func TestGraduateProject_NoAngleSelected(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"}); err != nil {
		t.Fatal(err)
	}

	_, err := repo.GraduateProject(context.Background(), "p1")
	if !errors.Is(err, repository.ErrValidation) {
		t.Fatalf("want ErrValidation for no angle selected, got %v", err)
	}
}

func TestGraduateProject_NotFound(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)

	_, err := repo.GraduateProject(context.Background(), "missing")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
