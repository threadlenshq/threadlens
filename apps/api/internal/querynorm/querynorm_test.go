package querynorm

import "testing"

func TestKey_RedditVariantsCollapse(t *testing.T) {
	// Canonical Phase A form, browser %20 form, and reordered params must all match.
	a := Key("reddit", "https://www.reddit.com/search.json?q=manual+invoicing&sort=new&t=month&limit=100")
	b := Key("reddit", "https://www.reddit.com/search?q=manual%20invoicing")
	c := Key("reddit", "https://www.reddit.com/search?sort=top&q=Manual+Invoicing&t=week")
	if a != b || a != c {
		t.Fatalf("reddit keys should match:\n a=%q\n b=%q\n c=%q", a, b, c)
	}
}

func TestKey_RedditSubredditIsDistinct(t *testing.T) {
	site := Key("reddit", "https://www.reddit.com/search?q=invoicing")
	sub := Key("reddit", "https://www.reddit.com/r/smallbusiness/search?q=invoicing&restrict_sr=1")
	if site == sub {
		t.Fatal("subreddit-restricted search must be a distinct target from site-wide")
	}
}

func TestKey_GenericTrimAndLower(t *testing.T) {
	a := Key("google", "  Manual   Invoicing ")
	b := Key("google", "manual invoicing")
	if a != b {
		t.Fatalf("generic keys should match: %q vs %q", a, b)
	}
	if Key("google", "x") == Key("bluesky", "x") {
		t.Fatal("keys should be platform-scoped")
	}
}
