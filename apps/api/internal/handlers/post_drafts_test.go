package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// --- Fake AI service ---

type fakeAI struct {
	response string
	modelID  string
	err      error
}

func (f *fakeAI) GenerateForTask(_ context.Context, _ string, _ string, _ string) (string, string, error) {
	return f.response, f.modelID, f.err
}

func (f *fakeAI) StripMarkdown(text string) string {
	// Mirror the real StripMarkdown: remove **bold** markers
	// For test purposes just return the text as-is (no markdown in test input).
	return text
}

// --- Fake Reddit context fetcher ---

type fakeReddit struct {
	fullBody    string
	topComments []services.RedditTopComment
	err         error
}

func (f *fakeReddit) FetchRedditContext(_ context.Context, _ string) (services.RedditContext, error) {
	return services.RedditContext{
		FullBody:    f.fullBody,
		TopComments: f.topComments,
	}, f.err
}

// --- Router builder for draft tests ---

func newDraftRouter(t *testing.T, ai services.AIService, reddit services.RedditContextFetcher) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	svc := services.NewPostServiceWithAI(repo, ai, reddit)
	r := chi.NewRouter()
	handlers.MountPostRoutes(r, svc)
	return r, repo
}

// seedPostWithPlatform inserts a post with a specific platform and engagement type.
func seedPostWithPlatform(t *testing.T, repo *repository.Repository, projectID, postID, platform, engagementType string) {
	t.Helper()
	_, err := repo.DB.Exec(
		`INSERT INTO posts (id, project_id, platform, title, body, author, url, subreddit, post_score, final_score, engagement_type, status, why, found_at, scouted_at)
		 VALUES (?, ?, ?, 'Test Title', 'Test body', 'testuser', 'https://reddit.com/r/test/comments/abc', 'testsubreddit', 5.0, 5.0, ?, 'new', 'good post', datetime('now'), datetime('now'))`,
		postID, projectID, platform, engagementType,
	)
	if err != nil {
		t.Fatalf("seed post: %v", err)
	}
}

