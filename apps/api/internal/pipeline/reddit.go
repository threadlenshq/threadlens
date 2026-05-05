// Package pipeline contains Scout data-fetching helpers that mirror the Express pipeline.
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	redditUserAgent  = "Scout/1.0"
	redditQueryDelay = 2500 * time.Millisecond
	redditMaxRetries = 2
)

// RedditComment is a single top-level comment returned by FetchRedditContext.
type RedditComment struct {
	Author string `json:"author"`
	Body   string `json:"body"`
	Score  int    `json:"score"`
}

// RedditContext holds the full post body and top comments retrieved from Reddit.
type RedditContext struct {
	FullBody    string
	TopComments []RedditComment
}

var redditHTTPClient = &http.Client{Timeout: 15 * time.Second}

// redditFetchWithRetry fetches a URL with retry logic for 429/503 responses.
func redditFetchWithRetry(ctx context.Context, url string) ([]byte, error) {
	for attempt := 0; attempt <= redditMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("User-Agent", redditUserAgent)

		resp, err := redditHTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read body: %w", readErr)
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		if (resp.StatusCode == 429 || resp.StatusCode == 503) && attempt < redditMaxRetries {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}

		return nil, fmt.Errorf("Reddit fetch failed: %d for %s", resp.StatusCode, url)
	}
	return nil, fmt.Errorf("Reddit fetch failed after retries: %s", url)
}

// FetchRedditPosts fetches posts from an array of search query URLs.
// It deduplicates across queries by post name.
// Mirrors fetchRedditPosts() from apps/api/server/pipeline/reddit.js.
func FetchRedditPosts(ctx context.Context, queryURLs []string, onProgress func(done int, total int)) ([]FetchedPost, error) {
	seen := make(map[string]FetchedPost)
	order := make([]string, 0)

	for i, url := range queryURLs {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(redditQueryDelay):
			}
		}

		body, err := redditFetchWithRetry(ctx, url)
		if err != nil {
			log.Printf("[reddit] Failed to fetch %s: %v", url, err)
		} else {
			var listing struct {
				Data struct {
					Children []struct {
						Data struct {
							Name        string  `json:"name"`
							Title       string  `json:"title"`
							Selftext    string  `json:"selftext"`
							Author      string  `json:"author"`
							Permalink   string  `json:"permalink"`
							Subreddit   string  `json:"subreddit"`
							Score       int     `json:"score"`
							NumComments int     `json:"num_comments"`
							CreatedUTC  float64 `json:"created_utc"`
							URL         string  `json:"url"`
						} `json:"data"`
					} `json:"children"`
				} `json:"data"`
			}
			if jsonErr := json.Unmarshal(body, &listing); jsonErr == nil {
				for _, child := range listing.Data.Children {
					d := child.Data
					if d.Name == "" {
						continue
					}
					if _, exists := seen[d.Name]; !exists {
						seen[d.Name] = FetchedPost{
							ID:          d.Name,
							Title:       d.Title,
							Selftext:    d.Selftext,
							Author:      d.Author,
							Permalink:   d.Permalink,
							Subreddit:   d.Subreddit,
							Score:       d.Score,
							NumComments: d.NumComments,
							CreatedUTC:  d.CreatedUTC,
							URL:         d.URL,
						}
						order = append(order, d.Name)
					}
				}
			}
		}

		if onProgress != nil {
			onProgress(i+1, len(queryURLs))
		}
	}

	posts := make([]FetchedPost, 0, len(order))
	for _, name := range order {
		posts = append(posts, seen[name])
	}
	return posts, nil
}

// FetchRedditContext fetches the full post body and top 5 comments for a Reddit
// post URL (permalink path, e.g. "/r/foo/comments/…").  It mirrors the Express
// fetchRedditContext() from apps/api/server/pipeline/reddit.js.
//
// On network failure the function returns an empty RedditContext and the error
// so that callers can decide whether to abort (500) or continue with partial data.
func FetchRedditContext(ctx context.Context, postURL string) (RedditContext, error) {
	// Build the JSON endpoint URL.
	// postURL may be a full https URL or a /r/… permalink path.
	var jsonURL string
	if strings.HasPrefix(postURL, "http") {
		// Strip trailing slash, append .json
		trimmed := strings.TrimRight(postURL, "/")
		jsonURL = trimmed + ".json?limit=5&depth=1&sort=top"
	} else {
		// Treat as a Reddit permalink path.
		trimmed := strings.TrimRight(postURL, "/")
		jsonURL = "https://www.reddit.com" + trimmed + ".json?limit=5&depth=1&sort=top"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jsonURL, nil)
	if err != nil {
		return RedditContext{}, fmt.Errorf("reddit context: build request: %w", err)
	}
	req.Header.Set("User-Agent", "scout-api-go/1.0")

	resp, err := redditHTTPClient.Do(req)
	if err != nil {
		return RedditContext{}, fmt.Errorf("reddit context: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RedditContext{}, fmt.Errorf("reddit context: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return RedditContext{}, fmt.Errorf("reddit context: read body: %w", err)
	}

	// Reddit returns a 2-element JSON array: [postListing, commentListing].
	var listings []json.RawMessage
	if err := json.Unmarshal(body, &listings); err != nil || len(listings) < 2 {
		return RedditContext{}, fmt.Errorf("reddit context: unexpected response format")
	}

	// Extract full body from post listing.
	var postListing struct {
		Data struct {
			Children []struct {
				Data struct {
					Selftext string `json:"selftext"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	_ = json.Unmarshal(listings[0], &postListing)
	fullBody := ""
	if len(postListing.Data.Children) > 0 {
		fullBody = postListing.Data.Children[0].Data.Selftext
	}

	// Extract top comments from comment listing.
	var commentListing struct {
		Data struct {
			Children []struct {
				Kind string `json:"kind"`
				Data struct {
					Author string `json:"author"`
					Body   string `json:"body"`
					Score  int    `json:"score"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	_ = json.Unmarshal(listings[1], &commentListing)

	var topComments []RedditComment
	for _, child := range commentListing.Data.Children {
		if child.Kind != "t1" {
			continue
		}
		body := child.Data.Body
		if len(body) > 500 {
			body = body[:500]
		}
		topComments = append(topComments, RedditComment{
			Author: child.Data.Author,
			Body:   body,
			Score:  child.Data.Score,
		})
		if len(topComments) >= 5 {
			break
		}
	}
	if topComments == nil {
		topComments = []RedditComment{}
	}

	return RedditContext{FullBody: fullBody, TopComments: topComments}, nil
}
