package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

var socialAll = []string{"reddit", "bluesky", "hackernews"}

func TestRunAll_MergesCountsAcrossPlatforms(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()
	mkProject(t, repo, "p1", "research")
	mkQuery(t, repo, "p1", "reddit", "https://www.reddit.com/search?q=x", "pain", true)
	mkQuery(t, repo, "p1", "bluesky", "x", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{{ID: "t3_r", Title: "R", Author: "u", Permalink: "/r/x/1", URL: "https://www.reddit.com/r/x/1"}}, nil
	}
	runner.fetchBluesky = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{{ID: "at://b/1", Text: "B", AuthorHandle: "u.bsky", PostURL: "https://bsky.app/p/1"}}, nil
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{
		"t3_r":     scored("t3_r", 5),
		"at://b/1": scored("at://b/1", 5),
	})

	runID, _ := repo.CreateScoutRun(ctx, "p1", "all")
	res, err := runner.runAll(ctx, "p1", runID, socialAll)
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}
	if res.PostsFound != 2 {
		t.Fatalf("merged postsFound: want 2, got %d", res.PostsFound)
	}
	run, _ := repo.GetScoutRun(ctx, "p1", runID)
	if run.Status != "completed" {
		t.Fatalf("run status = %s, want completed", run.Status)
	}
	posts, _ := repo.ListPosts(ctx, "p1", repository.PostFilters{})
	if len(posts) != 2 {
		t.Fatalf("want 2 merged posts, got %d", len(posts))
	}
}

func TestRunAll_DedupsDuplicateTargets(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()
	mkProject(t, repo, "p1", "research")
	// Two reddit queries with the same q term but different encoding/params.
	mkQuery(t, repo, "p1", "reddit", "https://www.reddit.com/search.json?q=manual+invoicing&sort=new", "pain", true)
	mkQuery(t, repo, "p1", "reddit", "https://www.reddit.com/search?q=manual%20invoicing", "angle2", true)

	var gotURLs []string
	runner.fetchReddit = func(_ context.Context, urls []string, _ func(int, int)) ([]FetchedPost, error) {
		gotURLs = urls
		return nil, nil
	}

	runID, _ := repo.CreateScoutRun(ctx, "p1", "all")
	if _, err := runner.runAll(ctx, "p1", runID, []string{"reddit"}); err != nil {
		t.Fatalf("runAll: %v", err)
	}
	if len(gotURLs) != 1 {
		t.Fatalf("expected deduped to 1 reddit fetch target, got %d: %v", len(gotURLs), gotURLs)
	}
}

func TestRunAll_PartialFailureContinues(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()
	mkProject(t, repo, "p1", "research")
	mkQuery(t, repo, "p1", "reddit", "https://www.reddit.com/search?q=x", "pain", true)
	mkQuery(t, repo, "p1", "bluesky", "x", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{{ID: "t3_r", Title: "R", Author: "u", Permalink: "/r/x/1", URL: "https://www.reddit.com/r/x/1"}}, nil
	}
	runner.fetchBluesky = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return nil, errors.New("bluesky boom")
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{"t3_r": scored("t3_r", 5)})

	runID, _ := repo.CreateScoutRun(ctx, "p1", "all")
	res, err := runner.runAll(ctx, "p1", runID, socialAll)
	if err != nil {
		t.Fatalf("runAll should not hard-fail on one platform: %v", err)
	}
	if res.PostsFound != 1 {
		t.Fatalf("reddit posts should still land: want 1, got %d", res.PostsFound)
	}
	run, _ := repo.GetScoutRun(ctx, "p1", runID)
	if run.Status != "completed" {
		t.Fatalf("run status = %s, want completed", run.Status)
	}
	if run.Warnings == nil || !containsStr(*run.Warnings, "bluesky") {
		t.Fatalf("expected a bluesky warning, got %v", run.Warnings)
	}
}

func TestRunAll_CancellationAbortsButKeepsAccumulatedCounts(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()
	mkProject(t, repo, "p1", "research")
	mkQuery(t, repo, "p1", "reddit", "https://www.reddit.com/search?q=x", "pain", true)
	mkQuery(t, repo, "p1", "bluesky", "x", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{{ID: "t3_r", Title: "R", Author: "u", Permalink: "/r/x/1", URL: "https://www.reddit.com/r/x/1"}}, nil
	}
	// Bluesky fetch fails with a context cancellation -> runAll must abort.
	runner.fetchBluesky = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return nil, context.Canceled
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{"t3_r": scored("t3_r", 5)})

	runID, _ := repo.CreateScoutRun(ctx, "p1", "all")
	res, err := runner.runAll(ctx, "p1", runID, socialAll)
	if err != nil {
		t.Fatalf("runAll returns nil error on abort (run marked failed internally): %v", err)
	}
	if res.PostsFound != 1 {
		t.Fatalf("abort must preserve reddit's accumulated PostsFound=1, got %d", res.PostsFound)
	}
	run, _ := repo.GetScoutRun(ctx, "p1", runID)
	if run.Status != "failed" {
		t.Fatalf("cancellation should mark run failed, got %s", run.Status)
	}
}

func TestRunAll_EmptyPoolCompletesZero(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()
	mkProject(t, repo, "p1", "research")

	runID, _ := repo.CreateScoutRun(ctx, "p1", "all")
	res, err := runner.runAll(ctx, "p1", runID, socialAll)
	if err != nil {
		t.Fatalf("runAll: %v", err)
	}
	if res.PostsChecked != 0 || res.PostsFound != 0 {
		t.Fatalf("want 0/0, got %d/%d", res.PostsChecked, res.PostsFound)
	}
	run, _ := repo.GetScoutRun(ctx, "p1", runID)
	if run.Status != "completed" {
		t.Fatalf("empty pool should still complete, got %s", run.Status)
	}
}

func TestRunAllAndReport_TriggersReportWhenRequested(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()
	mkProject(t, repo, "p1", "research")
	mkQuery(t, repo, "p1", "reddit", "https://www.reddit.com/search?q=x", "pain", true)
	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return nil, nil
	}

	var triggeredWith string
	runner.ReportTrigger = func(projectID string) { triggeredWith = projectID }

	runID, _ := repo.CreateScoutRun(ctx, "p1", "all")
	if _, err := runner.runAllAndReport(ctx, "p1", runID, socialAll, true); err != nil {
		t.Fatalf("runAllAndReport: %v", err)
	}
	if triggeredWith != "p1" {
		t.Fatalf("ReportTrigger should fire with projectID, got %q", triggeredWith)
	}
}

func TestRunAllAndReport_SkipsReportWhenNotRequested(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()
	mkProject(t, repo, "p1", "research")
	mkQuery(t, repo, "p1", "reddit", "https://www.reddit.com/search?q=x", "pain", true)
	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return nil, nil
	}
	called := false
	runner.ReportTrigger = func(_ string) { called = true }

	runID, _ := repo.CreateScoutRun(ctx, "p1", "all")
	if _, err := runner.runAllAndReport(ctx, "p1", runID, socialAll, false); err != nil {
		t.Fatalf("runAllAndReport: %v", err)
	}
	if called {
		t.Fatal("ReportTrigger must not fire when generateReport is false")
	}
}
