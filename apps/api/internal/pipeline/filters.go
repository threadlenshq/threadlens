package pipeline

import (
	"regexp"
	"strings"
)

// titlePatterns mirrors the TITLE_PATTERNS array from filters.js.
var titlePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)i built`),
	regexp.MustCompile(`(?i)i made`),
	regexp.MustCompile(`(?i)i created`),
	regexp.MustCompile(`(?i)show hn`),
	regexp.MustCompile(`(?i)show r/`),
	regexp.MustCompile(`(?i)introducing`),
	regexp.MustCompile(`(?i)launching`),
	regexp.MustCompile(`(?i)release`),
	regexp.MustCompile(`(?i)\[v`),
	regexp.MustCompile(`(?i)v[2-6]\.`),
	regexp.MustCompile(`(?i)open source`),
}

// bodyPatterns mirrors the BODY_PATTERNS array from filters.js.
var bodyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)github\.com`),
	regexp.MustCompile(`(?i)product hunt`),
	regexp.MustCompile(`(?i)producthunt\.com`),
	regexp.MustCompile(`(?i)my tool`),
	regexp.MustCompile(`(?i)we built`),
	regexp.MustCompile(`(?i)our product`),
	regexp.MustCompile(`(?i)feedback welcome`),
	regexp.MustCompile(`(?i)check it out at`),
}

// FetchedPost is the post shape used by filter, dedup, and platform fetchers.
// Reddit-specific fields: ID (name), Title, Selftext, Author, Permalink, URL,
// Subreddit, Score, NumComments, CreatedUTC.
// Bluesky-specific fields: CID, Text, AuthorHandle, AuthorDisplayName,
// LikeCount, ReplyCount, RepostCount, IndexedAt, PostURL.
type FetchedPost struct {
	// Common / Reddit
	ID        string
	Author    string
	Title     string
	Selftext  string
	URL       string
	Permalink string
	Score     int

	// Reddit extended
	Subreddit   string
	NumComments int
	CreatedUTC  float64

	// Bluesky
	CID               string
	Text              string
	AuthorHandle      string
	AuthorDisplayName string
	LikeCount         int
	ReplyCount        int
	RepostCount       int
	IndexedAt         string
	PostURL           string
}

// IsPromotional returns true if the post matches promotional patterns.
// It mirrors isPromotional() from apps/api/server/pipeline/filters.js.
func IsPromotional(post FetchedPost) bool {
	title := post.Title

	// Check title patterns.
	for _, re := range titlePatterns {
		if re.MatchString(title) {
			return true
		}
	}

	// Only inspect first 300 chars of body.
	body := post.Selftext
	if len(body) > 300 {
		body = body[:300]
	}

	// Check body patterns.
	for _, re := range bodyPatterns {
		if re.MatchString(body) {
			return true
		}
	}

	// Link post check: external URL (not reddit.com).
	url := post.URL
	permalink := post.Permalink
	if url != "" && permalink != "" && !strings.Contains(url, "reddit.com") {
		return true
	}

	return false
}

// DeduplicatePosts deduplicates posts by author+title key, keeping the
// highest-scored post in each group.
// It mirrors deduplicatePosts() from apps/api/server/pipeline/filters.js.
func DeduplicatePosts(posts []FetchedPost) []FetchedPost {
	type entry struct {
		post  FetchedPost
		index int // insertion order for stable output
	}

	groups := make(map[string]entry)
	order := make([]string, 0, len(posts))

	for _, post := range posts {
		key := post.Author + "::" + strings.ToLower(strings.TrimSpace(post.Title))
		existing, found := groups[key]
		if !found {
			groups[key] = entry{post: post, index: len(order)}
			order = append(order, key)
		} else if post.Score > existing.post.Score {
			groups[key] = entry{post: post, index: existing.index}
		}
	}

	result := make([]FetchedPost, len(order))
	for _, key := range order {
		e := groups[key]
		result[e.index] = e.post
	}
	return result
}
