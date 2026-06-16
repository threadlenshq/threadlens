package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

type fakeDMRepo struct {
	existing  map[string]int
	inserted  map[string][]domain.DMTargetInsert
	insertErr map[string]error
}

func newFakeDMRepo() *fakeDMRepo {
	return &fakeDMRepo{
		existing:  map[string]int{},
		inserted:  map[string][]domain.DMTargetInsert{},
		insertErr: map[string]error{},
	}
}

func (r *fakeDMRepo) CountDMTargets(_ context.Context, postID string) (int, error) {
	return r.existing[postID], nil
}

func (r *fakeDMRepo) InsertDMTargets(_ context.Context, postID string, targets []domain.DMTargetInsert) (int64, error) {
	if err := r.insertErr[postID]; err != nil {
		return 0, err
	}
	r.inserted[postID] = append([]domain.DMTargetInsert{}, targets...)
	return int64(len(targets)), nil
}

type fakeRedditContextFetcher struct {
	contexts map[string]RedditContext
	errors   map[string]error
}

func (f fakeRedditContextFetcher) FetchRedditContext(_ context.Context, postURL string) (RedditContext, error) {
	if err := f.errors[postURL]; err != nil {
		return RedditContext{}, err
	}
	return f.contexts[postURL], nil
}

type fakeBlueskyReplyFetcher struct {
	replies map[string][]BlueskyReply
	errors  map[string]error
}

func (f fakeBlueskyReplyFetcher) FetchBlueskyReplies(_ context.Context, postURI string) ([]BlueskyReply, error) {
	if err := f.errors[postURI]; err != nil {
		return nil, err
	}
	return f.replies[postURI], nil
}

func marketingProject() domain.Project {
	return domain.Project{ID: "p1", Name: "Marketing", Mode: "marketing"}
}

func researchProject() domain.Project {
	return domain.Project{ID: "p1", Name: "Research", Mode: "research"}
}

func redditPost(id string, finalScore float64) domain.Post {
	return domain.Post{
		ID:          id,
		ProjectID:   "p1",
		Platform:    "reddit",
		Title:       "Need a better workflow",
		Body:        "I am frustrated and looking for a tool to solve this",
		Author:      "post_author",
		URL:         "https://www.reddit.com/r/test/comments/" + id,
		FinalScore:  finalScore,
		FilterState: domain.FilterStateVisible,
		CreatedAt:   strPtr("2026-06-01T12:00:00Z"),
		NumComments: domain.IntPtr(6),
		RedditScore: domain.IntPtr(25),
		Status:      "new",
		Why:         "strong pain signal",
	}
}

func blueskyPost(id string, finalScore float64) domain.Post {
	uri := "at://did:plc:abc/app.bsky.feed.post/" + id
	return domain.Post{
		ID:          uri,
		ProjectID:   "p1",
		Platform:    "bluesky",
		Title:       "Need help with outreach",
		Body:        "Does anyone know a tool for this problem?",
		Author:      "author.bsky.social",
		URL:         "https://bsky.app/profile/author.bsky.social/post/" + id,
		FinalScore:  finalScore,
		FilterState: domain.FilterStateVisible,
		CreatedAt:   strPtr("2026-06-01T12:00:00Z"),
		LikeCount:   domain.IntPtr(10),
		ReplyCount:  domain.IntPtr(5),
		Status:      "new",
		Why:         "strong pain signal",
		BlueskyURI:  &uri,
	}
}

func strPtr(s string) *string { return &s }

func TestDMTargetGeneratorMarketingRedditGeneratesTopThreeTargets(t *testing.T) {
	repo := newFakeDMRepo()
	post := redditPost("t3_good", 8)
	fetcher := fakeRedditContextFetcher{contexts: map[string]RedditContext{
		post.URL: {
			FullBody: "I am frustrated and looking for a tool to solve this",
			TopComments: []RedditComment{
				{Author: "direct_request", Body: "Can someone recommend a tool? I need this now", Score: 9},
				{Author: "automoderator", Body: "removed", Score: 100},
				{Author: "low_value", Body: "same", Score: 1},
				{Author: "duplicate", Body: "I am struggling with this problem", Score: 3},
				{Author: "Duplicate", Body: "I need a better workaround please", Score: 6},
			},
		},
	}}
	generator := NewDMTargetGenerator(repo, fetcher, nil, nil)

	warnings := generator.Generate(context.Background(), marketingProject(), "reddit", []domain.Post{post})

	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	targets := repo.inserted[post.ID]
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d: %#v", len(targets), targets)
	}
	for _, target := range targets {
		if target.DMStatus != "new" {
			t.Fatalf("dm_status = %q, want new", target.DMStatus)
		}
		if target.Username == "automoderator" {
			t.Fatal("automoderator should be skipped")
		}
		if strings.TrimSpace(target.Signal) == "" || strings.TrimSpace(target.Context) == "" || strings.TrimSpace(target.Approach) == "" {
			t.Fatalf("target fields must be populated: %#v", target)
		}
	}
	if targets[0].IntentScore < targets[1].IntentScore || targets[1].IntentScore < targets[2].IntentScore {
		t.Fatalf("targets are not sorted by descending intent score: %#v", targets)
	}
	seenDuplicate := 0
	for _, target := range targets {
		if strings.EqualFold(target.Username, "duplicate") {
			seenDuplicate++
		}
	}
	if seenDuplicate > 1 {
		t.Fatalf("duplicate candidate was not deduplicated: %#v", targets)
	}
}

