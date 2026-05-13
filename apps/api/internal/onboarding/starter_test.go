package onboarding_test

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newStarterService(t *testing.T) *onboarding.Service {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	svc, err := onboarding.NewService(
		onboarding.Config{CompletionKey: completionKey, StateKey: stateKey},
		settings.NewRepository(db),
		repository.New(db),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.Save(context.Background(), map[string]string{"AI_PROVIDER": "anthropic"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return svc
}

func TestCreateStarterProjectCreatesProjectAndQuery(t *testing.T) {
	svc := newStarterService(t)
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
	svc := newStarterService(t)
	req := onboarding.StarterProjectRequest{ProjectID: "ai-notes", ProjectName: "AI Notes", Query: "meeting notes too time consuming", Platform: "reddit"}
	first, err := svc.CreateStarterProject(context.Background(), req)
	if err != nil {
		t.Fatalf("first CreateStarterProject: %v", err)
	}
	second, err := svc.CreateStarterProject(context.Background(), req)
	if err != nil {
		t.Fatalf("second CreateStarterProject: %v", err)
	}
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
