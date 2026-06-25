package pipeline

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestBuildRedditJSONURL tests URL construction from various input formats.
func TestBuildRedditJSONURL(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "full URL with trailing slash",
			input:  "https://www.reddit.com/r/golang/comments/abc/title/",
			expect: "https://www.reddit.com/r/golang/comments/abc/title.json?limit=5&depth=1&sort=top",
		},
		{
			name:   "full URL without trailing slash",
			input:  "https://www.reddit.com/r/golang/comments/abc/title",
			expect: "https://www.reddit.com/r/golang/comments/abc/title.json?limit=5&depth=1&sort=top",
		},
		{
			name:   "permalink with trailing slash",
			input:  "/r/golang/comments/abc/title/",
			expect: "https://www.reddit.com/r/golang/comments/abc/title.json?limit=5&depth=1&sort=top",
		},
		{
			name:   "permalink without trailing slash",
			input:  "/r/golang/comments/abc/title",
			expect: "https://www.reddit.com/r/golang/comments/abc/title.json?limit=5&depth=1&sort=top",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRedditJSONURL(tt.input)
			if got != tt.expect {
				t.Errorf("buildRedditJSONURL(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

// TestParseRedditPostResponse_Success tests parsing a valid Reddit JSON payload.
func TestParseRedditPostResponse_Success(t *testing.T) {
	payload := []byte(`[
		{
			"data": {
				"children": [
					{
						"data": {
							"name": "t3_abc123",
							"title": "Test Title",
							"selftext": "Test selftext",
							"author": "testauthor",
							"permalink": "/r/golang/comments/abc/title/",
							"subreddit": "golang",
							"score": 42,
							"num_comments": 10,
							"created_utc": 1609459200.0,
							"url": "https://example.com"
						}
					}
				]
			}
		},
		{
			"data": {
				"children": [
					{
						"kind": "t1",
						"data": {
							"author": "commenter1",
							"body": "First comment",
							"score": 5
						}
					},
					{
						"kind": "t1",
						"data": {
							"author": "commenter2",
							"body": "Second comment",
							"score": 3
						}
					}
				]
			}
		}
	]`)

	post, err := parseRedditPostResponse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if post.ID != "t3_abc123" {
		t.Errorf("ID: got %q, want %q", post.ID, "t3_abc123")
	}
	if post.Title != "Test Title" {
		t.Errorf("Title: got %q, want %q", post.Title, "Test Title")
	}
	if post.Selftext != "Test selftext" {
		t.Errorf("Selftext: got %q, want %q", post.Selftext, "Test selftext")
	}
	if post.Author != "testauthor" {
		t.Errorf("Author: got %q, want %q", post.Author, "testauthor")
	}
	if post.Permalink != "/r/golang/comments/abc/title/" {
		t.Errorf("Permalink: got %q, want %q", post.Permalink, "/r/golang/comments/abc/title/")
	}
	if post.Subreddit != "golang" {
		t.Errorf("Subreddit: got %q, want %q", post.Subreddit, "golang")
	}
	if post.Score != 42 {
		t.Errorf("Score: got %d, want %d", post.Score, 42)
	}
	if post.NumComments != 10 {
		t.Errorf("NumComments: got %d, want %d", post.NumComments, 10)
	}
	if post.CreatedUTC != 1609459200.0 {
		t.Errorf("CreatedUTC: got %f, want %f", post.CreatedUTC, 1609459200.0)
	}
	if post.URL != "https://example.com" {
		t.Errorf("URL: got %q, want %q", post.URL, "https://example.com")
	}

	if len(post.TopComments) != 2 {
		t.Fatalf("TopComments: expected 2, got %d", len(post.TopComments))
	}
	if post.TopComments[0].Author != "commenter1" {
		t.Errorf("TopComments[0].Author: got %q, want %q", post.TopComments[0].Author, "commenter1")
	}
	if post.TopComments[0].Body != "First comment" {
		t.Errorf("TopComments[0].Body: got %q, want %q", post.TopComments[0].Body, "First comment")
	}
	if post.TopComments[0].Score != 5 {
		t.Errorf("TopComments[0].Score: got %d, want %d", post.TopComments[0].Score, 5)
	}
	if post.TopComments[1].Author != "commenter2" {
		t.Errorf("TopComments[1].Author: got %q, want %q", post.TopComments[1].Author, "commenter2")
	}
	if post.TopComments[1].Body != "Second comment" {
		t.Errorf("TopComments[1].Body: got %q, want %q", post.TopComments[1].Body, "Second comment")
	}
	if post.TopComments[1].Score != 3 {
		t.Errorf("TopComments[1].Score: got %d, want %d", post.TopComments[1].Score, 3)
	}
}

// TestParseRedditPostResponse_MalformedJSON tests error on invalid JSON.
func TestParseRedditPostResponse_MalformedJSON(t *testing.T) {
	_, err := parseRedditPostResponse([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "unexpected response format") {
		t.Errorf("expected error to contain 'unexpected response format', got: %v", err)
	}
}

// TestParseRedditPostResponse_EmptyChildren tests error when post listing has no children.
func TestParseRedditPostResponse_EmptyChildren(t *testing.T) {
	payload := []byte(`[{"data":{"children":[]}},{"data":{"children":[]}}]`)
	_, err := parseRedditPostResponse(payload)
	if err == nil {
		t.Fatal("expected error for empty children")
	}
	if !strings.Contains(err.Error(), "unexpected response format") {
		t.Errorf("expected error to contain 'unexpected response format', got: %v", err)
	}
}

// TestParseRedditPostResponse_NoComments tests success when comment array is empty.
func TestParseRedditPostResponse_NoComments(t *testing.T) {
	payload := []byte(`[
		{
			"data": {
				"children": [
					{
						"data": {
							"name": "t3_xyz",
							"title": "No Comments",
							"selftext": "",
							"author": "author",
							"permalink": "/r/test/comments/xyz/",
							"subreddit": "test",
							"score": 1,
							"num_comments": 0,
							"created_utc": 1609459200.0,
							"url": "https://test.com"
						}
					}
				]
			}
		},
		{
			"data": {
				"children": []
			}
		}
	]`)

	post, err := parseRedditPostResponse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.TopComments == nil {
		t.Fatal("TopComments should be non-nil")
	}
	if len(post.TopComments) != 0 {
		t.Fatalf("TopComments should be empty, got %d", len(post.TopComments))
	}
}

// TestFetchSingleBlueskyPost_MissingAuth tests error when credentials are missing.
func TestFetchSingleBlueskyPost_MissingAuth(t *testing.T) {
	t.Setenv("BLUESKY_HANDLE", "")
	t.Setenv("BLUESKY_APP_PASSWORD", "")

	_, err := FetchSingleBlueskyPost(context.Background(), "https://bsky.app/profile/handle/post/rkey")
	if err == nil {
		t.Fatal("expected error for missing auth")
	}
	if !strings.Contains(err.Error(), "missing BLUESKY_HANDLE or BLUESKY_APP_PASSWORD") {
		t.Errorf("expected error to contain 'missing BLUESKY_HANDLE or BLUESKY_APP_PASSWORD', got: %v", err)
	}
}

// TestParseBlueskyThreadResponse_Success tests parsing a valid Bluesky thread response.
func TestParseBlueskyThreadResponse_Success(t *testing.T) {
	payload := []byte(`{
		"thread": {
			"post": {
				"uri": "at://did:plc:abc123/app.bsky.feed.post/rkey1",
				"cid": "bafytest123",
				"author": {
					"handle": "user.bsky.social",
					"displayName": "Test User"
				},
				"record": {
					"text": "Hello from Bluesky"
				},
				"likeCount": 10,
				"replyCount": 3,
				"repostCount": 2,
				"indexedAt": "2024-01-15T12:00:00Z"
			}
		}
	}`)

	post, err := parseBlueskyThreadResponse(payload, "fallback.handle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if post.ID != "at://did:plc:abc123/app.bsky.feed.post/rkey1" {
		t.Errorf("ID: got %q, want %q", post.ID, "at://did:plc:abc123/app.bsky.feed.post/rkey1")
	}
	if post.CID != "bafytest123" {
		t.Errorf("CID: got %q, want %q", post.CID, "bafytest123")
	}
	if post.Text != "Hello from Bluesky" {
		t.Errorf("Text: got %q, want %q", post.Text, "Hello from Bluesky")
	}
	if post.AuthorHandle != "user.bsky.social" {
		t.Errorf("AuthorHandle: got %q, want %q", post.AuthorHandle, "user.bsky.social")
	}
	if post.AuthorDisplayName != "Test User" {
		t.Errorf("AuthorDisplayName: got %q, want %q", post.AuthorDisplayName, "Test User")
	}
	if post.LikeCount != 10 {
		t.Errorf("LikeCount: got %d, want %d", post.LikeCount, 10)
	}
	if post.ReplyCount != 3 {
		t.Errorf("ReplyCount: got %d, want %d", post.ReplyCount, 3)
	}
	if post.RepostCount != 2 {
		t.Errorf("RepostCount: got %d, want %d", post.RepostCount, 2)
	}
	if post.IndexedAt != "2024-01-15T12:00:00Z" {
		t.Errorf("IndexedAt: got %q, want %q", post.IndexedAt, "2024-01-15T12:00:00Z")
	}
	expectedURL := "https://bsky.app/profile/user.bsky.social/post/rkey1"
	if post.PostURL != expectedURL {
		t.Errorf("PostURL: got %q, want %q", post.PostURL, expectedURL)
	}
}

// TestParseBlueskyThreadResponse_NotFound tests error when post URI is empty.
func TestParseBlueskyThreadResponse_NotFound(t *testing.T) {
	payload := []byte(`{"thread": {"post": {}}}`)
	_, err := parseBlueskyThreadResponse(payload, "")
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to contain 'not found', got: %v", err)
	}
}

// fakeBlueskyTransport intercepts Bluesky HTTP requests and returns canned responses.
type fakeBlueskyTransport struct {
	responses map[string]fakeResponse
}

type fakeResponse struct {
	status int
	body   string
}

func (t *fakeBlueskyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.Method + " " + req.URL.Path
	resp, ok := t.responses[key]
	if !ok {
		// Return a generic 404 for unmatched requests so that unexpected
		// calls (e.g. getProfile for AT-URI paths) surface as errors.
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}
	return &http.Response{
		StatusCode: resp.status,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

// withFakeBlueskyClient temporarily replaces blueskyHTTPClient with a fake.
func withFakeBlueskyClient(t *testing.T, transport *fakeBlueskyTransport) {
	t.Helper()
	origClient := blueskyHTTPClient
	blueskyHTTPClient = &http.Client{Transport: transport}
	t.Cleanup(func() { blueskyHTTPClient = origClient })
}

// TestFetchSingleBlueskyPost_InvalidURL tests error for malformed bsky.app URLs.
func TestFetchSingleBlueskyPost_InvalidURL(t *testing.T) {
	t.Setenv("BLUESKY_HANDLE", "test.handle")
	t.Setenv("BLUESKY_APP_PASSWORD", "test-pass")

	transport := &fakeBlueskyTransport{
		responses: map[string]fakeResponse{
			"POST /xrpc/com.atproto.server.createSession": {
				status: 200,
				body:   `{"accessJwt":"test-jwt","did":"did:plc:testdid"}`,
			},
		},
	}
	withFakeBlueskyClient(t, transport)

	_, err := FetchSingleBlueskyPost(context.Background(), "https://bsky.app/not-a-valid-path")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "invalid URL format") {
		t.Errorf("expected error to contain 'invalid URL format', got: %v", err)
	}
}

// TestFetchSingleBlueskyPost_ATURIFormat tests that at:// URIs bypass URL parsing
// and handle resolution, going straight to getPostThread.
func TestFetchSingleBlueskyPost_ATURIFormat(t *testing.T) {
	t.Setenv("BLUESKY_HANDLE", "test.handle")
	t.Setenv("BLUESKY_APP_PASSWORD", "test-pass")

	atURI := "at://did:plc:testdid/app.bsky.feed.post/rkey123"
	expectedThread := map[string]interface{}{
		"thread": map[string]interface{}{
			"post": map[string]interface{}{
				"uri":    atURI,
				"cid":    "bafycid123",
				"author": map[string]interface{}{"handle": "test.handle", "displayName": "Test"},
				"record": map[string]interface{}{"text": "AT URI post"},
				"likeCount":   5,
				"replyCount":  1,
				"repostCount": 0,
				"indexedAt":   "2024-06-01T00:00:00Z",
			},
		},
	}
	threadJSON, _ := json.Marshal(expectedThread)

	transport := &fakeBlueskyTransport{
		responses: map[string]fakeResponse{
			"POST /xrpc/com.atproto.server.createSession": {
				status: 200,
				body:   `{"accessJwt":"test-jwt","did":"did:plc:testdid"}`,
			},
			"GET /xrpc/app.bsky.feed.getPostThread": {
				status: 200,
				body:   string(threadJSON),
			},
		},
	}
	withFakeBlueskyClient(t, transport)

	post, err := FetchSingleBlueskyPost(context.Background(), atURI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if post.ID != atURI {
		t.Errorf("ID: got %q, want %q", post.ID, atURI)
	}
	if post.Text != "AT URI post" {
		t.Errorf("Text: got %q, want %q", post.Text, "AT URI post")
	}
	if post.AuthorHandle != "test.handle" {
		t.Errorf("AuthorHandle: got %q, want %q", post.AuthorHandle, "test.handle")
	}

	expectedURL := "https://bsky.app/profile/test.handle/post/rkey123"
	if post.PostURL != expectedURL {
		t.Errorf("PostURL: got %q, want %q", post.PostURL, expectedURL)
	}
}
