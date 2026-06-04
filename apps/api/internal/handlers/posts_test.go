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

func newPostRouter(t *testing.T) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	svc := services.NewPostService(repo)
	r := chi.NewRouter()
	handlers.MountPostRoutes(r, svc)
	return r, repo
}

// newPostRouterWithBsky creates a router with a fake Bluesky replier injected.
func newPostRouterWithBsky(t *testing.T, replier services.BlueskyReplier) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	svc := services.NewPostServiceFull(repo, nil, nil, replier)
	r := chi.NewRouter()
	handlers.MountPostRoutes(r, svc)
	return r, repo
}

// seedProject inserts a minimal project row.
func seedProject(t *testing.T, repo *repository.Repository, id string) {
	t.Helper()
	_, err := repo.DB.Exec(
		`INSERT INTO projects (id, name, mode, created_at, updated_at) VALUES (?, ?, 'research', datetime('now'), datetime('now'))`,
		id, id+"-name",
	)
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
}

// seedPost inserts a minimal post row.
func seedPost(t *testing.T, repo *repository.Repository, projectID, postID string) {
	t.Helper()
	_, err := repo.DB.Exec(
		`INSERT INTO posts (id, project_id, platform, title, body, author, url, post_score, final_score, engagement_type, status, found_at, scouted_at)
		 VALUES (?, ?, 'reddit', 'Test Post', 'body', 'user', 'http://example.com', 5.0, 5.0, 'karma', 'new', datetime('now'), datetime('now'))`,
		postID, projectID,
	)
	if err != nil {
		t.Fatalf("seed post: %v", err)
	}
}

