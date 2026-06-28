package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	hnSearchURL   = "https://hn.algolia.com/api/v1/search_by_date"
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
// (date-sorted, stories + comments) and returns deduped FetchedPost results.
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

// hnSearch performs one Algolia search_by_date request with retry on 429/503.
func hnSearch(ctx context.Context, query string) ([]hnHit, error) {
	q := url.Values{}
	q.Set("query", query)
	q.Set("tags", "(story,comment)")
	q.Set("hitsPerPage", strconv.Itoa(hnHitsPerPage))
	reqURL := hnSearchURL + "?" + q.Encode()

	for attempt := 0; attempt <= hnMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("hn build request: %w", err)
		}
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
