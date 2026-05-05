package google

import (
	"testing"
)

func floatPtr(f float64) *float64 { return &f }

func TestDedupeClustersByURL(t *testing.T) {
	input := []AnalyzedResult{
		{Title: "How to Resume Coding Projects", URL: "https://example.com/posts/resume-coding?utm_source=google", DisplayURL: "example.com/posts/resume-coding", Snippet: "First version", Rank: floatPtr(1)},
		{Title: "How to Resume Coding Projects", URL: "https://example.com/posts/resume-coding/", DisplayURL: "example.com/posts/resume-coding/", Snippet: "Duplicate from another query", Rank: floatPtr(4)},
		{Title: "How to Resume Coding Projects", URL: "https://www.example.com/posts/resume-coding#tips", DisplayURL: "www.example.com/posts/resume-coding", Snippet: "Duplicate with fragment", Rank: floatPtr(2)},
		{Title: "Different page", URL: "https://example.com/another-page", DisplayURL: "example.com/another-page", Snippet: "Unique", Rank: floatPtr(3)},
	}

	deduped := DedupeSearchResults(input)

	if len(deduped) != 2 {
		t.Fatalf("expected 2, got %d: %+v", len(deduped), deduped)
	}

	var canonical *AnalyzedResult
	for i := range deduped {
		if containsStr(deduped[i].URL, "/posts/resume-coding") {
			canonical = &deduped[i]
		}
	}
	if canonical == nil {
		t.Fatal("expected to find canonical resume-coding entry")
	}
	if canonical.Appearances != 3 {
		t.Errorf("expected appearances=3, got %d", canonical.Appearances)
	}
	if len(canonical.DuplicateURLs) != 3 {
		t.Errorf("expected 3 duplicate_urls, got %d: %v", len(canonical.DuplicateURLs), canonical.DuplicateURLs)
	}
	if canonical.Rank == nil || *canonical.Rank != 1 {
		t.Errorf("expected rank=1")
	}

	var unique *AnalyzedResult
	for i := range deduped {
		if containsStr(deduped[i].URL, "/another-page") {
			unique = &deduped[i]
		}
	}
	if unique == nil {
		t.Fatal("expected unique another-page entry")
	}
	if unique.Appearances != 1 {
		t.Errorf("expected appearances=1, got %d", unique.Appearances)
	}
	if len(unique.DuplicateURLs) != 1 || unique.DuplicateURLs[0] != "https://example.com/another-page" {
		t.Errorf("unexpected duplicate_urls: %v", unique.DuplicateURLs)
	}
}

func TestDedupeByContentHash(t *testing.T) {
	input := []AnalyzedResult{
		{Title: "Guide: Build a Dashboard", URL: "https://news.example.com/posts/dashboard-guide", DisplayURL: "news.example.com/posts/dashboard-guide", Snippet: "Original source copy", Rank: floatPtr(2), ContentHash: "sha1:abc123samecontent"},
		{Title: "Build Dashboard in 10 Steps", URL: "https://blog.example.org/engineering/dashboard-steps", DisplayURL: "blog.example.org/engineering/dashboard-steps", Snippet: "Syndicated copy on another host", Rank: floatPtr(1), ContentHash: "sha1:abc123samecontent"},
	}

	deduped := DedupeSearchResults(input)

	if len(deduped) != 1 {
		t.Fatalf("expected 1, got %d", len(deduped))
	}
	if deduped[0].Appearances != 2 {
		t.Errorf("expected appearances=2, got %d", deduped[0].Appearances)
	}
	if len(deduped[0].DuplicateURLs) != 2 {
		t.Errorf("expected 2 duplicate_urls, got %v", deduped[0].DuplicateURLs)
	}
	if deduped[0].DuplicateURLs[0] != "https://news.example.com/posts/dashboard-guide" {
		t.Errorf("unexpected first duplicate_url: %q", deduped[0].DuplicateURLs[0])
	}
	if deduped[0].DuplicateURLs[1] != "https://blog.example.org/engineering/dashboard-steps" {
		t.Errorf("unexpected second duplicate_url: %q", deduped[0].DuplicateURLs[1])
	}
	if deduped[0].Rank == nil || *deduped[0].Rank != 1 {
		t.Errorf("expected rank=1")
	}
	if deduped[0].Title != "Build Dashboard in 10 Steps" {
		t.Errorf("expected preferred title, got %q", deduped[0].Title)
	}
}

