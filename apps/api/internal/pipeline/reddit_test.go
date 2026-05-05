package pipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// redditListingResponse builds a minimal Reddit listing JSON for the given posts.
func redditListingResponse(posts []map[string]interface{}) []byte {
	children := make([]map[string]interface{}, len(posts))
	for i, p := range posts {
		children[i] = map[string]interface{}{"data": p}
	}
	listing := map[string]interface{}{
		"data": map[string]interface{}{"children": children},
	}
	b, _ := json.Marshal(listing)
	return b
}

func TestFetchRedditPosts_BasicMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify User-Agent header.
		if r.Header.Get("User-Agent") != "Scout/1.0" {
			t.Errorf("expected User-Agent Scout/1.0, got %q", r.Header.Get("User-Agent"))
		}
		body := redditListingResponse([]map[string]interface{}{
			{
				"name":         "t3_abc123",
				"title":        "Test post title",
				"selftext":     "Test post body",
				"author":       "testuser",
				"permalink":    "/r/golang/comments/abc123/test/",
				"subreddit":    "golang",
				"score":        42,
				"num_comments": 7,
				"created_utc":  1700000000.0,
				"url":          "https://www.reddit.com/r/golang/comments/abc123/test/",
			},
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	posts, err := FetchRedditPosts(context.Background(), []string{srv.URL + "/search.json"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	p := posts[0]
	if p.ID != "t3_abc123" {
		t.Errorf("ID: want t3_abc123, got %q", p.ID)
	}
	if p.Title != "Test post title" {
		t.Errorf("Title: want 'Test post title', got %q", p.Title)
	}
	if p.Selftext != "Test post body" {
		t.Errorf("Selftext: want 'Test post body', got %q", p.Selftext)
	}
	if p.Author != "testuser" {
		t.Errorf("Author: want testuser, got %q", p.Author)
	}
	if p.Permalink != "/r/golang/comments/abc123/test/" {
		t.Errorf("Permalink: want '/r/golang/...', got %q", p.Permalink)
	}
	if p.Subreddit != "golang" {
		t.Errorf("Subreddit: want golang, got %q", p.Subreddit)
	}
	if p.Score != 42 {
		t.Errorf("Score: want 42, got %d", p.Score)
	}
	if p.NumComments != 7 {
		t.Errorf("NumComments: want 7, got %d", p.NumComments)
	}
	if p.CreatedUTC != 1700000000.0 {
		t.Errorf("CreatedUTC: want 1700000000, got %f", p.CreatedUTC)
	}
}

func TestFetchRedditPosts_DeduplicatesByName(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		body := redditListingResponse([]map[string]interface{}{
			{"name": "t3_dup", "title": "Duplicate Post", "author": "alice", "score": callCount * 10},
		})
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	posts, err := FetchRedditPosts(context.Background(), []string{srv.URL + "/q1", srv.URL + "/q2"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both queries return same name — should deduplicate to 1
	if len(posts) != 1 {
		t.Fatalf("expected 1 deduplicated post, got %d", len(posts))
	}
}

func TestFetchRedditPosts_ProgressCallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(redditListingResponse(nil))
	}))
	defer srv.Close()

	var progressCalls [][2]int
	onProgress := func(done, total int) {
		progressCalls = append(progressCalls, [2]int{done, total})
	}

	urls := []string{srv.URL + "/q1", srv.URL + "/q2", srv.URL + "/q3"}
	_, err := FetchRedditPosts(context.Background(), urls, onProgress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(progressCalls) != 3 {
		t.Fatalf("expected 3 progress calls, got %d", len(progressCalls))
	}
	if progressCalls[0] != [2]int{1, 3} {
		t.Errorf("first progress call: want {1,3}, got %v", progressCalls[0])
	}
	if progressCalls[2] != [2]int{3, 3} {
		t.Errorf("last progress call: want {3,3}, got %v", progressCalls[2])
	}
}

func TestFetchRedditPosts_ToleratesFailure(t *testing.T) {
	goodCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/good" {
			goodCalled = true
			body := redditListingResponse([]map[string]interface{}{
				{"name": "t3_good", "title": "Good Post", "author": "bob", "score": 5},
			})
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
		} else {
			http.Error(w, "server error", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	posts, err := FetchRedditPosts(context.Background(), []string{srv.URL + "/bad", srv.URL + "/good"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !goodCalled {
		t.Error("expected good URL to be called even after bad URL failed")
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post from good URL, got %d", len(posts))
	}
}
