package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// buildRedditJSONURL constructs the .json endpoint URL from a post URL or
// permalink path. Accepts both https://www.reddit.com/... and /r/... formats.
func buildRedditJSONURL(postURL string) string {
	if strings.HasPrefix(postURL, "http") {
		trimmed := strings.TrimRight(postURL, "/")
		return trimmed + ".json?limit=5&depth=1&sort=top"
	}
	trimmed := strings.TrimRight(postURL, "/")
	return "https://www.reddit.com" + trimmed + ".json?limit=5&depth=1&sort=top"
}

// parseRedditPostResponse parses the 2-element Reddit .json array into a
// FetchedPost. The body is expected to be [postListing, commentListing].
func parseRedditPostResponse(body []byte) (*FetchedPost, error) {
	var listings []json.RawMessage
	if err := json.Unmarshal(body, &listings); err != nil || len(listings) < 2 {
		return nil, fmt.Errorf("reddit fetch: unexpected response format")
	}

	var postListing struct {
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
	if err := json.Unmarshal(listings[0], &postListing); err != nil {
		return nil, fmt.Errorf("reddit fetch: unexpected response format")
	}
	if len(postListing.Data.Children) == 0 {
		return nil, fmt.Errorf("reddit fetch: unexpected response format")
	}

	d := postListing.Data.Children[0].Data
	if d.Name == "" {
		return nil, fmt.Errorf("reddit fetch: unexpected response format")
	}

	post := &FetchedPost{
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

	// Parse the comment listing.
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
		commentBody := child.Data.Body
		if len(commentBody) > 500 {
			commentBody = commentBody[:500]
		}
		topComments = append(topComments, RedditComment{
			Author: child.Data.Author,
			Body:   commentBody,
			Score:  child.Data.Score,
		})
		if len(topComments) >= 5 {
			break
		}
	}
	if topComments == nil {
		topComments = []RedditComment{}
	}
	post.TopComments = topComments

	return post, nil
}

// parseBlueskyThreadResponse parses a getPostThread response into a FetchedPost.
// fallbackHandle is used when the response doesn't include an author handle.
func parseBlueskyThreadResponse(body []byte, fallbackHandle string) (*FetchedPost, error) {
	var result struct {
		Thread struct {
			Post struct {
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
			} `json:"post"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("bluesky fetch: unexpected response format")
	}

	p := result.Thread.Post
	if p.URI == "" {
		return nil, fmt.Errorf("bluesky fetch: not found")
	}

	authorHandle := p.Author.Handle
	if authorHandle == "" {
		authorHandle = fallbackHandle
	}

	post := &FetchedPost{
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
		Author:            authorHandle,
	}

	return post, nil
}

// FetchSingleRedditPost fetches a single Reddit post by URL and returns a
// fully-populated FetchedPost including the top 5 comments. It reuses
// redditFetchWithRetry for cookie-harvest / 403-retry / 429-cooldown logic
// but does not call redditThrottle (the batch per-query delay).
//
// Accepts either a full https://www.reddit.com/r/<sub>/comments/<id>/... URL
// or a /r/<sub>/comments/<id>/... permalink path.
func FetchSingleRedditPost(ctx context.Context, postURL string) (*FetchedPost, error) {
	jsonURL := buildRedditJSONURL(postURL)

	body, err := redditFetchWithRetry(ctx, jsonURL)
	if err != nil {
		return nil, fmt.Errorf("reddit fetch: %w", err)
	}

	return parseRedditPostResponse(body)
}

// FetchSingleBlueskyPost fetches a single Bluesky post by URL or AT URI and
// returns a fully-populated FetchedPost. It authenticates with BLUESKY_HANDLE
// and BLUESKY_APP_PASSWORD (same as FetchBlueskyReplies).
//
// Accepts either a https://bsky.app/profile/<handle>/post/<rkey> URL or an
// at://<did>/app.bsky.feed.post/<rkey> AT URI.
func FetchSingleBlueskyPost(ctx context.Context, postURL string) (*FetchedPost, error) {
	handle := os.Getenv("BLUESKY_HANDLE")
	appPassword := os.Getenv("BLUESKY_APP_PASSWORD")
	if handle == "" || appPassword == "" {
		return nil, fmt.Errorf("bluesky fetch: missing BLUESKY_HANDLE or BLUESKY_APP_PASSWORD")
	}

	sess, err := blueskyAuthenticate(ctx, handle, appPassword)
	if err != nil {
		return nil, fmt.Errorf("bluesky fetch: %w", err)
	}
	authHeaders := map[string]string{"Authorization": "Bearer " + sess.AccessJwt}

	// Determine the AT URI to query.
	var atURI string
	var authorHandle string
	if strings.HasPrefix(postURL, "at://") {
		atURI = postURL
	} else {
		// Parse bsky.app URL: https://bsky.app/profile/<handle>/post/<rkey>
		parsed, parseErr := url.Parse(postURL)
		if parseErr != nil {
			return nil, fmt.Errorf("bluesky fetch: invalid URL")
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) < 4 || parts[0] != "profile" || parts[2] != "post" {
			return nil, fmt.Errorf("bluesky fetch: invalid URL format")
		}
		authorHandle = parts[1]
		rkey := parts[3]

		// Resolve handle to DID via getProfile.
		profileURL := blueskyBaseURL + "/xrpc/app.bsky.actor.getProfile?actor=" + url.QueryEscape(authorHandle)
		profileBody, profileErr := blueskyFetchWithRetry(ctx, "GET", profileURL, authHeaders, nil)
		if profileErr != nil {
			return nil, fmt.Errorf("bluesky fetch: could not resolve handle: %w", profileErr)
		}
		var profile struct {
			DID string `json:"did"`
		}
		if jsonErr := json.Unmarshal(profileBody, &profile); jsonErr != nil || profile.DID == "" {
			return nil, fmt.Errorf("bluesky fetch: could not resolve DID from handle")
		}
		atURI = "at://" + profile.DID + "/app.bsky.feed.post/" + rkey
	}

	// Fetch the post thread (depth=0 to get only the root post).
	reqURL := blueskyBaseURL + "/xrpc/app.bsky.feed.getPostThread?uri=" +
		url.QueryEscape(atURI) + "&depth=0"

	body, err := blueskyFetchWithRetry(ctx, "GET", reqURL, authHeaders, nil)
	if err != nil {
		if strings.Contains(err.Error(), "Bluesky fetch failed: 400") {
			return nil, fmt.Errorf("bluesky fetch: not found (status 400)")
		}
		return nil, fmt.Errorf("bluesky fetch: %w", err)
	}

	return parseBlueskyThreadResponse(body, authorHandle)
}
