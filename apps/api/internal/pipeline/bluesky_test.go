package pipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// blueskyTestServer sets up an httptest server that handles createSession and searchPosts.
func blueskyTestServer(t *testing.T, posts []map[string]interface{}) (*httptest.Server, func()) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/xrpc/com.atproto.server.createSession", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST for createSession, got %s", r.Method)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["identifier"] == "" || body["password"] == "" {
			t.Error("expected identifier and password in createSession body")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"accessJwt": "test-jwt-token", "did": "did:plc:testdid"})
	})

	mux.HandleFunc("/xrpc/app.bsky.feed.searchPosts", func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header present
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("expected Bearer token in Authorization, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"posts": posts})
	})

	srv := httptest.NewServer(mux)
	return srv, srv.Close
}

func TestFetchBlueskyPosts_BasicMapping(t *testing.T) {
	testPosts := []map[string]interface{}{
		{
			"uri": "at://did:plc:abc123/app.bsky.feed.post/rkey1",
			"cid": "bafytest123",
			"author": map[string]interface{}{
				"handle":      "user.bsky.social",
				"displayName": "Test User",
			},
			"record": map[string]interface{}{
				"text": "Hello from Bluesky",
			},
			"likeCount":   10,
			"replyCount":  3,
			"repostCount": 2,
			"indexedAt":   "2024-01-15T12:00:00Z",
		},
	}
	srv, cleanup := blueskyTestServer(t, testPosts)
	defer cleanup()

	// Override base URL via environment + httptest
	origClient := blueskyHTTPClient
	blueskyHTTPClient = srv.Client()
	defer func() { blueskyHTTPClient = origClient }()

	// Override the base URL constant by patching the server URL in blueskyAuthenticate/search.
	// Since blueskyBaseURL is a const, we test via an env-based override pattern.
	// Instead, we use the server URL directly in a wrapper.
	t.Setenv("BLUESKY_HANDLE", "test.bsky.social")
	t.Setenv("BLUESKY_APP_PASSWORD", "test-password")

	// We need to redirect requests to our test server. Override blueskyBaseURL via
	// a transport that rewrites the host.
	blueskyHTTPClient = &http.Client{
		Transport: &rewriteTransport{base: srv.URL, inner: http.DefaultTransport},
	}
	defer func() { blueskyHTTPClient = origClient }()

	posts, err := FetchBlueskyPosts(context.Background(), []string{"hello world"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	p := posts[0]
	if p.ID != "at://did:plc:abc123/app.bsky.feed.post/rkey1" {
		t.Errorf("URI: got %q", p.ID)
	}
	if p.CID != "bafytest123" {
		t.Errorf("CID: got %q", p.CID)
	}
	if p.Text != "Hello from Bluesky" {
		t.Errorf("Text: got %q", p.Text)
	}
	if p.AuthorHandle != "user.bsky.social" {
		t.Errorf("AuthorHandle: got %q", p.AuthorHandle)
	}
	if p.AuthorDisplayName != "Test User" {
		t.Errorf("AuthorDisplayName: got %q", p.AuthorDisplayName)
	}
	if p.LikeCount != 10 {
		t.Errorf("LikeCount: got %d", p.LikeCount)
	}
	if p.ReplyCount != 3 {
		t.Errorf("ReplyCount: got %d", p.ReplyCount)
	}
	if p.RepostCount != 2 {
		t.Errorf("RepostCount: got %d", p.RepostCount)
	}
	if p.IndexedAt != "2024-01-15T12:00:00Z" {
		t.Errorf("IndexedAt: got %q", p.IndexedAt)
	}
	expectedURL := "https://bsky.app/profile/user.bsky.social/post/rkey1"
	if p.PostURL != expectedURL {
		t.Errorf("PostURL: want %q, got %q", expectedURL, p.PostURL)
	}
}

func TestFetchBlueskyPosts_MissingHandle(t *testing.T) {
	t.Setenv("BLUESKY_HANDLE", "")
	t.Setenv("BLUESKY_APP_PASSWORD", "password")

	_, err := FetchBlueskyPosts(context.Background(), []string{"query"}, nil)
	if err == nil {
		t.Fatal("expected error for missing BLUESKY_HANDLE")
	}
	if !strings.Contains(err.Error(), "BLUESKY_HANDLE") {
		t.Errorf("expected error to mention BLUESKY_HANDLE, got: %v", err)
	}
}

func TestFetchBlueskyPosts_MissingPassword(t *testing.T) {
	t.Setenv("BLUESKY_HANDLE", "test.bsky.social")
	t.Setenv("BLUESKY_APP_PASSWORD", "")

	_, err := FetchBlueskyPosts(context.Background(), []string{"query"}, nil)
	if err == nil {
		t.Fatal("expected error for missing BLUESKY_APP_PASSWORD")
	}
	if !strings.Contains(err.Error(), "BLUESKY_APP_PASSWORD") {
		t.Errorf("expected error to mention BLUESKY_APP_PASSWORD, got: %v", err)
	}
}

func TestFetchBlueskyPosts_DeduplicatesByURI(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/xrpc/com.atproto.server.createSession", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"accessJwt": "jwt", "did": "did:plc:testdid"})
	})
	mux.HandleFunc("/xrpc/app.bsky.feed.searchPosts", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		posts := []map[string]interface{}{
			{
				"uri":    "at://did:plc:abc/app.bsky.feed.post/dup",
				"cid":    "cid1",
				"author": map[string]interface{}{"handle": "user.bsky.social"},
				"record": map[string]interface{}{"text": "dup post"},
			},
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"posts": posts})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := blueskyHTTPClient
	blueskyHTTPClient = &http.Client{
		Transport: &rewriteTransport{base: srv.URL, inner: http.DefaultTransport},
	}
	defer func() { blueskyHTTPClient = origClient }()

	t.Setenv("BLUESKY_HANDLE", "h")
	t.Setenv("BLUESKY_APP_PASSWORD", "p")

	posts, err := FetchBlueskyPosts(context.Background(), []string{"q1", "q2"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 deduplicated post, got %d", len(posts))
	}
}

func TestFetchBlueskyPosts_ProgressCallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/xrpc/com.atproto.server.createSession", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"accessJwt": "jwt", "did": "did:plc:testdid"})
	})
	mux.HandleFunc("/xrpc/app.bsky.feed.searchPosts", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"posts": []interface{}{}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := blueskyHTTPClient
	blueskyHTTPClient = &http.Client{
		Transport: &rewriteTransport{base: srv.URL, inner: http.DefaultTransport},
	}
	defer func() { blueskyHTTPClient = origClient }()

	t.Setenv("BLUESKY_HANDLE", "h")
	t.Setenv("BLUESKY_APP_PASSWORD", "p")

	var progressCalls [][2]int
	queries := []string{"q1", "q2", "q3"}
	_, err := FetchBlueskyPosts(context.Background(), queries, func(done, total int) {
		progressCalls = append(progressCalls, [2]int{done, total})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(progressCalls) != 3 {
		t.Fatalf("expected 3 progress calls, got %d", len(progressCalls))
	}
	if progressCalls[0] != [2]int{1, 3} {
		t.Errorf("first progress: want {1,3}, got %v", progressCalls[0])
	}
	if progressCalls[2] != [2]int{3, 3} {
		t.Errorf("last progress: want {3,3}, got %v", progressCalls[2])
	}
}

func TestFetchBlueskyPosts_ToleratesSearchFailure(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/xrpc/com.atproto.server.createSession", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"accessJwt": "jwt", "did": "did:plc:testdid"})
	})
	mux.HandleFunc("/xrpc/app.bsky.feed.searchPosts", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		q := r.URL.Query().Get("q")
		if q == "bad" {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		posts := []map[string]interface{}{
			{
				"uri":    "at://did:plc:abc/app.bsky.feed.post/good1",
				"cid":    "cid1",
				"author": map[string]interface{}{"handle": "good.bsky.social"},
				"record": map[string]interface{}{"text": "good post"},
			},
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"posts": posts})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := blueskyHTTPClient
	blueskyHTTPClient = &http.Client{
		Transport: &rewriteTransport{base: srv.URL, inner: http.DefaultTransport},
	}
	defer func() { blueskyHTTPClient = origClient }()

	t.Setenv("BLUESKY_HANDLE", "h")
	t.Setenv("BLUESKY_APP_PASSWORD", "p")

	posts, err := FetchBlueskyPosts(context.Background(), []string{"bad", "good"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post from good query, got %d", len(posts))
	}
}

// rewriteTransport redirects all requests to a fixed base URL (for test servers).
type rewriteTransport struct {
	base  string
	inner http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace scheme+host with test server base.
	newURL := *req.URL
	baseURL := strings.TrimRight(t.base, "/")
	// parse base to get scheme and host
	parts := strings.SplitN(baseURL, "://", 2)
	if len(parts) == 2 {
		hostParts := strings.SplitN(parts[1], "/", 2)
		newURL.Scheme = parts[0]
		newURL.Host = hostParts[0]
	}
	newReq := req.Clone(req.Context())
	newReq.URL = &newURL
	return t.inner.RoundTrip(newReq)
}

func TestFetchBlueskyReplies_MapsTopLevelReplies(t *testing.T) {
	var badRequest string
	mux := http.NewServeMux()
	mux.HandleFunc("/xrpc/app.bsky.feed.getPostThread", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("uri") != "at://did:plc:abc/app.bsky.feed.post/root" {
			badRequest = "unexpected uri query: " + r.URL.RawQuery
		}
		if r.URL.Query().Get("depth") != "1" {
			badRequest = "expected depth=1, got " + r.URL.Query().Get("depth")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"thread": map[string]any{
				"replies": []map[string]any{
					{
						"post": map[string]any{
							"author":    map[string]any{"handle": "first.bsky.social"},
							"record":    map[string]any{"text": "I need a better tool"},
							"likeCount": 7,
							"indexedAt": "2026-06-01T12:00:00Z",
						},
					},
					{
						"post": map[string]any{
							"author":    map[string]any{"handle": "second.bsky.social"},
							"record":    map[string]any{"text": "Does anyone recommend something?"},
							"likeCount": 2,
							"indexedAt": "2026-06-02T12:00:00Z",
						},
					},
				},
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := blueskyHTTPClient
	blueskyHTTPClient = &http.Client{Transport: &rewriteTransport{base: srv.URL, inner: http.DefaultTransport}}
	defer func() { blueskyHTTPClient = origClient }()

	replies, err := FetchBlueskyReplies(context.Background(), "at://did:plc:abc/app.bsky.feed.post/root")
	if badRequest != "" {
		t.Fatalf("bad request to handler: %s", badRequest)
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(replies) != 2 {
		t.Fatalf("expected 2 replies, got %d", len(replies))
	}
	if replies[0].AuthorHandle != "first.bsky.social" || replies[0].Text != "I need a better tool" || replies[0].LikeCount != 7 || replies[0].IndexedAt != "2026-06-01T12:00:00Z" {
		t.Fatalf("unexpected first reply: %#v", replies[0])
	}
	if replies[1].AuthorHandle != "second.bsky.social" || replies[1].Text != "Does anyone recommend something?" || replies[1].LikeCount != 2 || replies[1].IndexedAt != "2026-06-02T12:00:00Z" {
		t.Fatalf("unexpected second reply: %#v", replies[1])
	}
}

func TestFetchBlueskyReplies_MapsTopLevelRepliesAndSkipsDeletedStubs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/xrpc/app.bsky.feed.getPostThread", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("uri") != "at://did:plc:abc/app.bsky.feed.post/root" {
			t.Fatalf("unexpected uri query: %q", r.URL.Query().Get("uri"))
		}
		if r.URL.Query().Get("depth") != "1" {
			t.Fatalf("depth = %q, want 1", r.URL.Query().Get("depth"))
		}
		json.NewEncoder(w).Encode(map[string]any{"thread": map[string]any{"replies": []any{
			map[string]any{"post": map[string]any{"author": map[string]any{"handle": "first.bsky.social"}, "record": map[string]any{"text": "I need a tool for this"}, "likeCount": 4, "indexedAt": "2026-06-01T10:00:00Z"}},
			map[string]any{"post": map[string]any{"author": map[string]any{"handle": ""}, "record": map[string]any{"text": "deleted"}}},
			map[string]any{"post": map[string]any{"author": map[string]any{"handle": "second.bsky.social"}, "record": map[string]any{"text": "Can someone recommend an app?"}, "likeCount": 2, "indexedAt": "2026-06-01T11:00:00Z"}},
		}}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	origClient := blueskyHTTPClient
	blueskyHTTPClient = &http.Client{Transport: &rewriteTransport{base: srv.URL, inner: http.DefaultTransport}}
	defer func() { blueskyHTTPClient = origClient }()

	replies, err := FetchBlueskyReplies(context.Background(), "at://did:plc:abc/app.bsky.feed.post/root")

	if err != nil {
		t.Fatalf("FetchBlueskyReplies: %v", err)
	}
	if len(replies) != 2 {
		t.Fatalf("expected 2 non-deleted replies, got %#v", replies)
	}
	if replies[0].AuthorHandle != "first.bsky.social" || replies[0].Text != "I need a tool for this" || replies[0].LikeCount != 4 || replies[0].IndexedAt != "2026-06-01T10:00:00Z" {
		t.Fatalf("unexpected first reply: %#v", replies[0])
	}
}

func TestPostBlueskyReply_SendsCorrectRequest(t *testing.T) {
	var capturedBody map[string]interface{}
	authedDID := "did:plc:authuser"

	mux := http.NewServeMux()
	mux.HandleFunc("/xrpc/com.atproto.server.createSession", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"accessJwt": "test-jwt", "did": authedDID})
	})
	mux.HandleFunc("/xrpc/com.atproto.repo.createRecord", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-jwt" {
			t.Errorf("expected Bearer test-jwt, got %q", auth)
		}
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"uri": "at://did:plc:authuser/app.bsky.feed.post/reply1", "cid": "bafyreply"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	origClient := blueskyHTTPClient
	blueskyHTTPClient = &http.Client{
		Transport: &rewriteTransport{base: srv.URL, inner: http.DefaultTransport},
	}
	defer func() { blueskyHTTPClient = origClient }()

	result, err := PostBlueskyReply(context.Background(), "user.bsky.social", "app-password", "hello!", "at://did:plc:target/app.bsky.feed.post/orig", "bafyorigcid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify request body structure
	if capturedBody["repo"] != authedDID {
		t.Errorf("repo = %q, want %q", capturedBody["repo"], authedDID)
	}
	if capturedBody["collection"] != "app.bsky.feed.post" {
		t.Errorf("collection = %q, want 'app.bsky.feed.post'", capturedBody["collection"])
	}
	record, ok := capturedBody["record"].(map[string]interface{})
	if !ok {
		t.Fatalf("record missing or wrong type: %v", capturedBody["record"])
	}
	if record["text"] != "hello!" {
		t.Errorf("record.text = %q, want 'hello!'", record["text"])
	}
	reply, ok := record["reply"].(map[string]interface{})
	if !ok {
		t.Fatalf("record.reply missing: %v", record["reply"])
	}
	parent, _ := reply["parent"].(map[string]interface{})
	if parent["uri"] != "at://did:plc:target/app.bsky.feed.post/orig" {
		t.Errorf("parent.uri = %q", parent["uri"])
	}
	if parent["cid"] != "bafyorigcid" {
		t.Errorf("parent.cid = %q", parent["cid"])
	}
}
