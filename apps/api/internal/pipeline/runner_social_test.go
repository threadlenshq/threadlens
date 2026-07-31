package pipeline

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// --- helpers ---

func newTestRunner(t *testing.T) (*Runner, *repository.Repository) {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	runner := NewRunner(repo, nil)
	return runner, repo
}

func mkProject(t *testing.T, repo *repository.Repository, id string, mode string) domain.Project {
	t.Helper()
	p, err := repo.CreateProject(context.Background(), domain.Project{
		ID:   id,
		Name: id,
		Mode: mode,
	})
	if err != nil {
		t.Fatalf("mkProject: %v", err)
	}
	return p
}

func mkQuery(t *testing.T, repo *repository.Repository, projectID, platform, queryURL, angle string, enabled bool) domain.Query {
	t.Helper()
	q, err := repo.CreateQuery(context.Background(), projectID, platform, queryURL, angle, enabled)
	if err != nil {
		t.Fatalf("mkQuery: %v", err)
	}
	return q
}

func fakeScorer(scores map[string]ScoredPost) func(context.Context, []ScoringPost, []string, *string, *string, func(int, int)) (ScoreResult, error) {
	return func(_ context.Context, posts []ScoringPost, _ []string, _ *string, _ *string, _ func(int, int)) (ScoreResult, error) {
		var out []ScoredPost
		for _, p := range posts {
			if s, ok := scores[p.ID]; ok {
				out = append(out, s)
			}
		}
		return ScoreResult{
			Scores: out,
			Stats: ScoreStats{
				TotalBatches: 1,
				TotalScored:  len(out),
				TotalPosts:   len(posts),
			},
		}, nil
	}
}

func scored(id string, score float64) ScoredPost {
	why := "test"
	return ScoredPost{ID: id, PostScore: score, Why: why, EngagementType: "product"}
}

// SocialRunnerNoEnabledQueries: run completes with 0/0 when no enabled queries exist.
func TestSocialRunnerNoEnabledQueries(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	// disabled query - should not be used
	mkQuery(t, repo, "proj1", "reddit", "https://reddit.com/q", "pain", false)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		t.Fatal("fetchReddit should not be called when no enabled queries")
		return nil, nil
	}

	runID, err := repo.CreateScoutRun(ctx, "proj1", "reddit")
	if err != nil {
		t.Fatal(err)
	}

	res, err := runner.Run(ctx, "proj1", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostsChecked != 0 || res.PostsFound != 0 {
		t.Errorf("expected 0/0, got %d/%d", res.PostsChecked, res.PostsFound)
	}

	run, _ := repo.GetScoutRun(ctx, "proj1", runID)
	if run.Status != "completed" {
		t.Errorf("expected completed, got %s", run.Status)
	}
}

// SocialRunnerSeenFiltering: posts already in seen_posts are excluded.
func TestSocialRunnerSeenFiltering(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "reddit", "https://reddit.com/q", "pain", true)

	// Pre-seed seen_posts with "t3_old"
	if err := repo.MarkSeen(ctx, "proj1", "reddit", []string{"t3_old"}); err != nil {
		t.Fatal(err)
	}

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{
			{ID: "t3_old", Title: "Old post", Author: "u1", Permalink: "/r/x/1", URL: "https://www.reddit.com/r/x/1"},
			{ID: "t3_new", Title: "New post", Author: "u2", Permalink: "/r/x/2", URL: "https://www.reddit.com/r/x/2"},
		}, nil
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{
		"t3_new": scored("t3_new", 5),
	})

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "reddit")
	res, err := runner.Run(ctx, "proj1", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostsChecked != 2 {
		t.Errorf("postsChecked: want 2, got %d", res.PostsChecked)
	}
	if res.PostsFound != 1 {
		t.Errorf("postsFound: want 1, got %d", res.PostsFound)
	}
}

