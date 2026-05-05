package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	blueskyBaseURL    = "https://bsky.social"
	blueskyQueryDelay = 2000 * time.Millisecond
	blueskyMaxRetries = 2
	blueskyBaseBackof = 2000 * time.Millisecond
)

var blueskyHTTPClient = &http.Client{Timeout: 15 * time.Second}

// blueskyFetchWithRetry fetches a URL with retry logic for 429/503 responses.
func blueskyFetchWithRetry(ctx context.Context, method, reqURL string, headers map[string]string, bodyBytes []byte) ([]byte, error) {
	for attempt := 0; attempt <= blueskyMaxRetries; attempt++ {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := blueskyHTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch: %w", err)
		}
		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read body: %w", readErr)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		if (resp.StatusCode == 429 || resp.StatusCode == 503) && attempt < blueskyMaxRetries {
			backoff := time.Duration(1<<uint(attempt)) * blueskyBaseBackof
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}

		return nil, fmt.Errorf("Bluesky fetch failed: %d for %s", resp.StatusCode, reqURL)
	}
	return nil, fmt.Errorf("Bluesky fetch failed after retries: %s", reqURL)
}

// blueskySession holds the result of a successful createSession call.
type blueskySession struct {
	AccessJwt string `json:"accessJwt"`
	DID       string `json:"did"`
}

// blueskyAuthenticate authenticates with Bluesky and returns the session.
func blueskyAuthenticate(ctx context.Context, handle, appPassword string) (blueskySession, error) {
	payload, err := json.Marshal(map[string]string{
		"identifier": handle,
		"password":   appPassword,
	})
	if err != nil {
		return blueskySession{}, fmt.Errorf("marshal auth payload: %w", err)
	}

	headers := map[string]string{"Content-Type": "application/json"}
	body, err := blueskyFetchWithRetry(ctx, http.MethodPost,
		blueskyBaseURL+"/xrpc/com.atproto.server.createSession",
		headers, payload)
	if err != nil {
		return blueskySession{}, fmt.Errorf("Bluesky authentication failed: %w", err)
	}

	var sess blueskySession
	if err := json.Unmarshal(body, &sess); err != nil {
		return blueskySession{}, fmt.Errorf("parse auth response: %w", err)
	}
	if sess.AccessJwt == "" {
		return blueskySession{}, fmt.Errorf("Bluesky authentication: empty accessJwt")
	}
	if sess.DID == "" {
		return blueskySession{}, fmt.Errorf("Bluesky authentication: empty did")
	}
	return sess, nil
}

// blueskyDerivePostURL derives a bsky.app post URL from author handle and AT URI.
// AT URI format: at://did:plc:<id>/app.bsky.feed.post/<rkey>
func blueskyDerivePostURL(authorHandle, uri string) string {
	parts := strings.Split(uri, "/")
	rkey := parts[len(parts)-1]
	return "https://bsky.app/profile/" + authorHandle + "/post/" + rkey
}

