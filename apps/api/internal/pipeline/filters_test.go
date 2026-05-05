package pipeline

import (
	"strings"
	"testing"
)

// --- IsPromotional tests ---

func TestIsPromotional_FlagsIBuiltTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "I built a tool to track my spending"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsIMadeTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "I made this open source app"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsICreatedTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "I created a new library for React"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsShowHNTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "Show HN: My SaaS app"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsShowRSlashTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "Show r/programming: my project"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsIntroducingTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "Introducing our new product"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsLaunchingTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "Launching my app today"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsReleaseTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "Release 2.0 of my tool"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsBracketVTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "[v1.2] My app update"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsV2DotTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "v2.0 is here"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsV3DotTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "v3.1 released"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsOpenSourceTitle(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "open source tool I made"}) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsGithubInBody(t *testing.T) {
	post := FetchedPost{
		Title:    "Having trouble with my workflow",
		Selftext: "Check out github.com/myrepo for the code",
	}
	if !IsPromotional(post) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsProductHuntInBody(t *testing.T) {
	post := FetchedPost{
		Title:    "Looking for feedback",
		Selftext: "We launched on Product Hunt yesterday",
	}
	if !IsPromotional(post) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsProducthuntDotComInBody(t *testing.T) {
	post := FetchedPost{
		Title:    "Feedback request",
		Selftext: "See producthunt.com/posts/myapp",
	}
	if !IsPromotional(post) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsMyToolInBody(t *testing.T) {
	post := FetchedPost{
		Title:    "Anyone interested?",
		Selftext: "Try my tool to solve this problem",
	}
	if !IsPromotional(post) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsWeBuiltInBody(t *testing.T) {
	post := FetchedPost{
		Title:    "Solving the email problem",
		Selftext: "we built this after facing the same issue",
	}
	if !IsPromotional(post) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsOurProductInBody(t *testing.T) {
	post := FetchedPost{
		Title:    "New release",
		Selftext: "our product is now available for free",
	}
	if !IsPromotional(post) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsFeedbackWelcomeInBody(t *testing.T) {
	post := FetchedPost{
		Title:    "New project",
		Selftext: "feedback welcome from the community",
	}
	if !IsPromotional(post) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_FlagsCheckItOutAtInBody(t *testing.T) {
	post := FetchedPost{
		Title:    "Cool thing",
		Selftext: "check it out at mywebsite.com",
	}
	if !IsPromotional(post) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_AllowsGenuinePainPost(t *testing.T) {
	post := FetchedPost{
		Title:    "Why is AWS billing so confusing?",
		Selftext: "I keep getting surprised by my bill every month. Anyone else?",
	}
	if IsPromotional(post) {
		t.Fatal("expected NOT promotional")
	}
}

func TestIsPromotional_AllowsGenuineQuestion(t *testing.T) {
	post := FetchedPost{
		Title:    "How do you manage multiple environments?",
		Selftext: "Looking for best practices for staging vs prod",
	}
	if IsPromotional(post) {
		t.Fatal("expected NOT promotional")
	}
}

func TestIsPromotional_FlagsExternalLinkPost(t *testing.T) {
	post := FetchedPost{
		Title:     "Check this out",
		URL:       "https://myapp.com/landing",
		Permalink: "/r/programming/comments/abc/check_this_out/",
	}
	if !IsPromotional(post) {
		t.Fatal("expected promotional")
	}
}

func TestIsPromotional_AllowsRedditSelfPost(t *testing.T) {
	post := FetchedPost{
		Title:     "Question about deployment",
		Selftext:  "How do you handle blue-green deploys?",
		URL:       "https://www.reddit.com/r/devops/comments/abc/question",
		Permalink: "/r/devops/comments/abc/question/",
	}
	if IsPromotional(post) {
		t.Fatal("expected NOT promotional")
	}
}

func TestIsPromotional_CaseInsensitive(t *testing.T) {
	if !IsPromotional(FetchedPost{Title: "i built a cli tool"}) {
		t.Fatal("expected promotional (lowercase)")
	}
	if !IsPromotional(FetchedPost{Title: "INTRODUCING our platform"}) {
		t.Fatal("expected promotional (uppercase)")
	}
}

func TestIsPromotional_OnlyFirst300CharsOfBody(t *testing.T) {
	longBody := strings.Repeat("A", 301) + "github.com/myrepo"
	post := FetchedPost{
		Title:    "Normal question",
		Selftext: longBody,
	}
	if IsPromotional(post) {
		t.Fatal("expected NOT promotional (pattern beyond 300 chars)")
	}
}

// --- DeduplicatePosts tests ---

func TestDeduplicatePosts_KeepsHigherScore(t *testing.T) {
	posts := []FetchedPost{
		{Author: "alice", Title: "My Question", Score: 10},
		{Author: "alice", Title: "My Question", Score: 50},
		{Author: "alice", Title: "My Question", Score: 5},
	}
	result := DeduplicatePosts(posts)
	if len(result) != 1 {
		t.Fatalf("expected 1 post, got %d", len(result))
	}
	if result[0].Score != 50 {
		t.Fatalf("expected score 50, got %d", result[0].Score)
	}
}

func TestDeduplicatePosts_KeepsBothDifferentAuthors(t *testing.T) {
	posts := []FetchedPost{
		{Author: "alice", Title: "My Question", Score: 10},
		{Author: "bob", Title: "My Question", Score: 50},
	}
	result := DeduplicatePosts(posts)
	if len(result) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(result))
	}
}

func TestDeduplicatePosts_KeepsBothDifferentTitles(t *testing.T) {
	posts := []FetchedPost{
		{Author: "alice", Title: "First Question", Score: 10},
		{Author: "alice", Title: "Second Question", Score: 20},
	}
	result := DeduplicatePosts(posts)
	if len(result) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(result))
	}
}

func TestDeduplicatePosts_NormalizesTitleCaseAndWhitespace(t *testing.T) {
	posts := []FetchedPost{
		{Author: "alice", Title: "  My Question  ", Score: 10},
		{Author: "alice", Title: "my question", Score: 25},
	}
	result := DeduplicatePosts(posts)
	if len(result) != 1 {
		t.Fatalf("expected 1 post, got %d", len(result))
	}
	if result[0].Score != 25 {
		t.Fatalf("expected score 25, got %d", result[0].Score)
	}
}

func TestDeduplicatePosts_EmptyInput(t *testing.T) {
	result := DeduplicatePosts([]FetchedPost{})
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d", len(result))
	}
}

func TestDeduplicatePosts_NoDuplicates(t *testing.T) {
	posts := []FetchedPost{
		{Author: "alice", Title: "Question A", Score: 10},
		{Author: "bob", Title: "Question B", Score: 20},
		{Author: "carol", Title: "Question C", Score: 30},
	}
	result := DeduplicatePosts(posts)
	if len(result) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(result))
	}
}