// SocialRunnerPersistsPromotionalRedditAsFiltered: promotional posts are persisted
// as filtered rows while only visible posts count toward postsFound.
func TestSocialRunnerPersistsPromotionalRedditAsFiltered(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "reddit", "https://reddit.com/q", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{
			{ID: "t3_promo", Title: "I built this", Author: "u1", Permalink: "/r/x/1", URL: "/r/x/1"},
			{ID: "t3_real", Title: "Real pain post", Author: "u2", Permalink: "/r/x/2", URL: "https://www.reddit.com/r/x/2"},
		}, nil
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{
		"t3_real": scored("t3_real", 5),
	})

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "reddit")
	res, err := runner.Run(ctx, "proj1", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostsFound != 1 {
		t.Errorf("visible postsFound: want 1, got %d", res.PostsFound)
	}
	filtered, err := repo.ListFilteredFindings(ctx, "proj1", repository.FilteredFindingFilters{Platform: "reddit"}, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].ID != "t3_promo" {
		t.Fatalf("filtered = %#v", filtered.Items)
	}
}

// SocialRunnerDedupeReddit: duplicate author+title posts collapse to highest-scored.
func TestSocialRunnerDedupeReddit(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "reddit", "https://reddit.com/q", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{
			{ID: "t3_a1", Title: "Same title", Author: "alice", Permalink: "/r/x/1", URL: "https://www.reddit.com/r/x/1", Score: 10},
			{ID: "t3_a2", Title: "Same title", Author: "alice", Permalink: "/r/x/2", URL: "https://www.reddit.com/r/x/2", Score: 20},
		}, nil
	}
	// Only one post remains after dedup - whichever ID the dedup keeps.
	// DeduplicatePosts keeps the first-seen key but updates to highest score.
	// The kept entry has ID t3_a1 (first insertion order) with score 20.
	runner.scorePosts = fakeScorer(map[string]ScoredPost{
		"t3_a1": scored("t3_a1", 5),
		"t3_a2": scored("t3_a2", 5),
	})

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "reddit")
	res, err := runner.Run(ctx, "proj1", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After dedup only 1 unique post.
	if res.PostsFound != 1 {
		t.Errorf("want 1 after dedup, got %d", res.PostsFound)
	}
}

// SocialRunnerWarningsText: failed batches produce warnings on the run record.
func TestSocialRunnerWarningsText(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "reddit", "https://reddit.com/q", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{
			{ID: "t3_x", Title: "X", Author: "u1", Permalink: "/r/x/1", URL: "https://www.reddit.com/r/x/1"},
		}, nil
	}
	runner.scorePosts = func(_ context.Context, posts []ScoringPost, _ []string, _ *string, _ *string, _ func(int, int)) (ScoreResult, error) {
		return ScoreResult{
			Scores: []ScoredPost{scored("t3_x", 5)},
			Stats: ScoreStats{
				TotalBatches:  2,
				FailedBatches: 1,
				TotalScored:   1,
				TotalPosts:    1,
				Errors:        []string{"Batch 1: ai timeout"},
			},
		}, nil
	}

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "reddit")
	_, err := runner.Run(ctx, "proj1", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	run, _ := repo.GetScoutRun(ctx, "proj1", runID)
	if run.Warnings == nil {
		t.Fatal("expected warnings, got nil")
	}
	if !containsStr(*run.Warnings, "Scoring: 1/2 batches failed") {
		t.Errorf("unexpected warnings text: %q", *run.Warnings)
	}
}

// SocialRunnerCancellationBeforeStorage: cancelling context before storage
// marks the run as failed and returns 0 posts found.
func TestSocialRunnerCancellationBeforeStorage(t *testing.T) {
	runner, repo := newTestRunner(t)

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "reddit", "https://reddit.com/q", "pain", true)

	// Use a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{
			{ID: "t3_x", Title: "X", Author: "u1", Permalink: "/r/x/1", URL: "https://www.reddit.com/r/x/1"},
		}, nil
	}
	runner.scorePosts = func(innerCtx context.Context, posts []ScoringPost, _ []string, _ *string, _ *string, _ func(int, int)) (ScoreResult, error) {
		// Cancel before we return - simulates cancellation between scoring and storage.
		cancel()
		return ScoreResult{
			Scores: []ScoredPost{scored("t3_x", 5)},
			Stats:  ScoreStats{TotalBatches: 1, TotalScored: 1, TotalPosts: 1},
		}, nil
	}

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "reddit")
	res, _ := runner.Run(ctx, "proj1", "reddit", &runID)

	// Either postsFound == 0 (cancelled path) or the run was marked failed.
	// Both are correct depending on timing; we check postsFound = 0.
	if res.PostsFound != 0 {
		t.Errorf("expected 0 posts found after cancellation, got %d", res.PostsFound)
	}

	run, err := repo.GetScoutRun(context.Background(), "proj1", runID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Status != "failed" {
		t.Fatalf("expected failed status after cancellation, got %q", run.Status)
	}
	if run.Error == nil {
		t.Fatal("expected non-nil run error after cancellation")
	}
	if *run.Error != "Cancelled" && *run.Error != context.Canceled.Error() {
		t.Fatalf("expected cancellation error, got %q", *run.Error)
	}
}

