package google

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// fakeProvider is a SearchProvider that returns a fixed set of results.
type fakeProvider struct {
	results []SearchResult
	err     error
}

func (f *fakeProvider) SearchBatch(_ context.Context, _ []string, _ SearchOptions) ([]SearchResult, error) {
	return f.results, f.err
}

func makeTestProject() domain.Project {
	desc := "test description"
	return domain.Project{
		ID:          "proj-test",
		Name:        "Test Project",
		Description: &desc,
	}
}

// createProject inserts a project and returns it.
func createProject(t *testing.T, repo *repository.Repository, id, name string) domain.Project {
	t.Helper()
	p, err := repo.CreateProject(context.Background(), domain.Project{ID: id, Name: name, Mode: "research"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return p
}

// createGoogleQuery inserts an enabled google query for the project.
func createGoogleQuery(t *testing.T, repo *repository.Repository, projectID, keyword string) {
	t.Helper()
	_, err := repo.CreateQuery(context.Background(), projectID, "google", keyword, "", true)
	if err != nil {
		t.Fatalf("create query: %v", err)
	}
}

// TestRunGoogleScoutPipeline_NoQueries verifies that a run with no enabled queries
// completes with 0 posts checked and 0 posts found.
func TestRunGoogleScoutPipeline_NoQueries(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	createProject(t, repo, "proj-noq", "No Queries")
	runID, err := repo.CreateScoutRun(ctx, "proj-noq", "google")
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	provider := &fakeProvider{results: nil}
	res, err := RunGoogleScoutPipeline(ctx, repo, nil, makeTestProject(), "proj-noq", runID, provider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostsChecked != 0 || res.PostsFound != 0 {
		t.Errorf("expected 0/0, got %d/%d", res.PostsChecked, res.PostsFound)
	}
	if res.RunID != runID {
		t.Errorf("expected runID %d, got %d", runID, res.RunID)
	}
}

// TestRunGoogleScoutPipeline_WithResults verifies a happy-path run with one query
// and two search results persists the results and marks the run completed.
func TestRunGoogleScoutPipeline_WithResults(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	createProject(t, repo, "proj-res", "With Results")
	runID, err := repo.CreateScoutRun(ctx, "proj-res", "google")
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	createGoogleQuery(t, repo, "proj-res", "project management pain points")

	rank1, rank2 := 1.0, 2.0
	provider := &fakeProvider{results: []SearchResult{
		{
			Title:   "Why project management tools are frustrating",
			URL:     "https://example.com/pm-frustration",
			Snippet: "Users report constant context switching and notification overload.",
			Rank:    &rank1,
		},
		{
			Title:   "The hidden costs of bad task trackers",
			URL:     "https://blog.example.org/task-trackers",
			Snippet: "Teams lose hours re-entering data across disconnected tools.",
			Rank:    &rank2,
		},
	}}

	project := makeTestProject()
	res, err := RunGoogleScoutPipeline(ctx, repo, nil, project, "proj-res", runID, provider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RunID != runID {
		t.Errorf("runID mismatch: got %d want %d", res.RunID, runID)
	}
	if res.PostsChecked < 1 {
		t.Errorf("expected PostsChecked >= 1, got %d", res.PostsChecked)
	}
	if res.PostsFound < 1 {
		t.Errorf("expected PostsFound >= 1, got %d", res.PostsFound)
	}

	// Verify run status is completed in the DB.
	run, err := repo.GetScoutRun(ctx, "proj-res", runID)
	if err != nil {
		t.Fatalf("get scout run: %v", err)
	}
	if run.Status != "completed" {
		t.Errorf("expected status=completed, got %q", run.Status)
	}
}

// TestRunGoogleScoutPipeline_Cancellation verifies that a cancelled context causes
// the pipeline to exit early (either no error or context.Canceled).
func TestRunGoogleScoutPipeline_Cancellation(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()

	createProject(t, repo, "proj-cancel", "Cancel Test")
	runID, err := repo.CreateScoutRun(ctx, "proj-cancel", "google")
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	createGoogleQuery(t, repo, "proj-cancel", "some keyword")

	cancelled, cancel := context.WithCancel(ctx)
	cancel() // cancel immediately

	provider := &fakeProvider{results: []SearchResult{
		{Title: "irrelevant", URL: "https://example.com/x"},
	}}

	_, runErr := RunGoogleScoutPipeline(cancelled, repo, nil, makeTestProject(), "proj-cancel", runID, provider)
	// Either no error or context.Canceled is acceptable; the pipeline must not panic.
	if runErr != nil && runErr != context.Canceled {
		// Unwrap to check for context.Canceled
		unwrapped := runErr
		for {
			if unwrapped == context.Canceled {
				return // acceptable
			}
			type unwrapper interface{ Unwrap() error }
			if u, ok := unwrapped.(unwrapper); ok {
				unwrapped = u.Unwrap()
			} else {
				break
			}
		}
		t.Errorf("unexpected error on cancellation: %v", runErr)
	}
}

// TestDeriveObjective verifies the objective string uses description when available.
func TestDeriveObjective(t *testing.T) {
	desc := "indie SaaS founders"
	project := domain.Project{Name: "Scout", Description: &desc}
	obj := deriveObjective(project, "saas tools")
	if obj == "" {
		t.Error("expected non-empty objective")
	}
	if len(obj) < 20 {
		t.Errorf("objective seems too short: %q", obj)
	}
}

// TestDeriveObjective_FallsBackToName verifies fallback to project name when no description.
func TestDeriveObjective_FallsBackToName(t *testing.T) {
	project := domain.Project{Name: "MyProject"}
	obj := deriveObjective(project, "fallback keyword")
	if obj == "" {
		t.Error("expected non-empty objective")
	}
}

// TestBuildKeywordSummaries verifies aggregation counts are correct for a known set.
func TestBuildKeywordSummaries(t *testing.T) {
	rel1 := 0.8
	rel2 := 0.6
	results := []normalizedResult{
		{RootKeyword: "kw1", RelevanceFit: "direct_fit", RelevanceScore: &rel1, OutreachCandidate: 1},
		{RootKeyword: "kw1", RelevanceFit: "weak_fit", RelevanceScore: &rel2},
		{RootKeyword: "kw2", RelevanceFit: "direct_fit", RelevanceScore: &rel1},
	}
	summaries := buildKeywordSummaries(results, []string{"kw1", "kw2"})
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	kw1 := summaries[0]
	if kw1.RootKeyword != "kw1" {
		t.Errorf("kw1 mismatch: %q", kw1.RootKeyword)
	}
	if kw1.TotalResults != 2 {
		t.Errorf("kw1 TotalResults: want 2, got %d", kw1.TotalResults)
	}
	if kw1.RelevantResults != 1 {
		t.Errorf("kw1 RelevantResults: want 1 (non-weak), got %d", kw1.RelevantResults)
	}
	if kw1.OutreachCandidates != 1 {
		t.Errorf("kw1 OutreachCandidates: want 1, got %d", kw1.OutreachCandidates)
	}

	kw2 := summaries[1]
	if kw2.TotalResults != 1 {
		t.Errorf("kw2 TotalResults: want 1, got %d", kw2.TotalResults)
	}
}

// TestBuildDomainStats verifies domain grouping and counts.
func TestBuildDomainStats(t *testing.T) {
	rel := 0.7
	results := []normalizedResult{
		{Domain: "example.com", RelevanceFit: "direct_fit", RelevanceScore: &rel, OutreachCandidate: 1},
		{Domain: "example.com", RelevanceFit: "weak_fit"},
		{Domain: "other.org", RelevanceFit: "direct_fit"},
	}
	stats := buildDomainStats(results)
	if len(stats) != 2 {
		t.Fatalf("expected 2 domain stats, got %d", len(stats))
	}
	ex := stats[0]
	if ex.Domain != "example.com" {
		t.Errorf("domain mismatch: %q", ex.Domain)
	}
	if ex.ResultCount != 2 {
		t.Errorf("ResultCount: want 2, got %d", ex.ResultCount)
	}
	if ex.RelevantCount != 1 {
		t.Errorf("RelevantCount: want 1, got %d", ex.RelevantCount)
	}
	if ex.OutreachCandidateCount != 1 {
		t.Errorf("OutreachCandidateCount: want 1, got %d", ex.OutreachCandidateCount)
	}
}
