package pipeline

import "testing"

func TestMapHNHitStory(t *testing.T) {
	points := 142
	numComments := 37
	h := hnHit{
		ObjectID:    "38493012",
		Title:       "Show HN: I built a thing",
		URL:         "https://example.com/thing",
		StoryText:   "Here is the body",
		Author:      "alice",
		Points:      &points,
		NumComments: &numComments,
		CreatedAtI:  1700000000,
		Tags:        []string{"story", "author_alice", "story_38493012"},
	}
	p := mapHNHit(h)
	if p.ID != "hn_38493012" {
		t.Errorf("ID = %q, want hn_38493012", p.ID)
	}
	if p.Title != "Show HN: I built a thing" {
		t.Errorf("Title = %q", p.Title)
	}
	if p.Selftext != "Here is the body" {
		t.Errorf("Selftext = %q", p.Selftext)
	}
	if p.URL != "https://example.com/thing" {
		t.Errorf("URL = %q", p.URL)
	}
	if p.Score != 142 {
		t.Errorf("Score = %d, want 142", p.Score)
	}
	if p.NumComments != 37 {
		t.Errorf("NumComments = %d, want 37", p.NumComments)
	}
	if p.Author != "alice" {
		t.Errorf("Author = %q", p.Author)
	}
	if p.Permalink != "https://news.ycombinator.com/item?id=38493012" {
		t.Errorf("Permalink = %q", p.Permalink)
	}
	if p.CreatedUTC != 1700000000 {
		t.Errorf("CreatedUTC = %v", p.CreatedUTC)
	}
}

func TestMapHNHitComment(t *testing.T) {
	h := hnHit{
		ObjectID:    "38493999",
		StoryTitle:  "Parent story title",
		CommentText: "I really hate doing invoicing by hand",
		Author:      "bob",
		CreatedAtI:  1700000500,
		Tags:        []string{"comment", "author_bob", "story_38493012"},
	}
	p := mapHNHit(h)
	if p.ID != "hn_38493999" {
		t.Errorf("ID = %q", p.ID)
	}
	if p.Title != "Parent story title" {
		t.Errorf("comment Title (parent) = %q", p.Title)
	}
	if p.Selftext != "I really hate doing invoicing by hand" {
		t.Errorf("Selftext = %q", p.Selftext)
	}
	if p.Score != 0 {
		t.Errorf("comment Score = %d, want 0", p.Score)
	}
	if p.URL != "" {
		t.Errorf("comment URL = %q, want empty", p.URL)
	}
	if p.Permalink != "https://news.ycombinator.com/item?id=38493999" {
		t.Errorf("Permalink = %q", p.Permalink)
	}
}

func TestNormalizeHNQuery(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bare term", "customer discovery", "customer discovery"},
		{"bare term trimmed", "  self hosted analytics  ", "self hosted analytics"},
		{"full algolia api url", "https://hn.algolia.com/api/v1/search?query=customer+discovery", "customer discovery"},
		{"full algolia ui url", "https://hn.algolia.com/?dateRange=all&type=all&query=open+core+business", "open core business"},
		{"url without query param falls back to raw", "https://hn.algolia.com/api/v1/search?tags=story", "https://hn.algolia.com/api/v1/search?tags=story"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeHNQuery(tc.in); got != tc.want {
				t.Errorf("normalizeHNQuery(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMapHNHitsDropsDead(t *testing.T) {
	points := 5
	hits := []hnHit{
		{ObjectID: "1", Title: "Show HN: a real launch", Points: &points, Tags: []string{"story"}},
		{ObjectID: "2", Title: "[dead]", Tags: []string{"story"}},
		{ObjectID: "3", Title: "[flagged]", Tags: []string{"story"}},
		{ObjectID: "4", Title: "[deleted]", Tags: []string{"story"}},
		{ObjectID: "5", StoryTitle: "Parent", CommentText: "[dead]", Tags: []string{"comment"}},
		{ObjectID: "6", StoryTitle: "Parent", CommentText: "I hate doing X by hand", Tags: []string{"comment"}},
	}
	out := mapHNHits(hits)
	if len(out) != 2 {
		t.Fatalf("mapHNHits dropped dead len = %d, want 2", len(out))
	}
	got := map[string]bool{}
	for _, p := range out {
		got[p.ID] = true
	}
	if !got["hn_1"] || !got["hn_6"] {
		t.Errorf("expected hn_1 and hn_6 to survive, got %v", got)
	}
}

func TestDedupHNHitsByID(t *testing.T) {
	points := 1
	hits := []hnHit{
		{ObjectID: "1", Title: "a", Points: &points, Tags: []string{"story"}},
		{ObjectID: "1", Title: "a", Points: &points, Tags: []string{"story"}},
		{ObjectID: "2", Title: "b", Points: &points, Tags: []string{"story"}},
	}
	out := dedupHNPosts(mapHNHits(hits))
	if len(out) != 2 {
		t.Fatalf("dedup len = %d, want 2", len(out))
	}
}