// SocialRunnerRedditInsertFields: verifies post row fields match Express column mapping.
func TestSocialRunnerRedditInsertFields(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "reddit", "https://reddit.com/q", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{{
			ID:          "t3_abc",
			Title:       "Pain title",
			Selftext:    "body text",
			Author:      "redditor",
			Permalink:   "/r/godev/comments/abc/pain_title/",
			Subreddit:   "godev",
			Score:       42,
			NumComments: 7,
			CreatedUTC:  1700000000,
			URL:         "https://www.reddit.com/r/godev/comments/abc",
		}}, nil
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{
		"t3_abc": {ID: "t3_abc", PostScore: 8, Why: "strong match", EngagementType: "product"},
	})

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "reddit")
	_, err := runner.Run(ctx, "proj1", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	posts, err := repo.ListPosts(ctx, "proj1", repository.PostFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	p := posts[0]
	if p.ID != "t3_abc" {
		t.Errorf("id: want t3_abc, got %s", p.ID)
	}
	if p.Platform != "reddit" {
		t.Errorf("platform: want reddit, got %s", p.Platform)
	}
	if p.Title != "Pain title" {
		t.Errorf("title: want 'Pain title', got %s", p.Title)
	}
	if p.Author != "redditor" {
		t.Errorf("author: want redditor, got %s", p.Author)
	}
	if p.URL != "https://www.reddit.com/r/godev/comments/abc/pain_title/" {
		t.Errorf("url: want reddit permalink URL, got %s", p.URL)
	}
	if p.Subreddit == nil || *p.Subreddit != "godev" {
		t.Errorf("subreddit: want godev, got %v", p.Subreddit)
	}
	if p.RedditScore == nil || *p.RedditScore != 42 {
		t.Errorf("reddit_score: want 42, got %v", p.RedditScore)
	}
	if p.NumComments == nil || *p.NumComments != 7 {
		t.Errorf("num_comments: want 7, got %v", p.NumComments)
	}
	if p.PostScore != 8 {
		t.Errorf("post_score: want 8, got %f", p.PostScore)
	}
	if p.Why != "strong match" {
		t.Errorf("why: want 'strong match', got %s", p.Why)
	}
	if p.CreatedAt == nil {
		t.Error("created_at should be set from created_utc")
	}
}

// SocialRunnerBlueskyInsertFields: verifies Bluesky post fields match Express mapping.
func TestSocialRunnerBlueskyInsertFields(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "bluesky", "some query", "pain", true)

	const testURI = "at://did:plc:abc/app.bsky.feed.post/xyz"

	runner.fetchBluesky = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{{
			ID:                testURI,
			CID:               "bafycid123",
			Text:              "Bluesky pain post with some long text here",
			AuthorHandle:      "user.bsky.social",
			AuthorDisplayName: "User",
			LikeCount:         10,
			ReplyCount:        3,
			RepostCount:       1,
			IndexedAt:         "2024-01-01T00:00:00Z",
			PostURL:           "https://bsky.app/profile/user.bsky.social/post/xyz",
			Author:            "user.bsky.social",
		}}, nil
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{
		testURI: {ID: testURI, PostScore: 6, Why: "moderate", EngagementType: "karma"},
	})

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "bluesky")
	_, err := runner.Run(ctx, "proj1", "bluesky", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	posts, err := repo.ListPosts(ctx, "proj1", repository.PostFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	p := posts[0]
	if p.ID != testURI {
		t.Errorf("id: want %s, got %s", testURI, p.ID)
	}
	if p.Platform != "bluesky" {
		t.Errorf("platform: want bluesky, got %s", p.Platform)
	}
	if p.Author != "user.bsky.social" {
		t.Errorf("author: want user.bsky.social, got %s", p.Author)
	}
	if p.URL != "https://bsky.app/profile/user.bsky.social/post/xyz" {
		t.Errorf("url mismatch: %s", p.URL)
	}
	if p.BlueskyURI == nil || *p.BlueskyURI != testURI {
		t.Errorf("bluesky_uri: want %s, got %v", testURI, p.BlueskyURI)
	}
	if p.BlueskyCID == nil || *p.BlueskyCID != "bafycid123" {
		t.Errorf("bluesky_cid: want bafycid123, got %v", p.BlueskyCID)
	}
	if p.LikeCount == nil || *p.LikeCount != 10 {
		t.Errorf("like_count: want 10, got %v", p.LikeCount)
	}
	if p.PostScore != 6 {
		t.Errorf("post_score: want 6, got %f", p.PostScore)
	}
}

// SocialRunnerScoreThreshold: posts with score < 2 are not stored.
func TestSocialRunnerScoreThreshold(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "reddit", "https://reddit.com/q", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{
			{ID: "t3_low", Title: "Low score", Author: "u1", Permalink: "/r/x/1", URL: "https://www.reddit.com/r/x/1"},
			{ID: "t3_high", Title: "High score", Author: "u2", Permalink: "/r/x/2", URL: "https://www.reddit.com/r/x/2"},
		}, nil
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{
		"t3_low":  scored("t3_low", 1),  // below threshold
		"t3_high": scored("t3_high", 5), // above threshold
	})

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "reddit")
	res, err := runner.Run(ctx, "proj1", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostsFound != 1 {
		t.Errorf("want 1 stored (score>=2), got %d", res.PostsFound)
	}

	posts, _ := repo.ListPosts(ctx, "proj1", repository.PostFilters{})
	if len(posts) != 1 || posts[0].ID != "t3_high" {
		t.Errorf("expected only t3_high stored")
	}
}

