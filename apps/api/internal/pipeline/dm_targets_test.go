package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

// ---------------------------------------------------------------------------
// Fake repository
// ---------------------------------------------------------------------------

type fakeDMRepo struct {
	// posts to return from ListPostsByProject (keyed by projectID)
	posts map[string][]domain.Post

	// Calls recorded for InsertDMTarget
	inserted []dmInsertCall

	// If set, InsertDMTarget returns this error for the matching postID
	insertErrFor map[string]error
}

type dmInsertCall struct {
	PostID string
	Target domain.DMTargetInsert
}

func (f *fakeDMRepo) ListEligibleDMPosts(ctx context.Context, projectID string) ([]domain.Post, error) {
	return f.posts[projectID], nil
}

func (f *fakeDMRepo) InsertDMTarget(ctx context.Context, postID string, t domain.DMTargetInsert) (domain.DMTarget, error) {
	if f.insertErrFor != nil {
		if err, ok := f.insertErrFor[postID]; ok {
			return domain.DMTarget{}, err
		}
	}
	f.inserted = append(f.inserted, dmInsertCall{PostID: postID, Target: t})
	return domain.DMTarget{
		PostID:   postID,
		Username: t.Username,
		Signal:   t.Signal,
	}, nil
}

func (f *fakeDMRepo) ListExistingDMTargets(ctx context.Context, postID string) ([]domain.DMTarget, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Fake Reddit context fetcher
// ---------------------------------------------------------------------------

type fakeRedditContextFetcher struct {
	// Map from post URL → list of comment authors to return
	commentAuthors map[string][]string
	fetchErr       error
}

func (f *fakeRedditContextFetcher) FetchCommentAuthors(ctx context.Context, postURL string) ([]string, error) {
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	return f.commentAuthors[postURL], nil
}

// ---------------------------------------------------------------------------
// Fake Bluesky reply fetcher
// ---------------------------------------------------------------------------

type fakeBlueskyReplyFetcher struct {
	// Map from bluesky URI → list of reply authors to return
	replyAuthors map[string][]string
	fetchErr     error
}

func (f *fakeBlueskyReplyFetcher) FetchReplyAuthors(ctx context.Context, uri string) ([]string, error) {
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	return f.replyAuthors[uri], nil
}

// ---------------------------------------------------------------------------
// Helper builders
// ---------------------------------------------------------------------------

func marketingProject() domain.Project {
	return domain.Project{
		ID:   "proj-marketing",
		Name: "Marketing Project",
		Mode: "marketing",
	}
}

func researchProject() domain.Project {
	return domain.Project{
		ID:   "proj-research",
		Name: "Research Project",
		Mode: "research",
	}
}

func redditPost(id, author, url string, score float64) domain.Post {
	sub := "golang"
	return domain.Post{
		ID:       id,
		Platform: "reddit",
		Author:   author,
		URL:      url,
		Title:    "Test post " + id,
		Body:     "Body of " + id,
		PostScore: score,
		FinalScore: score,
		Subreddit: &sub,
		Status:   "new",
	}
}

func blueskyPost(id, author, uri string, score float64) domain.Post {
	return domain.Post{
		ID:         id,
		Platform:   "bluesky",
		Author:     author,
		URL:        "https://bsky.app/profile/" + author + "/post/abc",
		BlueskyURI: strPtr(uri),
		Title:      "Bluesky post " + id,
		Body:       "Body of " + id,
		PostScore:  score,
		FinalScore: score,
		Status:     "new",
	}
}

func strPtr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestDMTargetGeneratorMarketingRedditGeneratesTopThreeTargets verifies that
// a marketing-mode project with reddit posts produces up to three DM targets
// per post (capped at 3 even when more commenters are available).
func TestDMTargetGeneratorMarketingRedditGeneratesTopThreeTargets(t *testing.T) {
	repo := &fakeDMRepo{
		posts: map[string][]domain.Post{
			"proj-marketing": {
				redditPost("post-1", "alice", "https://reddit.com/r/golang/comments/abc/", 0.9),
			},
		},
	}
	redditFetcher := &fakeRedditContextFetcher{
		commentAuthors: map[string][]string{
			"https://reddit.com/r/golang/comments/abc/": {"bob", "carol", "dave", "eve"},
		},
	}
	blueskyFetcher := &fakeBlueskyReplyFetcher{}

	gen := NewDMTargetGenerator(repo, redditFetcher, blueskyFetcher)
	warnings, err := gen.Generate(context.Background(), marketingProject())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got: %v", warnings)
	}
	// Expect exactly 3 targets for post-1 (top 3 of 4 commenters)
	if len(repo.inserted) != 3 {
		t.Errorf("expected 3 inserted targets, got %d", len(repo.inserted))
	}
	for _, call := range repo.inserted {
		if call.PostID != "post-1" {
			t.Errorf("expected postID=post-1, got %s", call.PostID)
		}
	}
}

