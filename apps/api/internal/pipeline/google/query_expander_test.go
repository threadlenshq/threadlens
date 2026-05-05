package google

import (
	"testing"
)

func TestExpandQueriesExpands(t *testing.T) {
	queries := ExpandQueries("resume coding project")
	expected := []string{
		"resume coding project",
		"how to resume coding project",
		"why is it hard to resume coding project",
		"best tool resume coding project",
		"site:reddit.com resume coding project",
		"site:reddit.com resume coding project problem",
		"site:reddit.com resume coding project frustrated",
		"site:news.ycombinator.com resume coding project",
		"site:news.ycombinator.com resume coding project problem",
		"site:stackexchange.com resume coding project",
		"site:github.com/discussions resume coding project",
	}
	if len(queries) != len(expected) {
		t.Fatalf("expected %d queries, got %d: %v", len(expected), len(queries), queries)
	}
	for i, q := range queries {
		if q != expected[i] {
			t.Errorf("query[%d]: expected %q, got %q", i, expected[i], q)
		}
	}
}

func TestExpandQueriesDeduplicatesAndNormalizes(t *testing.T) {
	first := ExpandQueries("  resume coding project  ")
	second := ExpandQueries("resume coding project")

	// No duplicates
	seen := map[string]bool{}
	for _, q := range first {
		if seen[q] {
			t.Errorf("duplicate query: %q", q)
		}
		seen[q] = true
	}

	// Same output
	if len(first) != len(second) {
		t.Fatalf("lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("query[%d] mismatch: %q vs %q", i, first[i], second[i])
		}
	}
}

func TestExpandQueriesForumBias(t *testing.T) {
	queries := ExpandQueries("resume coding project")

	mustContain := []string{
		"site:news.ycombinator.com resume coding project",
		"site:stackexchange.com resume coding project",
		"site:github.com/discussions resume coding project",
		"site:reddit.com resume coding project problem",
		"site:reddit.com resume coding project frustrated",
		"site:news.ycombinator.com resume coding project problem",
	}
	for _, q := range mustContain {
		found := false
		for _, got := range queries {
			if got == q {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find %q in queries", q)
		}
	}

	forumCount := 0
	articleCount := 0
	for _, q := range queries {
		if len(q) >= 5 && q[:5] == "site:" {
			forumCount++
		} else {
			articleCount++
		}
	}
	if forumCount <= articleCount {
		t.Errorf("expected more forum queries than article queries, got %d forum vs %d article", forumCount, articleCount)
	}

	for _, q := range queries {
		if containsStr(q, "stackoverflow.com") {
			t.Errorf("should not contain stackoverflow.com in queries")
		}
	}
}

func TestExpandQueriesEmpty(t *testing.T) {
	queries := ExpandQueries("")
	if len(queries) != 0 {
		t.Errorf("expected empty, got %v", queries)
	}
	queries2 := ExpandQueries("   ")
	if len(queries2) != 0 {
		t.Errorf("expected empty for whitespace, got %v", queries2)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()))
}