// SocialRunnerAllBatchesFailed: if all scoring batches fail, seen is NOT marked.
func TestSocialRunnerAllBatchesFailed(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "reddit", "https://reddit.com/q", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{
			{ID: "t3_x", Title: "X", Author: "u1", Permalink: "/r/x/1", URL: "https://www.reddit.com/r/x/1"},
		}, nil
	}
	runner.scorePosts = func(_ context.Context, _ []ScoringPost, _ []string, _ *string, _ *string, _ func(int, int)) (ScoreResult, error) {
		return ScoreResult{
			Scores: nil,
			Stats: ScoreStats{
				TotalBatches:  1,
				FailedBatches: 1,
				TotalScored:   0,
				TotalPosts:    1,
				Errors:        []string{"Batch 1: complete failure"},
			},
		}, nil
	}

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "reddit")
	_, err := runner.Run(ctx, "proj1", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// seen_posts should NOT have t3_x since all batches failed.
	seenIDs, err := repo.SeenIDs(ctx, "proj1", "reddit")
	if err != nil {
		t.Fatal(err)
	}
	if seenIDs["t3_x"] {
		t.Error("t3_x should NOT be in seen_posts when all batches failed")
	}
}

// containsStr checks if s contains substr.
func containsStr(s, substr string) bool {
	return fmt.Sprintf("%s", s) != "" && len(s) >= len(substr) &&
		(s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSocialRunnerGeneratesDMTargetsAfterSuccessfulMarketingRun(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()
	mkProject(t, repo, "dmrun", "marketing")
	mkQuery(t, repo, "dmrun", "reddit", "https://reddit.com/q", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{{
			ID: "t3_dm", Title: "Need tool", Selftext: "I am frustrated and need a tool", Author: "author", Permalink: "/r/x/comments/dm", URL: "https://www.reddit.com/r/x/comments/dm", Score: 12, NumComments: 3,
		}}, nil
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{"t3_dm": scored("t3_dm", 8)})
	runner.dmTargets = NewDMTargetGenerator(repo, fakeRedditContextFetcher{contexts: map[string]RedditContext{
		"https://www.reddit.com/r/x/comments/dm": {TopComments: []RedditComment{{Author: "alice", Body: "Can someone recommend a tool?", Score: 4}, {Author: "bob", Body: "I need this too", Score: 3}}},
	}}, nil, nil)

	runID, _ := repo.CreateScoutRun(ctx, "dmrun", "reddit")
	res, err := runner.Run(ctx, "dmrun", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostsChecked != 1 || res.PostsFound != 1 {
		t.Fatalf("run counts changed by DM generation: checked=%d found=%d", res.PostsChecked, res.PostsFound)
	}
	post, err := repo.GetPost(ctx, "dmrun", "t3_dm")
	if err != nil {
		t.Fatal(err)
	}
	if len(post.DMTargets) != 3 {
		t.Fatalf("expected 3 generated targets, got %#v", post.DMTargets)
	}
	run, _ := repo.GetScoutRun(ctx, "dmrun", runID)
	if run.Status != "completed" {
		t.Fatalf("run status = %s, want completed", run.Status)
	}
	if run.Warnings != nil && containsStr(*run.Warnings, "DM targets") {
		t.Fatalf("did not expect DM warnings, got %q", *run.Warnings)
	}
}

func TestRunSocialDispatchesHackerNews(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "hackernews", "manual invoicing", "pain", true)

	called := false
	runner.fetchHackerNews = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		called = true
		return nil, nil
	}

	runID, err := repo.CreateScoutRun(ctx, "proj1", "hackernews")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "proj1", "hackernews", &runID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected fetchHackerNews to be called for platform=hackernews")
	}
}