func TestDedupeMergesSources(t *testing.T) {
	input := []AnalyzedResult{
		{Title: "Current representative", URL: "https://example.com/docs/start", Rank: floatPtr(1), Sources: []string{"google:web"}},
		{Title: "Duplicate from another source", URL: "https://www.example.com/docs/start?utm_source=newsletter", Rank: floatPtr(3), Sources: []string{"google:news", "google:web"}},
	}

	deduped := DedupeSearchResults(input)
	if len(deduped) != 1 {
		t.Fatalf("expected 1, got %d", len(deduped))
	}
	rep := deduped[0]
	if rep.Rank == nil || *rep.Rank != 1 {
		t.Errorf("expected rank=1")
	}
	if len(rep.Sources) != 2 {
		t.Errorf("expected 2 sources, got %v", rep.Sources)
	}
	// Should have google:web and google:news
	hasFn := func(s string) bool {
		for _, src := range rep.Sources {
			if src == s {
				return true
			}
		}
		return false
	}
	if !hasFn("google:web") || !hasFn("google:news") {
		t.Errorf("expected both sources, got %v", rep.Sources)
	}
}

func TestDedupeReplacesRepresentativeWithBetterRank(t *testing.T) {
	input := []AnalyzedResult{
		{
			Title: "Legacy title", URL: "https://example.com/guides/startup?utm_source=newsletter",
			DisplayURL: "example.com/guides/startup", Snippet: "Legacy snippet", Rank: floatPtr(7),
			RelevanceFit: "adjacent", RelevanceScore: 3.2, ConfidenceScore: 0.42,
			OpportunityTypes: []string{"other"}, KeegoingFitReasons: []string{"old reason"},
			Disqualifiers: []string{"old disqualifier"}, Summary: "Old summary",
			ActionRecommendation: "Old recommendation", OutreachCandidate: 0,
			Sources: []string{"google:web"},
		},
		{
			Title: "Preferred title", URL: "https://www.example.com/guides/startup",
			DisplayURL: "www.example.com/guides/startup", Snippet: "Preferred snippet", Rank: floatPtr(1),
			RelevanceFit: "direct_fit", RelevanceScore: 9.3, ConfidenceScore: 0.91,
			OpportunityTypes: []string{"content_gap", "competitor_weakness"}, KeegoingFitReasons: []string{"high purchase intent"},
			Disqualifiers: []string{}, Summary: "Preferred summary",
			ActionRecommendation: "Prioritize this page", OutreachCandidate: 1,
			Sources: []string{"google:news"},
		},
	}

	deduped := DedupeSearchResults(input)
	if len(deduped) != 1 {
		t.Fatalf("expected 1, got %d", len(deduped))
	}
	rep := deduped[0]
	if rep.Rank == nil || *rep.Rank != 1 {
		t.Errorf("expected rank=1")
	}
	if rep.Title != "Preferred title" {
		t.Errorf("expected Preferred title, got %q", rep.Title)
	}
	if rep.Summary != "Preferred summary" {
		t.Errorf("expected Preferred summary, got %q", rep.Summary)
	}
	if rep.RelevanceFit != "direct_fit" {
		t.Errorf("expected direct_fit, got %q", rep.RelevanceFit)
	}
	if rep.RelevanceScore != 9.3 {
		t.Errorf("expected 9.3, got %v", rep.RelevanceScore)
	}
	if rep.OutreachCandidate != 1 {
		t.Errorf("expected outreach_candidate=1")
	}
	if rep.Appearances != 2 {
		t.Errorf("expected appearances=2, got %d", rep.Appearances)
	}
	if len(rep.DuplicateURLs) != 2 {
		t.Errorf("expected 2 duplicate_urls, got %v", rep.DuplicateURLs)
	}
}