func TestDMTargetGeneratorSkipsIneligiblePostsAndPreservesExistingTargets(t *testing.T) {
	repo := newFakeDMRepo()
	repo.existing["t3_existing"] = 1
	generator := NewDMTargetGenerator(repo, fakeRedditContextFetcher{}, nil, nil)
	filtered := redditPost("t3_filtered", 8)
	filtered.FilterState = domain.FilterStateFiltered

	warnings := generator.Generate(context.Background(), marketingProject(), "reddit", []domain.Post{
		redditPost("t3_low", 4.99),
		redditPost("t3_existing", 9),
		filtered,
	})

	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for skipped posts, got %v", warnings)
	}
	if len(repo.inserted) != 0 {
		t.Fatalf("expected no inserts, got %#v", repo.inserted)
	}

	repo = newFakeDMRepo()
	generator = NewDMTargetGenerator(repo, fakeRedditContextFetcher{}, nil, nil)
	generator.Generate(context.Background(), researchProject(), "reddit", []domain.Post{redditPost("t3_research", 8)})
	generator.Generate(context.Background(), marketingProject(), "google", []domain.Post{redditPost("t3_google", 8)})
	if len(repo.inserted) != 0 {
		t.Fatalf("research mode and google platform must not generate targets: %#v", repo.inserted)
	}
}

func TestDMTargetGeneratorFetchFailureInsertsAuthorOnlyAndWarns(t *testing.T) {
	repo := newFakeDMRepo()
	post := redditPost("t3_fetch_error", 7)
	generator := NewDMTargetGenerator(repo, fakeRedditContextFetcher{errors: map[string]error{post.URL: errors.New("reddit unavailable")}}, nil, nil)

	warnings := generator.Generate(context.Background(), marketingProject(), "reddit", []domain.Post{post})

	if len(warnings) != 1 || !strings.Contains(warnings[0], "reddit context fetch failed") {
		t.Fatalf("expected fetch warning mentioning reddit context fetch failed, got %v", warnings)
	}
	targets := repo.inserted[post.ID]
	if len(targets) != 1 || targets[0].Username != "post_author" {
		t.Fatalf("expected only author target, got %#v", targets)
	}
}

func TestDMTargetGeneratorInsertErrorDoesNotStopOtherPosts(t *testing.T) {
	repo := newFakeDMRepo()
	repo.insertErr["t3_bad"] = errors.New("database locked")
	bad := redditPost("t3_bad", 8)
	good := redditPost("t3_good", 8)
	fetcher := fakeRedditContextFetcher{contexts: map[string]RedditContext{
		bad.URL:  {TopComments: []RedditComment{{Author: "a", Body: "I need this", Score: 1}, {Author: "b", Body: "I want this", Score: 1}}},
		good.URL: {TopComments: []RedditComment{{Author: "c", Body: "I need this", Score: 1}, {Author: "d", Body: "I want this", Score: 1}}},
	}}
	generator := NewDMTargetGenerator(repo, fetcher, nil, nil)

	warnings := generator.Generate(context.Background(), marketingProject(), "reddit", []domain.Post{bad, good})

	if len(warnings) != 1 || !strings.Contains(warnings[0], "database locked") {
		t.Fatalf("expected insert warning, got %v", warnings)
	}
	if len(repo.inserted[good.ID]) != 3 {
		t.Fatalf("expected good post to still receive 3 targets, got %#v", repo.inserted[good.ID])
	}
}

func TestDMTargetGeneratorBlueskyUsesRepliesAndStableTieBreak(t *testing.T) {
	repo := newFakeDMRepo()
	post := blueskyPost("abc", 6)
	fetcher := fakeBlueskyReplyFetcher{replies: map[string][]BlueskyReply{
		post.ID: {
			{AuthorHandle: "zeta.bsky.social", Text: "I need a tool for this", LikeCount: 0, IndexedAt: "2026-06-01T12:00:00Z"},
			{AuthorHandle: "alpha.bsky.social", Text: "I need a tool for this", LikeCount: 0, IndexedAt: "2026-06-01T12:00:00Z"},
		},
	}}
	generator := NewDMTargetGenerator(repo, nil, fetcher, nil)

	generator.Generate(context.Background(), marketingProject(), "bluesky", []domain.Post{post})

	targets := repo.inserted[post.ID]
	if len(targets) != 3 {
		t.Fatalf("expected author plus two replies, got %#v", targets)
	}
	if targets[0].Username != "author.bsky.social" {
		t.Fatalf("author should win same-score source-priority tie, got %#v", targets)
	}
	if targets[1].Username != "alpha.bsky.social" || targets[2].Username != "zeta.bsky.social" {
		t.Fatalf("reply tie should sort by username, got %#v", targets)
	}
}