// seedDMTarget inserts a dm_target row.
func seedDMTarget(t *testing.T, repo *repository.Repository, postID, username string, intentScore float64) int64 {
	t.Helper()
	res, err := repo.DB.Exec(
		`INSERT INTO dm_targets (post_id, username, intent_score, signal, context, approach, dm_status)
		 VALUES (?, ?, ?, 'signal', 'ctx', 'approach', 'new')`,
		postID, username, intentScore,
	)
	if err != nil {
		t.Fatalf("seed dm_target: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// --- Tests ---

func TestPostList_NoPagination_ReturnsArray(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p1")
	seedPost(t, repo, "p1", "post1")

	rr := doRequest(t, router, http.MethodGet, "/api/projects/p1/posts", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result []any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("expected array, got: %s", rr.Body.String())
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 post, got %d", len(result))
	}
}

func TestPostList_WithPagination_ReturnsPaginatedObject(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p2")
	seedPost(t, repo, "p2", "post-a")
	seedPost(t, repo, "p2", "post-b")
	seedPost(t, repo, "p2", "post-c")

	rr := doRequest(t, router, http.MethodGet, "/api/projects/p2/posts?page=1&limit=2", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("expected object: %s", rr.Body.String())
	}
	if _, ok := result["items"]; !ok {
		t.Fatal("missing 'items' key")
	}
	pagination, ok := result["pagination"].(map[string]any)
	if !ok {
		t.Fatalf("missing 'pagination' key, got: %v", result)
	}
	// camelCase keys
	for _, key := range []string{"page", "limit", "total", "totalPages", "hasPreviousPage", "hasNextPage"} {
		if _, exists := pagination[key]; !exists {
			t.Fatalf("pagination missing key %q", key)
		}
	}
	items := result["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if pagination["hasNextPage"].(bool) != true {
		t.Fatalf("expected hasNextPage=true")
	}
}

func TestPostGet_SinglePost_AttachesDMTargets(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p3")
	seedPost(t, repo, "p3", "post3")
	seedDMTarget(t, repo, "post3", "user_high", 9.0)
	seedDMTarget(t, repo, "post3", "user_low", 3.0)

	rr := doRequest(t, router, http.MethodGet, "/api/projects/p3/posts/post3", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	targets, ok := result["dm_targets"].([]any)
	if !ok {
		t.Fatalf("expected dm_targets array, got: %v", result["dm_targets"])
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 dm_targets, got %d", len(targets))
	}
	// first should be user_high (highest intent_score)
	first := targets[0].(map[string]any)
	if first["username"] != "user_high" {
		t.Fatalf("expected user_high first, got %v", first["username"])
	}
}

func TestPostGet_NotFound(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p4")

	rr := doRequest(t, router, http.MethodGet, "/api/projects/p4/posts/nonexistent", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "Post not found" {
		t.Fatalf("error = %q", result["error"])
	}
}

func TestBulkPatch_InvalidBody_Returns400(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p5")

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p5/posts/bulk", map[string]any{})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "ids (array) and status are required" {
		t.Fatalf("error = %q", result["error"])
	}
}

func TestBulkPatch_InvalidStatus_Returns400(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p6")
	seedPost(t, repo, "p6", "post6")

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p6/posts/bulk",
		map[string]any{"ids": []string{"post6"}, "status": "invalid_status"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	want := "Invalid status. Must be one of: new, drafted, commented, skipped, reviewed, starred, excluded"
	if result["error"] != want {
		t.Fatalf("error = %q, want %q", result["error"], want)
	}
}

func TestBulkPatch_ValidRequest_ReturnsUpdatedCount(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p7")
	seedPost(t, repo, "p7", "post7a")
	seedPost(t, repo, "p7", "post7b")

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p7/posts/bulk",
		map[string]any{"ids": []string{"post7a", "post7b"}, "status": "starred"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["updated"].(float64) != 2 {
		t.Fatalf("updated = %v, want 2", result["updated"])
	}
}

func TestPatchPost_DraftComment_AutoSetsDrafted(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p8")
	seedPost(t, repo, "p8", "post8")

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p8/posts/post8",
		map[string]any{"draft_comment": "my draft"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["status"] != "drafted" {
		t.Fatalf("status = %q, want drafted", result["status"])
	}
	if result["draft_comment"] != "my draft" {
		t.Fatalf("draft_comment = %q", result["draft_comment"])
	}
}

func TestPatchPost_ExplicitStatus_OverridesDrafted(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p9")
	seedPost(t, repo, "p9", "post9")

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p9/posts/post9",
		map[string]any{"draft_comment": "comment", "status": "starred"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["status"] != "starred" {
		t.Fatalf("status = %q, want starred", result["status"])
	}
}

func TestPatchDMTarget_UpdatesTarget(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "p10")
	seedPost(t, repo, "p10", "post10")
	seedDMTarget(t, repo, "post10", "targetuser", 7.5)

	draft := "hello DM"
	rr := doRequest(t, router, http.MethodPatch, "/api/projects/p10/posts/post10/dm/targetuser",
		map[string]any{"draft_dm": draft})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rr.Code, rr.Body.String())
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["draft_dm"] != draft {
		t.Fatalf("draft_dm = %q, want %q", result["draft_dm"], draft)
	}
}

// --- seedBlueskyPost inserts a post with bluesky_uri and bluesky_cid set ---

func seedBlueskyPost(t *testing.T, repo *repository.Repository, projectID, postID string) {
	t.Helper()
	_, err := repo.DB.Exec(
		`INSERT INTO posts (id, project_id, platform, title, body, author, url, post_score, final_score, engagement_type, status, bluesky_uri, bluesky_cid, found_at, scouted_at)
		 VALUES (?, ?, 'bluesky', 'Bluesky Post', 'body', 'user.bsky.social', 'https://bsky.app/profile/user.bsky.social/post/abc', 5.0, 5.0, 'karma', 'new', 'at://did:plc:abc/app.bsky.feed.post/abc', 'bafyreiexample', datetime('now'), datetime('now'))`,
		postID, projectID,
	)
	if err != nil {
		t.Fatalf("seed bluesky post: %v", err)
	}
}

// --- Post-reply tests ---

func TestPostReply_MissingText_Returns400(t *testing.T) {
	succeedReplier := services.BlueskyReplierFunc(func(_ context.Context, _, _, _, _, _ string) (json.RawMessage, error) {
		return json.RawMessage(`{}`), nil
	})
	router, repo := newPostRouterWithBsky(t, succeedReplier)
	seedProject(t, repo, "pr1")
	seedBlueskyPost(t, repo, "pr1", "bsky1")
	t.Setenv("BLUESKY_HANDLE", "handle.bsky.social")
	t.Setenv("BLUESKY_PASSWORD", "app-password")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/pr1/posts/bsky1/post-reply",
		map[string]any{})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400\nbody: %s", rr.Code, rr.Body.String())
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "text is required" {
		t.Fatalf("error = %q, want 'text is required'", result["error"])
	}
}

func TestPostReply_PostNotFound_Returns404(t *testing.T) {
	succeedReplier := services.BlueskyReplierFunc(func(_ context.Context, _, _, _, _, _ string) (json.RawMessage, error) {
		return json.RawMessage(`{}`), nil
	})
	router, repo := newPostRouterWithBsky(t, succeedReplier)
	seedProject(t, repo, "pr2")
	t.Setenv("BLUESKY_HANDLE", "handle.bsky.social")
	t.Setenv("BLUESKY_PASSWORD", "app-password")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/pr2/posts/nonexistent/post-reply",
		map[string]any{"text": "hello"})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404\nbody: %s", rr.Code, rr.Body.String())
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "Post not found" {
		t.Fatalf("error = %q, want 'Post not found'", result["error"])
	}
}

func TestPostReply_MissingBlueskyMeta_Returns400(t *testing.T) {
	succeedReplier := services.BlueskyReplierFunc(func(_ context.Context, _, _, _, _, _ string) (json.RawMessage, error) {
		return json.RawMessage(`{}`), nil
	})
	router, repo := newPostRouterWithBsky(t, succeedReplier)
	seedProject(t, repo, "pr3")
	// seedPost creates a reddit post — no bluesky_uri/cid
	seedPost(t, repo, "pr3", "reddit-post")
	t.Setenv("BLUESKY_HANDLE", "handle.bsky.social")
	t.Setenv("BLUESKY_PASSWORD", "app-password")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/pr3/posts/reddit-post/post-reply",
		map[string]any{"text": "hello"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400\nbody: %s", rr.Code, rr.Body.String())
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "Post is missing Bluesky uri or cid" {
		t.Fatalf("error = %q, want 'Post is missing Bluesky uri or cid'", result["error"])
	}
}

func TestPostReply_MissingEnvVars_Returns500(t *testing.T) {
	succeedReplier := services.BlueskyReplierFunc(func(_ context.Context, _, _, _, _, _ string) (json.RawMessage, error) {
		return json.RawMessage(`{}`), nil
	})
	router, repo := newPostRouterWithBsky(t, succeedReplier)
	seedProject(t, repo, "pr4")
	seedBlueskyPost(t, repo, "pr4", "bsky4")
	t.Setenv("BLUESKY_HANDLE", "")
	t.Setenv("BLUESKY_PASSWORD", "")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/pr4/posts/bsky4/post-reply",
		map[string]any{"text": "hello"})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500\nbody: %s", rr.Code, rr.Body.String())
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["error"] != "BLUESKY_HANDLE or BLUESKY_PASSWORD not configured" {
		t.Fatalf("error = %q", result["error"])
	}
}

func TestPostList_DefaultExcludesFilteredPosts(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pf1")
	seedPost(t, repo, "pf1", "visible-post")
	seedPost(t, repo, "pf1", "filtered-post")
	_, err := repo.DB.Exec(`UPDATE posts SET filter_state = 'filtered', filter_reason = 'spam', filter_reasons_json = '["spam"]', filter_explanation = 'test', filter_source = 'rules', filtered_at = datetime('now') WHERE id = 'filtered-post'`)
	if err != nil { t.Fatal(err) }

	rr := doRequest(t, router, http.MethodGet, "/api/projects/pf1/posts?status=new", nil)
	if rr.Code != http.StatusOK { t.Fatalf("status = %d", rr.Code) }
	var result []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil { t.Fatal(err) }
	if len(result) != 1 || result[0]["id"] != "visible-post" { t.Fatalf("result = %#v", result) }
}

func TestPostGet_FilteredPost_IsStillReachableByID(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pf2")
	seedPost(t, repo, "pf2", "filtered-id-post")
	_, err := repo.DB.Exec(`UPDATE posts SET filter_state = 'filtered', filter_reason = 'spam', filter_reasons_json = '["spam"]', filter_explanation = 'test', filter_source = 'rules', filtered_at = datetime('now') WHERE id = 'filtered-id-post'`)
	if err != nil { t.Fatal(err) }

	rr := doRequest(t, router, http.MethodGet, "/api/projects/pf2/posts/filtered-id-post", nil)
	if rr.Code != http.StatusOK { t.Fatalf("status = %d, want 200", rr.Code) }
	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil { t.Fatal(err) }
	if result["filter_state"] != "filtered" { t.Fatalf("filter_state = %v", result["filter_state"]) }
}

func TestPostList_DMFilter_ReturnsOnlyPostsWithTargets(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pdm1")

	// post with a dm_target
	seedPost(t, repo, "pdm1", "post-with-target")
	seedDMTarget(t, repo, "post-with-target", "someuser", 8.0)
	_, err := repo.DB.Exec(`UPDATE posts SET filter_state = 'visible' WHERE id = 'post-with-target'`)
	if err != nil {
		t.Fatal(err)
	}

	// post without any dm_target
	seedPost(t, repo, "pdm1", "post-no-target")

	// filtered post with a target should still be excluded by the visible clause
	seedPost(t, repo, "pdm1", "post-filtered-target")
	seedDMTarget(t, repo, "post-filtered-target", "hiddenuser", 9.0)
	_, err = repo.DB.Exec(`UPDATE posts SET filter_state = 'filtered', filter_reason = 'spam', filter_reasons_json = '["spam"]', filter_explanation = 'test', filter_source = 'rules', filtered_at = datetime('now') WHERE id = 'post-filtered-target'`)
	if err != nil {
		t.Fatal(err)
	}

	rr := doRequest(t, router, http.MethodGet, "/api/projects/pdm1/posts?dm=true&page=1&limit=20", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rr.Code, rr.Body.String())
	}
	var result struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("expected paginated response: %s", rr.Body.String())
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 visible post with targets, got %d", len(result.Items))
	}
	if result.Items[0]["id"] != "post-with-target" {
		t.Fatalf("expected post-with-target, got %v", result.Items[0]["id"])
	}
	if dmTargets, ok := result.Items[0]["dm_targets"].([]any); !ok || len(dmTargets) != 1 {
		t.Fatalf("expected attached dm_targets, got %#v", result.Items[0]["dm_targets"])
	}
}

func TestPostReply_Success_SetsCommentedStatus(t *testing.T) {
	succeedReplier := services.BlueskyReplierFunc(func(_ context.Context, _, _, _, _, _ string) (json.RawMessage, error) {
		return json.RawMessage(`{"uri":"at://did:plc:abc/app.bsky.feed.post/reply123","cid":"bafyreinewcid"}`), nil
	})
	router, repo := newPostRouterWithBsky(t, succeedReplier)
	seedProject(t, repo, "pr5")
	seedBlueskyPost(t, repo, "pr5", "bsky5")
	t.Setenv("BLUESKY_HANDLE", "handle.bsky.social")
	t.Setenv("BLUESKY_PASSWORD", "app-password")

	rr := doRequest(t, router, http.MethodPost, "/api/projects/pr5/posts/bsky5/post-reply",
		map[string]any{"text": "great post!"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rr.Code, rr.Body.String())
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["status"] != "commented" {
		t.Fatalf("status = %q, want 'commented'", result["status"])
	}
}