func TestRunSocialStoresHackerNewsPosts(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()

	mkProject(t, repo, "proj1", "research")
	mkQuery(t, repo, "proj1", "hackernews", "invoicing", "pain", true)

	runner.fetchHackerNews = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{
			{ID: "hn_1", Title: "Ask HN: invoicing pain", Selftext: "story body", Author: "alice", Score: 50, NumComments: 10, Permalink: "https://news.ycombinator.com/item?id=1", CreatedUTC: 1700000000},
			{ID: "hn_2", Title: "Parent title", Selftext: "comment about invoicing", Author: "bob", Score: 0, Permalink: "https://news.ycombinator.com/item?id=2", CreatedUTC: 1700000500},
		}, nil
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{
		"hn_1": scored("hn_1", 7),
		"hn_2": scored("hn_2", 4),
	})

	runID, _ := repo.CreateScoutRun(ctx, "proj1", "hackernews")
	res, err := runner.Run(ctx, "proj1", "hackernews", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostsChecked != 2 || res.PostsFound != 2 {
		t.Fatalf("want checked=2 found=2, got %d/%d", res.PostsChecked, res.PostsFound)
	}

	posts, err := repo.ListPosts(ctx, "proj1", repository.PostFilters{})
	if err != nil {
		t.Fatal(err)
	}
	var hn1 *domain.Post
	for i := range posts {
		if posts[i].ID == "hn_1" {
			hn1 = &posts[i]
		}
	}
	if hn1 == nil {
		t.Fatal("hn_1 not stored")
	}
	if hn1.Platform != "hackernews" {
		t.Errorf("platform = %q", hn1.Platform)
	}
	if hn1.URL != "https://news.ycombinator.com/item?id=1" {
		t.Errorf("url = %q (should be the permalink, not reddit-prefixed)", hn1.URL)
	}
	if hn1.RedditScore == nil || *hn1.RedditScore != 50 {
		t.Errorf("RedditScore = %v, want 50", hn1.RedditScore)
	}
	if hn1.Subreddit != nil {
		t.Errorf("Subreddit should be nil for HN, got %q", *hn1.Subreddit)
	}
}

