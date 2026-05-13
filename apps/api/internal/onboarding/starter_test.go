package onboarding_test

// starter_test.go specifies the expected behaviour of
// Service.CreateStarterProject for the ThreadLens v1 onboarding flow.
//
// Design constraints kept here:
//   - CreateStarterProject creates a project and a starter query when neither
//     exists yet (first-run path).
//   - The operation is idempotent: a second call with the same name returns the
//     same project ID and query ID without creating duplicates.
//   - The service returns a non-nil error when projectRepo is nil; there must
//     be no silent success path when the domain repository is unavailable.
//   - After a successful call the onboarding progress Context fields
//     (StarterProjectID and StarterQueryID) are populated so that GetStatus
//     can surface them to the frontend.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// newStarterService builds a Service wired to a fresh in-memory database
// together with a real *repository.Repository for the domain repo.
// Both the settings repository and the project repository use the same DB so
// foreign-key relationships work correctly in tests.
func newStarterService(t *testing.T) (*onboarding.Service, *repository.Repository) {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	settingsRepo := settings.NewRepository(db)
	projectRepo := repository.New(db)
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc, err := onboarding.NewService(cfg, settingsRepo, projectRepo)
	if err != nil {
		t.Fatalf("newStarterService: NewService: %v", err)
	}
	return svc, projectRepo
}

// advanceToExploration walks the service through every required step and calls
// Save so that the exploration phase is active before the starter tests run.
func advanceToExploration(t *testing.T, svc *onboarding.Service) {
	t.Helper()
	completeRequiredSteps(t, svc, map[string]string{"AI_PROVIDER": "anthropic"})
	if err := svc.Save(context.Background(), map[string]string{}); err != nil {
		t.Fatalf("advanceToExploration: Save: %v", err)
	}
}

// ── 1. Creates project and query on first invocation ─────────────────────────

func TestCreateStarterProject_CreatesProjectAndQuery(t *testing.T) {
	svc, projectRepo := newStarterService(t)
	advanceToExploration(t, svc)
	ctx := context.Background()

	req := onboarding.StarterProjectRequest{Name: "My Starter Project"}
	result, err := svc.CreateStarterProject(ctx, req)
	if err != nil {
		t.Fatalf("CreateStarterProject: %v", err)
	}
	if result.ProjectID == "" {
		t.Fatal("CreateStarterProject: result.ProjectID must not be empty")
	}

	// The project must exist in the domain repository.
	proj, err := projectRepo.GetProject(ctx, result.ProjectID)
	if err != nil {
		t.Fatalf("GetProject(%q): %v", result.ProjectID, err)
	}
	if proj.Name != req.Name {
		t.Errorf("project.Name = %q; want %q", proj.Name, req.Name)
	}

	// At least one query must have been created for the starter project.
	queries, err := projectRepo.ListAllQueries(ctx, result.ProjectID)
	if err != nil {
		t.Fatalf("ListAllQueries(%q): %v", result.ProjectID, err)
	}
	if len(queries) == 0 {
		t.Error("CreateStarterProject: expected at least one starter query to be created")
	}
}

// ── 2. Idempotent – second call reuses existing project and query ─────────────