// TestDMTargetGeneratorSkipsIneligiblePostsAndPreservesExistingTargets verifies
// that posts already having DM targets are skipped while posts without targets
// are still processed normally.
func TestDMTargetGeneratorSkipsIneligiblePostsAndPreservesExistingTargets(t *testing.T) {
	postWithTargets := redditPost("post-existing", "alice", "https://reddit.com/r/golang/comments/xyz/", 0.8)
	postWithTargets.DMTargets = []domain.DMTarget{
		{Username: "existinguser", PostID: "post-existing"},
	}
	postWithoutTargets := redditPost("post-new", "bob", "https://reddit.com/r/golang/comments/new/", 0.7)

	repo := &fakeDMRepo{
		posts: map[string][]domain.Post{
			"proj-marketing": {
				postWithTargets,
				postWithoutTargets,
			},
		},
	}
	redditFetcher := &fakeRedditContextFetcher{
		commentAuthors: map[string][]string{
			"https://reddit.com/r/golang/comments/new/": {"carol"},
		},
	}
	blueskyFetcher := &fakeBlueskyReplyFetcher{}

	gen := NewDMTargetGenerator(repo, redditFetcher, blueskyFetcher)
	_, err := gen.Generate(context.Background(), marketingProject())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// post-existing already has targets, so only post-new should produce inserts
	for _, call := range repo.inserted {
		if call.PostID == "post-existing" {
			t.Error("generator should not insert targets for post-existing which already has DM targets")
		}
	}
	found := false
	for _, call := range repo.inserted {
		if call.PostID == "post-new" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one target inserted for post-new")
	}
}

// TestDMTargetGeneratorFetchFailureInsertsAuthorOnlyAndWarns verifies that when
// the context fetcher fails, the generator falls back to inserting just the post
// author as a target and records a warning.
func TestDMTargetGeneratorFetchFailureInsertsAuthorOnlyAndWarns(t *testing.T) {
	repo := &fakeDMRepo{
		posts: map[string][]domain.Post{
			"proj-marketing": {
				redditPost("post-1", "alice", "https://reddit.com/r/golang/comments/abc/", 0.9),
			},
		},
	}
	redditFetcher := &fakeRedditContextFetcher{
		fetchErr: errors.New("network error"),
	}
	blueskyFetcher := &fakeBlueskyReplyFetcher{}

	gen := NewDMTargetGenerator(repo, redditFetcher, blueskyFetcher)
	warnings, err := gen.Generate(context.Background(), marketingProject())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) == 0 {
		t.Error("expected at least one warning for fetch failure")
	}
	// Should still insert the post author as the sole target
	if len(repo.inserted) != 1 {
		t.Fatalf("expected 1 inserted target (author fallback), got %d", len(repo.inserted))
	}
	if repo.inserted[0].Target.Username != "alice" {
		t.Errorf("expected author 'alice' as fallback target, got %q", repo.inserted[0].Target.Username)
	}
}

// TestDMTargetGeneratorInsertErrorDoesNotStopOtherPosts verifies that an insert
// failure for one post does not prevent processing of subsequent posts.
func TestDMTargetGeneratorInsertErrorDoesNotStopOtherPosts(t *testing.T) {
	repo := &fakeDMRepo{
		posts: map[string][]domain.Post{
			"proj-marketing": {
				redditPost("post-fail", "alice", "https://reddit.com/r/golang/comments/fail/", 0.9),
				redditPost("post-ok", "bob", "https://reddit.com/r/golang/comments/ok/", 0.8),
			},
		},
		insertErrFor: map[string]error{
			"post-fail": errors.New("db error"),
		},
	}
	redditFetcher := &fakeRedditContextFetcher{
		commentAuthors: map[string][]string{
			"https://reddit.com/r/golang/comments/fail/": {"carol"},
			"https://reddit.com/r/golang/comments/ok/":  {"dave"},
		},
	}
	blueskyFetcher := &fakeBlueskyReplyFetcher{}

	gen := NewDMTargetGenerator(repo, redditFetcher, blueskyFetcher)
	warnings, err := gen.Generate(context.Background(), marketingProject())

	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if len(warnings) == 0 {
		t.Error("expected a warning for insert failure")
	}
	// post-ok should still be processed
	found := false
	for _, call := range repo.inserted {
		if call.PostID == "post-ok" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected post-ok to be processed despite post-fail insert error")
	}
}

// TestDMTargetGeneratorBlueskyUsesRepliesAndStableTieBreak verifies that
// Bluesky posts use the reply fetcher, and when scores tie, ordering is stable
// (alphabetical by username as tiebreak).
func TestDMTargetGeneratorBlueskyUsesRepliesAndStableTieBreak(t *testing.T) {
	uri := "at://did:plc:abc123/app.bsky.feed.post/xyz"
	repo := &fakeDMRepo{
		posts: map[string][]domain.Post{
			"proj-marketing": {
				blueskyPost("bsky-1", "alice", uri, 0.85),
			},
		},
	}
	redditFetcher := &fakeRedditContextFetcher{}
	blueskyFetcher := &fakeBlueskyReplyFetcher{
		replyAuthors: map[string][]string{
			uri: {"zebra", "apple", "mango"},
		},
	}

	gen := NewDMTargetGenerator(repo, redditFetcher, blueskyFetcher)
	warnings, err := gen.Generate(context.Background(), marketingProject())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(repo.inserted) != 3 {
		t.Fatalf("expected 3 inserted targets, got %d", len(repo.inserted))
	}
	// Tiebreak: alphabetical by username → apple, mango, zebra
	names := make([]string, len(repo.inserted))
	for i, c := range repo.inserted {
		names[i] = c.Target.Username
	}
	if names[0] != "apple" || names[1] != "mango" || names[2] != "zebra" {
		t.Errorf("expected stable alphabetical tiebreak order [apple mango zebra], got %v", names)
	}
}