// BlueskyReplyRef is a reference to the root/parent post for a reply.
type BlueskyReplyRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// PostBlueskyReply logs in with the given credentials and posts a reply to the
// identified post. It returns the raw JSON response from the Bluesky API.
// Mirrors the BskyAgent.post() call in apps/api/server/routes/posts.js.
//
// Note: the Express implementation uses @atproto/api's RichText.detectFacets()
// to extract mention/link facets from the text. Go has no equivalent SDK, so
// facets are omitted; plain-text replies will work correctly and this is
// acceptable parity for the current use case.
func PostBlueskyReply(ctx context.Context, handle, appPassword, text, parentURI, parentCID string) (json.RawMessage, error) {
	sess, err := blueskyAuthenticate(ctx, handle, appPassword)
	if err != nil {
		return nil, err
	}

	ref := BlueskyReplyRef{URI: parentURI, CID: parentCID}
	// com.atproto.repo.createRecord requires repo + collection + record wrapper.
	payload, err := json.Marshal(map[string]any{
		"repo":       sess.DID,
		"collection": "app.bsky.feed.post",
		"record": map[string]any{
			"$type": "app.bsky.feed.post",
			"text":  text,
			"reply": map[string]any{
				"root":   ref,
				"parent": ref,
			},
			"createdAt": time.Now().UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal reply payload: %w", err)
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + sess.AccessJwt,
	}
	body, err := blueskyFetchWithRetry(ctx, http.MethodPost,
		blueskyBaseURL+"/xrpc/com.atproto.repo.createRecord",
		headers, payload)
	if err != nil {
		return nil, fmt.Errorf("post reply: %w", err)
	}
	return json.RawMessage(body), nil
}

// FetchBlueskyPosts fetches posts from an array of search query strings.
// It deduplicates across queries by post URI.
// Mirrors fetchBlueskyPosts() from apps/api/server/pipeline/bluesky.js.
// Uses BLUESKY_HANDLE and BLUESKY_APP_PASSWORD environment variables.
func FetchBlueskyPosts(ctx context.Context, queries []string, onProgress func(done int, total int)) ([]FetchedPost, error) {
	handle := os.Getenv("BLUESKY_HANDLE")
	appPassword := os.Getenv("BLUESKY_APP_PASSWORD")

	if handle == "" {
		return nil, fmt.Errorf("missing required environment variable: BLUESKY_HANDLE")
	}
	if appPassword == "" {
		return nil, fmt.Errorf("missing required environment variable: BLUESKY_APP_PASSWORD")
	}

	accessJwt, err := blueskyAuthenticate(ctx, handle, appPassword)
	if err != nil {
		return nil, err
	}
	authHeaders := map[string]string{"Authorization": "Bearer " + accessJwt.AccessJwt}

	seen := make(map[string]FetchedPost)
	order := make([]string, 0)

	for i, query := range queries {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(blueskyQueryDelay):
			}
		}

		searchURL := blueskyBaseURL + "/xrpc/app.bsky.feed.searchPosts?q=" +
			url.QueryEscape(query) + "&limit=25&sort=latest"

		body, fetchErr := blueskyFetchWithRetry(ctx, http.MethodGet, searchURL, authHeaders, nil)
		if fetchErr != nil {
			log.Printf("[bluesky] Failed to fetch query %q: %v", query, fetchErr)
		} else {
			var result struct {
				Posts []struct {
					URI    string `json:"uri"`
					CID    string `json:"cid"`
					Author struct {
						Handle      string `json:"handle"`
						DisplayName string `json:"displayName"`
					} `json:"author"`
					Record struct {
						Text string `json:"text"`
					} `json:"record"`
					LikeCount   int    `json:"likeCount"`
					ReplyCount  int    `json:"replyCount"`
					RepostCount int    `json:"repostCount"`
					IndexedAt   string `json:"indexedAt"`
				} `json:"posts"`
			}
			if jsonErr := json.Unmarshal(body, &result); jsonErr == nil {
				for _, p := range result.Posts {
					if p.URI == "" {
						continue
					}
					if _, exists := seen[p.URI]; !exists {
						authorHandle := p.Author.Handle
						post := FetchedPost{
							ID:                p.URI,
							CID:               p.CID,
							Text:              p.Record.Text,
							AuthorHandle:      authorHandle,
							AuthorDisplayName: p.Author.DisplayName,
							LikeCount:         p.LikeCount,
							ReplyCount:        p.ReplyCount,
							RepostCount:       p.RepostCount,
							IndexedAt:         p.IndexedAt,
							PostURL:           blueskyDerivePostURL(authorHandle, p.URI),
							// Map common fields too
							Author: authorHandle,
						}
						seen[p.URI] = post
						order = append(order, p.URI)
					}
				}
			}
		}

		if onProgress != nil {
			onProgress(i+1, len(queries))
		}
	}

	posts := make([]FetchedPost, 0, len(order))
	for _, uri := range order {
		posts = append(posts, seen[uri])
	}
	return posts, nil
}