func TestDMCandidateFilterHardExcludesClearBotsAutomationAndSpam(t *testing.T) {
	filter := DMCandidateFilter{}
	cases := []DMCandidateProfile{
		{Platform: "reddit", Username: "[deleted]", Text: "I need help"},
		{Platform: "reddit", Username: "AutoModerator", Text: "I need help"},
		{Platform: "bluesky", Username: "updates-bot", Text: "I need help"},
		{Platform: "bluesky", Username: "rss_feed_daily", Text: "I need help"},
		{Platform: "reddit", Username: "human", Text: "I am a bot that mirrors posts automatically"},
		{Platform: "reddit", Username: "human", Text: "Crypto airdrop giveaway referral link claim now"},
		{Platform: "reddit", Username: "human", DisplayName: "Mirror Bot", Text: "I need help"},
	}

	for _, tc := range cases {
		result := filter.Evaluate(tc)
		if result.Outcome != DMCandidateExclude {
			t.Fatalf("Evaluate(%#v) outcome = %q, want %q", tc, result.Outcome, DMCandidateExclude)
		}
	}
}

func TestDMCandidateFilterDoesNotExcludeGenericBusinessTopics(t *testing.T) {
	filter := DMCandidateFilter{}
	profile := DMCandidateProfile{
		Platform:    "bluesky",
		Username:    "founder.bsky.social",
		DisplayName: "Startup Automation Consultant",
		Bio:         "I write about marketing software and crypto payment tooling.",
		Text:        "I need a better workflow for tracking replies from prospects.",
	}

	result := filter.Evaluate(profile)

	if result.Outcome == DMCandidateExclude {
		t.Fatalf("generic business terms must not exclude candidate: %#v", result)
	}
}

func TestDMCandidateFilterMissingMetadataIsNeutral(t *testing.T) {
	filter := DMCandidateFilter{}
	profile := DMCandidateProfile{Platform: "reddit", Username: "real_user", Text: "I need a better tool for this workflow"}

	result := filter.Evaluate(profile)

	if result.Outcome != DMCandidateAllow {
		t.Fatalf("missing display name, bio, age, timestamps, and engagement should be neutral, got %#v", result)
	}
	if result.Penalty != 0 {
		t.Fatalf("missing metadata penalty = %v, want 0", result.Penalty)
	}
}

func TestDMCandidateFilterSoftPenaltiesRequireReliableSignals(t *testing.T) {
	filter := DMCandidateFilter{}
	newAccountAt := time.Now().UTC().Add(-12 * time.Hour)
	cases := []DMCandidateProfile{
		{Platform: "bluesky", Username: "newuser.bsky.social", Text: "Interesting update", ProfileCreatedAt: &newAccountAt},
		{Platform: "reddit", Username: "promo_person", DisplayName: "Promo Writer", Text: "I need a tool for organizing outreach"},
		{Platform: "reddit", Username: "link_person", Text: "This is useful https://a.example https://b.example #sales @team"},
	}

	for _, tc := range cases {
		result := filter.Evaluate(tc)
		if result.Outcome != DMCandidatePenalize {
			t.Fatalf("Evaluate(%#v) outcome = %q, want %q", tc, result.Outcome, DMCandidatePenalize)
		}
		if result.Penalty <= 0 {
			t.Fatalf("Evaluate(%#v) penalty = %v, want positive", tc, result.Penalty)
		}
	}
}

func TestDMTargetGeneratorInsertsFewerThanThreeWhenOnlyValidCandidatesExist(t *testing.T) {
	repo := newFakeDMRepo()
	post := redditPost("t3_two", 7)
	fetcher := fakeRedditContextFetcher{contexts: map[string]RedditContext{
		post.URL: {TopComments: []RedditComment{{Author: "helper", Body: "I need this too", Score: 2}, {Author: "automoderator", Body: "removed", Score: 100}}},
	}}
	generator := NewDMTargetGenerator(repo, fetcher, nil, nil)

	warnings := generator.Generate(context.Background(), marketingProject(), "reddit", []domain.Post{post})

	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	targets := repo.inserted[post.ID]
	if len(targets) != 2 {
		t.Fatalf("expected author plus one valid commenter, got %#v", targets)
	}
}

func TestDMTargetGeneratorPreservesExistingPartialTargetSet(t *testing.T) {
	repo := newFakeDMRepo()
	repo.existing["t3_partial"] = 2
	post := redditPost("t3_partial", 9)
	fetcher := fakeRedditContextFetcher{contexts: map[string]RedditContext{
		post.URL: {TopComments: []RedditComment{{Author: "a", Body: "I need this", Score: 2}, {Author: "b", Body: "I want this", Score: 2}, {Author: "c", Body: "I need this too", Score: 2}}},
	}}
	generator := NewDMTargetGenerator(repo, fetcher, nil, nil)

	generator.Generate(context.Background(), marketingProject(), "reddit", []domain.Post{post})

	if len(repo.inserted) != 0 {
		t.Fatalf("existing targets must be preserved without top-up, got inserts %#v", repo.inserted)
	}
}
