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
	// posts to return from ListEligibleDMPosts (keyed by projectID)
	posts map[string][]domain.Post

	// Calls recorded for InsertDMTarget
	inserted []dmInsertCall

	// If set, InsertDMTarget returns this error for the matching postID
	insertErrFor map[string]error

	// existingTargets is queried by ListExistingDMTargets (keyed by postID).
	// Tests that exercise the "skip posts that already have targets" path must
	// populate this map; the generator must not rely solely on Post.DMTargets.
	existingTargets map[string][]domain.DMTarget
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

// ListExistingDMTargets returns whatever the test has pre-populated in
// existingTargets for the given post, mirroring the repo query the generator
// must make before deciding whether to process a post.
func (f *fakeDMRepo) ListExistingDMTargets(ctx context.Context, postID string) ([]domain.DMTarget, error) {
	return f.existingTargets[postID], nil
}

// ---------------------------------------------------------------------------
// Fake Reddit context fetcher
// ---------------------------------------------------------------------------

type fakeRedditContextFetcher struct {
	// Map from post URL → ordered list of comment authors to return
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
		ID:         id,
		Platform:   "reddit",
		Author:     author,
		URL:        url,
		Title:      "Test post " + id,
		Body:       "Body of " + id,
		PostScore:  score,
		FinalScore: score,
		Subreddit:  &sub,
		Status:     "new",
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

// insertedUsernames returns the set of usernames across all recorded inserts
// for the given postID, for convenient membership checks.
func insertedUsernames(calls []dmInsertCall, postID string) map[string]bool {
	out := make(map[string]bool)
	for _, c := range calls {
		if c.PostID == postID {
			out[c.Target.Username] = true
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestDMTargetGeneratorMarketingRedditGeneratesTopThreeTargets verifies that
// a marketing-mode project with a reddit post produces DM targets from the
// candidate pool of post author PLUS commenters, capped at three.
//
// Design: 2 commenters + 1 author = 3 candidates total. Asserting that all
// three appear in the inserted set — including the author — confirms the
// generator treats the author as part of the candidate pool.
func TestDMTargetGeneratorMarketingRedditGeneratesTopThreeTargets(t *testing.T) {
	const postURL = "https://reddit.com/r/golang/comments/abc/"

	repo := &fakeDMRepo{
		posts: map[string][]domain.Post{
			"proj-marketing": {
				redditPost("post-1", "alice", postURL, 0.9),
			},
		},
	}
	redditFetcher := &fakeRedditContextFetcher{
		commentAuthors: map[string][]string{
			postURL: {"bob", "carol"}, // 2 commenters; author "alice" should be the 3rd candidate
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

	names := insertedUsernames(repo.inserted, "post-1")

	// Author must be in the candidate pool; with only 2 commenters, all three
	// (alice, bob, carol) should be inserted.
	if !names["alice"] {
		t.Error("expected post author 'alice' to be included as a DM target candidate")
	}
	if !names["bob"] {
		t.Error("expected commenter 'bob' to be included as a DM target candidate")
	}
	if !names["carol"] {
		t.Error("expected commenter 'carol' to be included as a DM target candidate")
	}
	if len(repo.inserted) != 3 {
		t.Errorf("expected exactly 3 inserted targets (author + 2 commenters), got %d", len(repo.inserted))
	}
}

// TestDMTargetGeneratorSkipsIneligiblePostsAndPreservesExistingTargets verifies
// that the generator queries the repo for existing targets and skips posts that
// already have them, while still processing posts that have none.
//
// Design: existing-target state is expressed via fakeDMRepo.existingTargets
// (the repo contract), NOT via a pre-populated Post.DMTargets field.  This
// ensures the generator is actually calling ListExistingDMTargets rather than
// relying on an in-memory cache.
func TestDMTargetGeneratorSkipsIneligiblePostsAndPreservesExistingTargets(t *testing.T) {
	const urlExisting = "https://reddit.com/r/golang/comments/xyz/"
	const urlNew = "https://reddit.com/r/golang/comments/new/"

	repo := &fakeDMRepo{
		posts: map[string][]domain.Post{
			"proj-marketing": {
				// post-existing: clean struct — no DMTargets pre-loaded
				redditPost("post-existing", "alice", urlExisting, 0.8),
				redditPost("post-new", "bob", urlNew, 0.7),
			},
		},
		// Existing target expressed through the repo query path only
		existingTargets: map[string][]domain.DMTarget{
			"post-existing": {
				{Username: "existinguser", PostID: "post-existing"},
			},
		},
	}
	redditFetcher := &fakeRedditContextFetcher{
		commentAuthors: map[string][]string{
			urlNew: {"carol"},
		},
	}
	blueskyFetcher := &fakeBlueskyReplyFetcher{}

	gen := NewDMTargetGenerator(repo, redditFetcher, blueskyFetcher)
	_, err := gen.Generate(context.Background(), marketingProject())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// post-existing already has targets in the repo → must not be touched
	for _, call := range repo.inserted {
		if call.PostID == "post-existing" {
			t.Error("generator must not insert targets for post-existing: repo already has targets for it")
		}
	}

	// post-new has no existing targets → must be processed
	if len(insertedUsernames(repo.inserted, "post-new")) == 0 {
		t.Error("expected at least one target inserted for post-new")
	}
}

// TestDMTargetGeneratorFetchFailureInsertsAuthorOnlyAndWarns verifies that when
// the context fetcher fails, the generator falls back to inserting just the post
// author as the sole target and records a warning rather than failing hard.
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

	// Author-only fallback: exactly one target inserted and it must be the author
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
	const urlFail = "https://reddit.com/r/golang/comments/fail/"
	const urlOK = "https://reddit.com/r/golang/comments/ok/"

	repo := &fakeDMRepo{
		posts: map[string][]domain.Post{
			"proj-marketing": {
				redditPost("post-fail", "alice", urlFail, 0.9),
				redditPost("post-ok", "bob", urlOK, 0.8),
			},
		},
		insertErrFor: map[string]error{
			"post-fail": errors.New("db error"),
		},
	}
	redditFetcher := &fakeRedditContextFetcher{
		commentAuthors: map[string][]string{
			urlFail: {"carol"},
			urlOK:   {"dave"},
		},
	}
	blueskyFetcher := &fakeBlueskyReplyFetcher{}

	gen := NewDMTargetGenerator(repo, redditFetcher, blueskyFetcher)
	warnings, err := gen.Generate(context.Background(), marketingProject())

	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if len(warnings) == 0 {
		t.Error("expected a warning for insert failure on post-fail")
	}

	// post-ok must still be processed despite the failure on post-fail
	if len(insertedUsernames(repo.inserted, "post-ok")) == 0 {
		t.Error("expected post-ok to be processed despite post-fail insert error")
	}
}

// TestDMTargetGeneratorBlueskyUsesReplyFetcherAndIncludesAuthor verifies that
// Bluesky posts source candidates from the reply fetcher (not the Reddit
// fetcher), that the post author is included in the candidate pool, and that
// all candidates within the cap are inserted.
//
// Design: 2 reply authors + 1 author = 3 candidates. All three must appear in
// the inserted set.
func TestDMTargetGeneratorBlueskyUsesReplyFetcherAndIncludesAuthor(t *testing.T) {
	const uri = "at://did:plc:abc123/app.bsky.feed.post/xyz"

	repo := &fakeDMRepo{
		posts: map[string][]domain.Post{
			"proj-marketing": {
				blueskyPost("bsky-1", "alice", uri, 0.85),
			},
		},
	}
	// Reddit fetcher is left empty; a correct implementation should not call it
	// for Bluesky posts, so no commentAuthors are registered.
	redditFetcher := &fakeRedditContextFetcher{}
	blueskyFetcher := &fakeBlueskyReplyFetcher{
		replyAuthors: map[string][]string{
			uri: {"bob", "carol"}, // 2 reply authors; "alice" (author) is the 3rd candidate
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

	names := insertedUsernames(repo.inserted, "bsky-1")

	// All three candidates (author + 2 reply authors) must be present
	if !names["alice"] {
		t.Error("expected post author 'alice' to be included as a DM target candidate")
	}
	if !names["bob"] {
		t.Error("expected reply author 'bob' to be included as a DM target candidate")
	}
	if !names["carol"] {
		t.Error("expected reply author 'carol' to be included as a DM target candidate")
	}
	if len(repo.inserted) != 3 {
		t.Errorf("expected exactly 3 inserted targets (author + 2 reply authors), got %d", len(repo.inserted))
	}
}
