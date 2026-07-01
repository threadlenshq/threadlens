package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// hnSearchURL is the relevance + popularity ranked Algolia endpoint. We
	// deliberately avoid search_by_date: date sorting surfaces brand-new,
	// zero-engagement submissions, while relevance ranking returns substantive,
	// well-discussed posts that carry real pain signal.
	hnSearchURL   = "https://hn.algolia.com/api/v1/search"
	hnHitsPerPage = 50
	hnMaxRetries  = 2
	hnBaseBackoff = 1000 * time.Millisecond
)

var hnHTTPClient = &http.Client{Timeout: 15 * time.Second}

// hnHit is a single Algolia HN search hit. Nullable numeric fields (points,
// num_comments) are pointers because comments omit them.
type hnHit struct {
	ObjectID    string   `json:"objectID"`
	Title       string   `json:"title"`
	StoryTitle  string   `json:"story_title"`
	URL         string   `json:"url"`
	StoryText   string   `json:"story_text"`
	CommentText string   `json:"comment_text"`
	Author      string   `json:"author"`
	Points      *int     `json:"points"`
	NumComments *int     `json:"num_comments"`
	CreatedAtI  int64    `json:"created_at_i"`
	Tags        []string `json:"_tags"`
}

type hnSearchResponse struct {
	Hits []hnHit `json:"hits"`
}

// isComment reports whether the hit is a comment (vs a story) based on _tags.
func (h hnHit) isComment() bool {
	for _, t := range h.Tags {
		if t == "comment" {
			return true
		}
	}
	return false
}

// isDead reports whether the hit is a dead, deleted, or flagged item. HN renders
// these with a placeholder such as "[dead]" in place of the real content, so
// they carry no usable signal and should be dropped before scoring.
func (h hnHit) isDead() bool {
	dead := func(s string) bool {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "[dead]", "[deleted]", "[flagged]":
			return true
		}
		return false
	}
	if h.isComment() {
		return dead(h.CommentText)
	}
	return dead(h.Title)
}

// mapHNHit converts one Algolia hit into the common FetchedPost shape.
func mapHNHit(h hnHit) FetchedPost {
	fp := FetchedPost{
		ID:        "hn_" + h.ObjectID,
		Author:    h.Author,
		Permalink: "https://news.ycombinator.com/item?id=" + h.ObjectID,
	}
	if h.CreatedAtI != 0 {
		fp.CreatedUTC = float64(h.CreatedAtI)
	}
	if h.isComment() {
		// Comments carry the parent story title for cluster/context readability.
		fp.Title = h.StoryTitle
		fp.Selftext = h.CommentText
		fp.Score = 0 // HN comments have no points
	} else {
		fp.Title = h.Title
		fp.Selftext = h.StoryText
		fp.URL = h.URL
		if h.Points != nil {
			fp.Score = *h.Points
		}
		if h.NumComments != nil {
			fp.NumComments = *h.NumComments
		}
	}
	return fp
}

func mapHNHits(hits []hnHit) []FetchedPost {
	out := make([]FetchedPost, 0, len(hits))
	for _, h := range hits {
		if h.ObjectID == "" {
			continue // skip malformed hit
		}
		if h.isDead() {
			continue // skip dead/deleted/flagged items
		}
		out = append(out, mapHNHit(h))
	}
	return out
}

// dedupHNPosts removes duplicate posts by ID, preserving first-seen order.
func dedupHNPosts(posts []FetchedPost) []FetchedPost {
	seen := make(map[string]bool, len(posts))
	out := make([]FetchedPost, 0, len(posts))
	for _, p := range posts {
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		out = append(out, p)
	}
	return out
}

// FetchHackerNewsPosts runs each query string against the Algolia HN Search API
// (relevance-ranked, stories + comments) and returns deduped FetchedPost results.
func FetchHackerNewsPosts(ctx context.Context, queries []string, onProgress func(done, total int)) ([]FetchedPost, error) {
	all := make([]FetchedPost, 0)
	total := len(queries)
	for i, q := range queries {
		hits, err := hnSearch(ctx, q)
		if err != nil {
			return nil, err
		}
		all = append(all, mapHNHits(hits)...)
		if onProgress != nil {
			onProgress(i+1, total)
		}
	}
	return dedupHNPosts(all), nil
}

// normalizeHNQuery extracts a bare search term from a stored query value. The
// Algolia search endpoint expects a plain term, but some HN queries were stored
// as full Algolia URLs (e.g. "https://hn.algolia.com/api/v1/search?query=customer+discovery").
// Passing such a URL verbatim as the query param matches nothing, so when the
// value looks like a URL carrying a "query" parameter, return that parameter;
// otherwise return the value unchanged.
func normalizeHNQuery(query string) string {
	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "http://") && !strings.HasPrefix(query, "https://") {
		return query
	}
	u, err := url.Parse(query)
	if err != nil {
		return query
	}
	if term := strings.TrimSpace(u.Query().Get("query")); term != "" {
		return term
	}
	return query
}

// hnSearch performs one Algolia search request with retry on 429/503.
func hnSearch(ctx context.Context, query string) ([]hnHit, error) {
	q := url.Values{}
	q.Set("query", normalizeHNQuery(query))
	q.Set("tags", "(story,comment)")
	q.Set("hitsPerPage", strconv.Itoa(hnHitsPerPage))
	reqURL := hnSearchURL + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("hn build request: %w", err)
	}

	for attempt := 0; attempt <= hnMaxRetries; attempt++ {
		resp, err := hnHTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("hn fetch: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("hn read body: %w", readErr)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var parsed hnSearchResponse
			if err := json.Unmarshal(body, &parsed); err != nil {
				return nil, fmt.Errorf("hn parse: %w", err)
			}
			return parsed.Hits, nil
		}
		if (resp.StatusCode == 429 || resp.StatusCode == 503) && attempt < hnMaxRetries {
			if err := retryBackoff(ctx, attempt, hnBaseBackoff); err != nil {
				return nil, err
			}
			continue
		}
		return nil, fmt.Errorf("hn search failed: status %d", resp.StatusCode)
	}
	return nil, fmt.Errorf("hn search: exhausted retries")
}