// seedPrompt inserts a project_prompts row.
func seedPrompt(t *testing.T, repo *repository.Repository, projectID, typ, platform, promptText string) {
	t.Helper()
	_, err := repo.DB.Exec(
		`INSERT INTO project_prompts (project_id, type, platform, prompt_text, created_at, updated_at)
		 VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
		projectID, typ, platform, promptText,
	)
	if err != nil {
		t.Fatalf("seed prompt: %v", err)
	}
}

// --- Tests ---

func TestGenerateDraft_MissingPost_Returns404(t *testing.T) {
	router, repo := newDraftRouter(t, &fakeAI{response: "draft text", modelID: "test-model"}, &fakeReddit{})
	seedProject(t, repo, "proj1")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj1/posts/nonexistent/generate-draft", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "Post not found" {
		t.Fatalf("error = %q, want 'Post not found'", result["error"])
	}
}

func TestGenerateDraft_MissingKarmaPromptReddit_Returns400(t *testing.T) {
	router, repo := newDraftRouter(t, &fakeAI{response: "draft", modelID: "m"}, &fakeReddit{})
	seedProject(t, repo, "proj2")
	seedPostWithPlatform(t, repo, "proj2", "post2", "reddit", "karma")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj2/posts/post2/generate-draft", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "No karma prompt configured for reddit" {
		t.Fatalf("error = %q", result["error"])
	}
}

func TestGenerateDraft_MissingProductPromptBluesky_Returns400(t *testing.T) {
	router, repo := newDraftRouter(t, &fakeAI{response: "draft", modelID: "m"}, &fakeReddit{})
	seedProject(t, repo, "proj3")
	seedPostWithPlatform(t, repo, "proj3", "post3", "bluesky", "product")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj3/posts/post3/generate-draft", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "No product prompt configured for bluesky" {
		t.Fatalf("error = %q", result["error"])
	}
}

func TestGenerateDraft_Reddit_UserMessageIncludesTitleSubredditWhyBodyComments(t *testing.T) {
	capturedUserMsg := ""

	// Build custom fake that captures the user message
	captureFake := &capturingFakeAI{response: "generated draft", modelID: "test-model", capture: &capturedUserMsg}
	reddit := &fakeReddit{
		fullBody: "full reddit body",
		topComments: []services.RedditTopComment{
			{Author: "commenter1", Body: "top comment body", Score: 42},
		},
	}

	router, repo := newDraftRouter(t, captureFake, reddit)
	seedProject(t, repo, "proj4")
	seedPostWithPlatform(t, repo, "proj4", "post4", "reddit", "karma")
	seedPrompt(t, repo, "proj4", "karma", "reddit", "You are a Reddit commenter")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj4/posts/post4/generate-draft", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	// Check user message includes required fields
	for _, want := range []string{"Title:", "Subreddit:", "Why this post:", "Body:", "Top comments:"} {
		if !containsString(capturedUserMsg, want) {
			t.Fatalf("userMessage missing %q\nfull message: %s", want, capturedUserMsg)
		}
	}
	if !containsString(capturedUserMsg, "commenter1") {
		t.Fatalf("userMessage missing comment author\nfull message: %s", capturedUserMsg)
	}
}

func TestGenerateDraft_Bluesky_UserMessageIncludesAuthorWhyBody(t *testing.T) {
	capturedUserMsg := ""
	captureFake := &capturingFakeAI{response: "bluesky draft", modelID: "bsky-model", capture: &capturedUserMsg}

	router, repo := newDraftRouter(t, captureFake, &fakeReddit{})
	seedProject(t, repo, "proj5")
	seedPostWithPlatform(t, repo, "proj5", "post5", "bluesky", "karma")
	seedPrompt(t, repo, "proj5", "karma", "bluesky", "You are a Bluesky commenter")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj5/posts/post5/generate-draft", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	for _, want := range []string{"Author:", "Why this post:", "Body:"} {
		if !containsString(capturedUserMsg, want) {
			t.Fatalf("userMessage missing %q\nfull message: %s", want, capturedUserMsg)
		}
	}
}

func TestGenerateDraft_StoredInDraftCommentProviderStatus(t *testing.T) {
	ai := &fakeAI{response: "**bold** draft", modelID: "test-model"}
	reddit := &fakeReddit{}

	router, repo := newDraftRouter(t, ai, reddit)
	seedProject(t, repo, "proj6")
	seedPostWithPlatform(t, repo, "proj6", "post6", "reddit", "karma")
	seedPrompt(t, repo, "proj6", "karma", "reddit", "system prompt")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj6/posts/post6/generate-draft", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)

	if result["status"] != "drafted" {
		t.Fatalf("status = %q, want drafted", result["status"])
	}
	// fakeAI.StripMarkdown returns text as-is, so draft_comment = "**bold** draft"
	if result["draft_comment"] != "**bold** draft" {
		t.Fatalf("draft_comment = %q", result["draft_comment"])
	}
	if result["draft_provider"] != "test-model" {
		t.Fatalf("draft_provider = %q, want test-model", result["draft_provider"])
	}
}

func TestGenerateDMDraft_MissingPost_Returns404(t *testing.T) {
	router, repo := newDraftRouter(t, &fakeAI{response: "dm draft", modelID: "m"}, &fakeReddit{})
	seedProject(t, repo, "proj7")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj7/posts/nonexistent/dm/user1/generate-draft", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "Post not found" {
		t.Fatalf("error = %q", result["error"])
	}
}

func TestGenerateDMDraft_MissingDMPromptReddit_Returns400(t *testing.T) {
	router, repo := newDraftRouter(t, &fakeAI{response: "dm", modelID: "m"}, &fakeReddit{})
	seedProject(t, repo, "proj8")
	seedPostWithPlatform(t, repo, "proj8", "post8", "reddit", "karma")
	seedDMTarget(t, repo, "post8", "targetuser", 8.0)

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj8/posts/post8/dm/targetuser/generate-draft", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "No DM prompt configured for reddit" {
		t.Fatalf("error = %q", result["error"])
	}
}

func TestGenerateDMDraft_StoresDraftInDMTarget(t *testing.T) {
	ai := &fakeAI{response: "hello DM", modelID: "dm-model"}
	router, repo := newDraftRouter(t, ai, &fakeReddit{})
	seedProject(t, repo, "proj9")
	seedPostWithPlatform(t, repo, "proj9", "post9", "reddit", "karma")
	seedDMTarget(t, repo, "post9", "dmuser", 9.0)
	seedPrompt(t, repo, "proj9", "dm", "reddit", "dm system prompt")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/proj9/posts/post9/dm/dmuser/generate-draft", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["draft_dm"] != "hello DM" {
		t.Fatalf("draft_dm = %q, want 'hello DM'", result["draft_dm"])
	}
}

// --- capturingFakeAI captures user messages for assertion ---

type capturingFakeAI struct {
	response string
	modelID  string
	capture  *string
}

func (c *capturingFakeAI) GenerateForTask(_ context.Context, _ string, _ string, userMessage string) (string, string, error) {
	if c.capture != nil {
		*c.capture = userMessage
	}
	return c.response, c.modelID, nil
}

func (c *capturingFakeAI) StripMarkdown(text string) string { return text }

func containsString(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
