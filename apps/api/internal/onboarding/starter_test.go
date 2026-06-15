package onboarding_test

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// newStarterService builds a Service wired to a fresh in-memory DB and returns
// both the service and the underlying repository so tests can inspect
// persisted state directly.  It advances the service to the exploration phase
// via Save so CreateStarterProject pre-conditions are satisfied.
func newStarterService(t *testing.T) (*onboarding.Service, *repository.Repository) {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	svc, err := onboarding.NewService(
		onboarding.Config{CompletionKey: completionKey, StateKey: stateKey},
		settings.NewRepository(db),
		repo, nil,
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.Save(context.Background(), map[string]string{"AI_PROVIDER": "anthropic"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return svc, repo
}

func TestCreateStarterProjectCreatesProjectAndQuery(t *testing.T) {
	svc, _ := newStarterService(t)
	result, err := svc.CreateStarterProject(context.Background(), onboarding.StarterProjectRequest{ProjectID: "ai-notes", ProjectName: "AI Notes", Query: "meeting notes too time consuming", Platform: "reddit"})
	if err != nil {
		t.Fatalf("CreateStarterProject: %v", err)
	}
	if result.Project.ID != "ai-notes" {
		t.Fatalf("project id = %q, want ai-notes", result.Project.ID)
	}
	if result.Query.ID == 0 {
		t.Fatal("starter query id should be non-zero")
	}
	if result.CreatedProject != true || result.CreatedQuery != true {
		t.Fatalf("created flags = project %v query %v, want both true", result.CreatedProject, result.CreatedQuery)
	}
}

func TestCreateStarterProjectIsIdempotent(t *testing.T) {
	svc, repo := newStarterService(t)
	ctx := context.Background()
	req := onboarding.StarterProjectRequest{ProjectID: "ai-notes", ProjectName: "AI Notes", Query: "meeting notes too time consuming", Platform: "reddit"}

	first, err := svc.CreateStarterProject(ctx, req)
	if err != nil {
		t.Fatalf("first CreateStarterProject: %v", err)
	}

	// Capture persisted state after the first call.
	projectsAfterFirst, err := repo.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects after first call: %v", err)
	}
	queriesAfterFirst, err := repo.ListAllQueries(ctx, first.Project.ID)
	if err != nil {
		t.Fatalf("ListAllQueries after first call: %v", err)
	}

	second, err := svc.CreateStarterProject(ctx, req)
	if err != nil {
		t.Fatalf("second CreateStarterProject: %v", err)
	}

	// Capture persisted state after the second call and compare.
	projectsAfterSecond, err := repo.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects after second call: %v", err)
	}
	queriesAfterSecond, err := repo.ListAllQueries(ctx, second.Project.ID)
	if err != nil {
		t.Fatalf("ListAllQueries after second call: %v", err)
	}

	if len(projectsAfterSecond) != len(projectsAfterFirst) {
		t.Fatalf("project count changed: %d after first, %d after second", len(projectsAfterFirst), len(projectsAfterSecond))
	}
	if len(queriesAfterSecond) != len(queriesAfterFirst) {
		t.Fatalf("query count changed: %d after first, %d after second", len(queriesAfterFirst), len(queriesAfterSecond))
	}

	// Return values must also be stable across calls.
	if first.Project.ID != second.Project.ID {
		t.Fatalf("project ids differ: %q vs %q", first.Project.ID, second.Project.ID)
	}
	if first.Query.ID != second.Query.ID {
		t.Fatalf("query ids differ: %d vs %d", first.Query.ID, second.Query.ID)
	}
	if second.CreatedProject || second.CreatedQuery {
		t.Fatalf("second call should reuse existing objects, got created flags project=%v query=%v", second.CreatedProject, second.CreatedQuery)
	}
}