func TestSocialRunnerDMTargetWarningsDoNotFailRun(t *testing.T) {
	runner, repo := newTestRunner(t)
	ctx := context.Background()
	mkProject(t, repo, "dmwarn", "marketing")
	mkQuery(t, repo, "dmwarn", "reddit", "https://reddit.com/q", "pain", true)

	runner.fetchReddit = func(_ context.Context, _ []string, _ func(int, int)) ([]FetchedPost, error) {
		return []FetchedPost{{ID: "t3_warn", Title: "Need tool", Selftext: "I need help", Author: "author", Permalink: "/r/x/comments/warn", URL: "https://www.reddit.com/r/x/comments/warn"}}, nil
	}
	runner.scorePosts = fakeScorer(map[string]ScoredPost{"t3_warn": scored("t3_warn", 8)})
	runner.dmTargets = NewDMTargetGenerator(repo, fakeRedditContextFetcher{errors: map[string]error{
		"https://www.reddit.com/r/x/comments/warn": errors.New("context timeout"),
	}}, nil, nil)

	runID, _ := repo.CreateScoutRun(ctx, "dmwarn", "reddit")
	res, err := runner.Run(ctx, "dmwarn", "reddit", &runID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PostsFound != 1 {
		t.Fatalf("postsFound = %d, want 1", res.PostsFound)
	}
	run, _ := repo.GetScoutRun(ctx, "dmwarn", runID)
	if run.Status != "completed" {
		t.Fatalf("run status = %s, want completed", run.Status)
	}
	if run.Warnings == nil || !containsStr(*run.Warnings, "DM targets: t3_warn reddit context fetch failed") {
		t.Fatalf("expected DM warning, got %v", run.Warnings)
	}
}

func TestRunSocialInsertsDMTargetsWithNewStatusAndTimestamp(t *testing.T) {
	// Standalone in-memory DB; same setup as the existing social-runner tests.
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	// Seed a marketing-mode project (the runner only generates DM targets
	// for marketing projects; see dm_targets.go Generate).
	if _, err := db.Exec(
		`INSERT INTO projects (id, name, mode, created_at, updated_at) VALUES ('mkt', 'mkt', 'marketing', datetime('now'), datetime('now'))`,
	); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	// Seed a post with a high enough final_score to qualify for DM target
	// generation. The exact threshold lives in the DMTargetGenerator; a score
	// of 9.0 is well above it.
	if _, err := db.Exec(
		`INSERT INTO posts (id, project_id, platform, title, body, author, url, post_score, final_score, engagement_type, status, found_at, scouted_at)
		 VALUES ('p-mkt', 'mkt', 'reddit', 'Hot post', 'body', 'author', 'https://reddit.com/r/x/comments/1', 9.0, 9.0, 'karma', 'new', datetime('now'), datetime('now'))`,
	); err != nil {
		t.Fatalf("seed post: %v", err)
	}

	// Run the DM target generator directly (no need to exercise the full
	// scout pipeline here — the assertion is about the post-rebuild insert
	// behaviour, which lives in InsertDMTargets).
	targets := []domain.DMTargetInsert{
		{Username: "u1", IntentScore: 7.5, Signal: "s", Context: "c", Approach: "a", DMStatus: "new"},
	}
	if _, err := repo.InsertDMTargets(context.Background(), "p-mkt", targets); err != nil {
		t.Fatalf("InsertDMTargets: %v", err)
	}

	var (
		status  string
		updated string
		inserted int
	)
	if err := db.QueryRow(`SELECT dm_status, dm_status_updated_at FROM dm_targets WHERE post_id = 'p-mkt' AND username = 'u1'`).Scan(&status, &updated); err != nil {
		t.Fatalf("read dm_target: %v", err)
	}
	if status != "new" {
		t.Fatalf("dm_status = %q, want new", status)
	}
	if updated == "" {
		t.Fatalf("dm_status_updated_at must be populated by column default on insert, got empty")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM dm_targets WHERE post_id = 'p-mkt'`).Scan(&inserted); err != nil {
		t.Fatal(err)
	}
	if inserted != 1 {
		t.Fatalf("inserted rows = %d, want 1", inserted)
	}
}