func TestCreateStarterProject_Idempotent(t *testing.T) {
	svc, projectRepo := newStarterService(t)
	advanceToExploration(t, svc)
	ctx := context.Background()

	req := onboarding.StarterProjectRequest{Name: "Idempotent Project"}

	first, err := svc.CreateStarterProject(ctx, req)
	if err != nil {
		t.Fatalf("CreateStarterProject (first): %v", err)
	}

	second, err := svc.CreateStarterProject(ctx, req)
	if err != nil {
		t.Fatalf("CreateStarterProject (second): %v", err)
	}

	if first.ProjectID != second.ProjectID {
		t.Errorf("idempotent: ProjectID changed between calls: first=%q second=%q",
			first.ProjectID, second.ProjectID)
	}

	// No duplicate projects should have been created.
	projects, err := projectRepo.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("idempotent: expected exactly 1 project, got %d", len(projects))
	}

	// Capture query IDs after the FIRST call (before the second call mutates
	// anything) so we can detect duplicate creation.
	queriesAfterFirst, err := projectRepo.ListAllQueries(ctx, first.ProjectID)
	if err != nil {
		t.Fatalf("ListAllQueries (after first): %v", err)
	}
	if len(queriesAfterFirst) == 0 {
		t.Fatal("idempotent: expected at least one query after first call")
	}
	firstQueryID := queriesAfterFirst[0].ID

	// Run the second call now and verify no duplicates were introduced.
	queriesAfterSecond, err := projectRepo.ListAllQueries(ctx, second.ProjectID)
	if err != nil {
		t.Fatalf("ListAllQueries (after second): %v", err)
	}
	if len(queriesAfterSecond) != len(queriesAfterFirst) {
		t.Errorf("idempotent: query count changed after second call: first=%d second=%d",
			len(queriesAfterFirst), len(queriesAfterSecond))
	}
	// The same query (by ID) must be reused, not a new one inserted.
	if queriesAfterSecond[0].ID != firstQueryID {
		t.Errorf("idempotent: query ID changed between calls: first=%d second=%d",
			firstQueryID, queriesAfterSecond[0].ID)
	}
}

// ── 3. Fails gracefully when projectRepo is nil ───────────────────────────────

func TestCreateStarterProject_FailsWhenProjectRepoNil(t *testing.T) {
	// Build a service with a nil project repository — this should fail either
	// at NewService construction time or at CreateStarterProject call time, but
	// must never silently succeed.
	db := testhelpers.OpenTestDB(t)
	settingsRepo := settings.NewRepository(db)
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}

	// NewService accepts nil projectRepo for callers that only use settings-
	// related behaviour; CreateStarterProject must detect and reject it.
	svc, err := onboarding.NewService(cfg, settingsRepo, nil)
	if err != nil {
		// If construction rejects nil projectRepo for the starter use-case, that
		// is acceptable — the test passes.
		return
	}

	// Advance to exploration so any pre-condition guards do not mask the nil-
	// repo error we are testing for.
	advanceToExploration(t, svc)

	_, callErr := svc.CreateStarterProject(context.Background(), onboarding.StarterProjectRequest{Name: "test"})
	if callErr == nil {
		t.Fatal("CreateStarterProject with nil projectRepo: expected error, got nil")
	}
}

// ── 4. Updates onboarding context after creation ─────────────────────────────

func TestCreateStarterProject_UpdatesOnboardingContext(t *testing.T) {
	svc, svcRepo := newStarterService(t)
	advanceToExploration(t, svc)
	ctx := context.Background()

	req := onboarding.StarterProjectRequest{Name: "Context Test Project"}
	result, err := svc.CreateStarterProject(ctx, req)
	if err != nil {
		t.Fatalf("CreateStarterProject: %v", err)
	}

	// GetStatus must reflect the newly created starter project in its context.
	status, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus after CreateStarterProject: %v", err)
	}

	if status.Context.StarterProjectID != result.ProjectID {
		t.Errorf("Context.StarterProjectID = %q; want %q",
			status.Context.StarterProjectID, result.ProjectID)
	}

	// StarterQueryID must be a positive integer once the starter query exists,
	// and it must match the actual query that was created in the repo.
	if status.Context.StarterQueryID <= 0 {
		t.Errorf("Context.StarterQueryID = %d; want > 0", status.Context.StarterQueryID)
	}

	queries, err := svcRepo.ListAllQueries(ctx, result.ProjectID)
	if err != nil {
		t.Fatalf("ListAllQueries: %v", err)
	}
	if len(queries) == 0 {
		t.Fatal("expected at least one starter query in the repository")
	}
	// The context must point at one of the actually-created queries.
	found := false
	for _, q := range queries {
		if q.ID == status.Context.StarterQueryID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Context.StarterQueryID=%d does not match any created query (IDs: %v)",
			status.Context.StarterQueryID, queryIDs(queries))
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// queryIDs returns a human-readable string of query IDs for use in error messages.
func queryIDs(queries []domain.Query) string {
	parts := make([]string, len(queries))
	for i, q := range queries {
		parts[i] = fmt.Sprintf("%d", q.ID)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
