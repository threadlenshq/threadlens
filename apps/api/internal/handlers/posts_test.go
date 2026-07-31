package handlers_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	dbpkg "github.com/kyle/scout/open-core/apps/api/internal/db"
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

// seedDMTarget inserts a dm_target row. The status argument defaults to "new"
// when empty so existing callers keep working. The dm_status_updated_at column
// is always set to datetime('now') on insert (column default), and can be
// overridden by passing non-zero overrideUpdatedAt.
func seedDMTarget(t *testing.T, repo *repository.Repository, postID, username string, intentScore float64, status string, overrideUpdatedAt string) int64 {
	t.Helper()
	if status == "" {
		status = "new"
	}
	updatedClause := "datetime('now')"
	if overrideUpdatedAt != "" {
		updatedClause = "?"
	}
	stmt := "INSERT INTO dm_targets (post_id, username, intent_score, signal, context, approach, dm_status, dm_status_updated_at) VALUES (?, ?, ?, 'signal', 'ctx', 'approach', ?, " + updatedClause + ")"
	var res sql.Result
	var err error
	if overrideUpdatedAt != "" {
		res, err = repo.DB.Exec(stmt, postID, username, intentScore, status, overrideUpdatedAt)
	} else {
		res, err = repo.DB.Exec(stmt, postID, username, intentScore, status)
	}
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
	seedDMTarget(t, repo, "post3", "user_high", 9.0, "", "")
	seedDMTarget(t, repo, "post3", "user_low", 3.0, "", "")

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
	seedDMTarget(t, repo, "post10", "targetuser", 7.5, "", "")

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
	if result["dm_status_updated_at"] == nil || result["dm_status_updated_at"] == "" {
		t.Fatalf("dm_status_updated_at must be set on response, got %v", result["dm_status_updated_at"])
	}
}

func TestPatchDMTarget_AcceptsAllFourStatuses(t *testing.T) {
	for _, status := range []string{"new", "sent", "replied", "ignored"} {
		t.Run(status, func(t *testing.T) {
			router, repo := newPostRouter(t)
			seedProject(t, repo, "ps-"+status)
			seedPost(t, repo, "ps-"+status, "post-"+status)
			seedDMTarget(t, repo, "post-"+status, "user-"+status, 5.0, "", "")

			rr := doRequest(t, router, http.MethodPatch, "/api/projects/ps-"+status+"/posts/post-"+status+"/dm/user-"+status,
				map[string]any{"dm_status": status})
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200\nbody: %s", rr.Code, rr.Body.String())
			}
			var result map[string]any
			json.Unmarshal(rr.Body.Bytes(), &result)
			if result["dm_status"] != status {
				t.Fatalf("dm_status = %q, want %q", result["dm_status"], status)
			}
		})
	}
}

func TestPatchDMTarget_RejectsInvalidStatus(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pbad")
	seedPost(t, repo, "pbad", "postbad")
	seedDMTarget(t, repo, "postbad", "userbad", 5.0, "", "")

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/pbad/posts/postbad/dm/userbad",
		map[string]any{"dm_status": "bogus"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400\nbody: %s", rr.Code, rr.Body.String())
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	want := "Invalid dm_status. Must be one of: new, sent, replied, ignored"
	if result["error"] != want {
		t.Fatalf("error = %q, want %q", result["error"], want)
	}
}

func TestPatchDMTarget_DraftOnlyPatch_DoesNotAdvanceStatusTimestamp(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pt")
	seedPost(t, repo, "pt", "postT")
	// Seed a target with a known fixed timestamp so we can assert it doesn't change.
	seedDMTarget(t, repo, "postT", "userT", 5.0, "new", "2020-01-01 00:00:00")

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/pt/posts/postT/dm/userT",
		map[string]any{"draft_dm": "new draft text"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rr.Code, rr.Body.String())
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if got := result["dm_status_updated_at"]; !strings.HasPrefix(got.(string), "2020-01-01") {
		t.Fatalf("draft-only patch must not advance dm_status_updated_at, got %v", got)
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
	if err != nil {
		t.Fatal(err)
	}

	rr := doRequest(t, router, http.MethodGet, "/api/projects/pf1/posts?status=new", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var result []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0]["id"] != "visible-post" {
		t.Fatalf("result = %#v", result)
	}
}

func TestPostGet_FilteredPost_IsStillReachableByID(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pf2")
	seedPost(t, repo, "pf2", "filtered-id-post")
	_, err := repo.DB.Exec(`UPDATE posts SET filter_state = 'filtered', filter_reason = 'spam', filter_reasons_json = '["spam"]', filter_explanation = 'test', filter_source = 'rules', filtered_at = datetime('now') WHERE id = 'filtered-id-post'`)
	if err != nil {
		t.Fatal(err)
	}

	rr := doRequest(t, router, http.MethodGet, "/api/projects/pf2/posts/filtered-id-post", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["filter_state"] != "filtered" {
		t.Fatalf("filter_state = %v", result["filter_state"])
	}
}

func TestPostList_DMFilter_ReturnsOnlyPostsWithTargets(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pdm1")

	// post with a dm_target
	seedPost(t, repo, "pdm1", "post-with-target")
	seedDMTarget(t, repo, "post-with-target", "someuser", 8.0, "", "")
	_, err := repo.DB.Exec(`UPDATE posts SET filter_state = 'visible' WHERE id = 'post-with-target'`)
	if err != nil {
		t.Fatal(err)
	}

	// post without any dm_target
	seedPost(t, repo, "pdm1", "post-no-target")

	// filtered post with a target should still be excluded by the visible clause
	seedPost(t, repo, "pdm1", "post-filtered-target")
	seedDMTarget(t, repo, "post-filtered-target", "hiddenuser", 9.0, "", "")
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

func TestPostList_DMFilter_ReturnsOnlyVisiblePostsWithTargets(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "dmfilter")
	seedPost(t, repo, "dmfilter", "with-target")
	seedPost(t, repo, "dmfilter", "without-target")
	seedPost(t, repo, "dmfilter", "filtered-with-target")
	seedDMTarget(t, repo, "with-target", "targetuser", 8.5, "", "")
	seedDMTarget(t, repo, "filtered-with-target", "hiddenuser", 9.5, "", "")
	_, err := repo.DB.Exec(`UPDATE posts SET filter_state = 'filtered', filter_reason = 'spam', filter_reasons_json = '["spam"]', filter_explanation = 'test', filter_source = 'rules', filtered_at = datetime('now') WHERE id = 'filtered-with-target'`)
	if err != nil {
		t.Fatalf("mark filtered post: %v", err)
	}

	rr := doRequest(t, router, http.MethodGet, "/api/projects/dmfilter/posts?status=new&dm=true", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}
	var result []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected exactly one visible post with DM targets, got %#v", result)
	}
	if result[0]["id"] != "with-target" {
		t.Fatalf("expected with-target, got %#v", result[0])
	}
	targets, ok := result[0]["dm_targets"].([]any)
	if !ok || len(targets) != 1 {
		t.Fatalf("expected one attached dm target, got %#v", result[0]["dm_targets"])
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

// --- DM Targets list endpoint ---

func seedPostWithTitle(t *testing.T, repo *repository.Repository, projectID, postID, title, subreddit, url string) {
	t.Helper()
	_, err := repo.DB.Exec(
		`INSERT INTO posts (id, project_id, platform, title, body, author, url, subreddit, post_score, final_score, engagement_type, status, found_at, scouted_at)
		 VALUES (?, ?, 'reddit', ?, 'body', 'author', ?, ?, 5.0, 5.0, 'karma', 'new', datetime('now'), datetime('now'))`,
		postID, projectID, title, url, subreddit,
	)
	if err != nil {
		t.Fatalf("seed post with title: %v", err)
	}
}

func TestListDMTargets_NoStatus_ReturnsAllAndCounts(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pl1")
	seedPostWithTitle(t, repo, "pl1", "postA", "Post A", "r/foo", "https://reddit.com/r/foo/comments/a")
	seedPostWithTitle(t, repo, "pl1", "postB", "Post B", "r/bar", "https://reddit.com/r/bar/comments/b")
	seedDMTarget(t, repo, "postA", "alice", 7.0, "", "")
	seedDMTarget(t, repo, "postA", "bob", 8.0, "sent", "")
	seedDMTarget(t, repo, "postB", "carol", 6.0, "replied", "")

	rr := doRequest(t, router, http.MethodGet, "/api/projects/pl1/dm-targets", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rr.Code, rr.Body.String())
	}
	var result struct {
		Items  []map[string]any `json:"items"`
		Counts map[string]int64 `json:"counts"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, rr.Body.String())
	}
	if len(result.Items) != 3 {
		t.Fatalf("items len = %d, want 3", len(result.Items))
	}
	wantCounts := map[string]int64{"new": 1, "sent": 1, "replied": 1, "ignored": 0}
	for k, v := range wantCounts {
		if result.Counts[k] != v {
			t.Fatalf("counts[%q] = %d, want %d (full counts: %#v)", k, result.Counts[k], v, result.Counts)
		}
	}
	// Join fields populated for the first item.
	first := result.Items[0]
	if first["post_title"] == "" || first["post_title"] == nil {
		t.Fatalf("post_title missing on item: %#v", first)
	}
	if first["post_url"] == "" || first["post_url"] == nil {
		t.Fatalf("post_url missing on item: %#v", first)
	}
}

func TestListDMTargets_StatusFilter_AppliesToItemsNotCounts(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pl2")
	seedPostWithTitle(t, repo, "pl2", "p1", "P1", "", "")
	seedPostWithTitle(t, repo, "pl2", "p2", "P2", "", "")
	seedDMTarget(t, repo, "p1", "u1", 5.0, "sent", "")
	seedDMTarget(t, repo, "p2", "u2", 5.0, "new", "")

	rr := doRequest(t, router, http.MethodGet, "/api/projects/pl2/dm-targets?status=sent", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rr.Code, rr.Body.String())
	}
	var result struct {
		Items  []map[string]any `json:"items"`
		Counts map[string]int64 `json:"counts"`
	}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if len(result.Items) != 1 {
		t.Fatalf("items len = %d, want 1 (status=sent)", len(result.Items))
	}
	if result.Items[0]["dm_status"] != "sent" {
		t.Fatalf("filtered item dm_status = %v, want sent", result.Items[0]["dm_status"])
	}
	if result.Counts["new"] != 1 || result.Counts["sent"] != 1 {
		t.Fatalf("counts must reflect ALL targets regardless of status filter, got %#v", result.Counts)
	}
}

func TestListDMTargets_OrderingByUpdatedAtDesc(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pl3")
	seedPostWithTitle(t, repo, "pl3", "p", "P", "", "")
	seedDMTarget(t, repo, "p", "old", 5.0, "new", "2024-01-01 00:00:00")
	seedDMTarget(t, repo, "p", "mid", 5.0, "new", "2025-01-01 00:00:00")
	seedDMTarget(t, repo, "p", "newest", 5.0, "new", "2026-01-01 00:00:00")

	rr := doRequest(t, router, http.MethodGet, "/api/projects/pl3/dm-targets", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result struct {
		Items []map[string]any `json:"items"`
	}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if len(result.Items) != 3 {
		t.Fatalf("items len = %d, want 3", len(result.Items))
	}
	if result.Items[0]["username"] != "newest" {
		t.Fatalf("first item username = %v, want newest", result.Items[0]["username"])
	}
	if result.Items[2]["username"] != "old" {
		t.Fatalf("last item username = %v, want old", result.Items[2]["username"])
	}
}

func TestListDMTargets_InvalidStatusFilter_Returns400(t *testing.T) {
	router, _ := newPostRouter(t)
	rr := doRequest(t, router, http.MethodGet, "/api/projects/x/dm-targets?status=bogus", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	want := "Invalid status filter. Must be one of: new, sent, replied, ignored"
	if result["error"] != want {
		t.Fatalf("error = %q, want %q", result["error"], want)
	}
}

func TestListDMTargets_NonexistentProject_ReturnsEmptyItemsAndZeroCounts(t *testing.T) {
	router, _ := newPostRouter(t)
	rr := doRequest(t, router, http.MethodGet, "/api/projects/nope/dm-targets", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result struct {
		Items  []map[string]any `json:"items"`
		Counts map[string]int64 `json:"counts"`
	}
	json.Unmarshal(rr.Body.Bytes(), &result)
	if len(result.Items) != 0 {
		t.Fatalf("items len = %d, want 0", len(result.Items))
	}
	for _, k := range []string{"new", "sent", "replied", "ignored"} {
		if result.Counts[k] != 0 {
			t.Fatalf("counts[%q] = %d, want 0", k, result.Counts[k])
		}
	}
}

// --- DM Targets bulk endpoint ---

func TestBulkPatchDMTargets_UpdatesAndRefreshesTimestamp(t *testing.T) {
	router, repo := newPostRouter(t)
	seedProject(t, repo, "pb1")
	seedPostWithTitle(t, repo, "pb1", "p", "P", "", "")
	id1 := seedDMTarget(t, repo, "p", "u1", 5.0, "new", "2020-01-01 00:00:00")
	id2 := seedDMTarget(t, repo, "p", "u2", 5.0, "new", "2020-01-01 00:00:00")

	rr := doRequest(t, router, http.MethodPatch, "/api/projects/pb1/dm-targets/bulk",
		map[string]any{"ids": []int64{id1, id2}, "dm_status": "sent"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rr.Code, rr.Body.String())
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["updated"].(float64) != 2 {
		t.Fatalf("updated = %v, want 2", result["updated"])
	}
	// Confirm via direct query that dm_status_updated_at is no longer the
	// 2020 seed value (the bulk UPDATE refreshes it via datetime('now')).
	var got string
	err := repo.DB.QueryRow("SELECT dm_status_updated_at FROM dm_targets WHERE id = ?", id1).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(got, "2020-01-01") {
		t.Fatalf("dm_status_updated_at must be refreshed on bulk update, got %q", got)
	}
}

func TestBulkPatchDMTargets_EmptyIDs_Returns400(t *testing.T) {
	router, _ := newPostRouter(t)
	rr := doRequest(t, router, http.MethodPatch, "/api/projects/x/dm-targets/bulk",
		map[string]any{"ids": []int64{}, "dm_status": "sent"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	want := "ids (array) and dm_status are required"
	if result["error"] != want {
		t.Fatalf("error = %q, want %q", result["error"], want)
	}
}

func TestBulkPatchDMTargets_InvalidStatus_Returns400(t *testing.T) {
	router, _ := newPostRouter(t)
	rr := doRequest(t, router, http.MethodPatch, "/api/projects/x/dm-targets/bulk",
		map[string]any{"ids": []int64{1}, "dm_status": "bogus"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var result map[string]string
	json.Unmarshal(rr.Body.Bytes(), &result)
	want := "Invalid dm_status. Must be one of: new, sent, replied, ignored"
	if result["error"] != want {
		t.Fatalf("error = %q, want %q", result["error"], want)
	}
}

func TestBulkPatchDMTargets_CrossProjectIDIsIgnored(t *testing.T) {
	router, repo := newPostRouter(t)
	// Two projects, one target each.
	seedProject(t, repo, "own1")
	seedProject(t, repo, "own2")
	seedPostWithTitle(t, repo, "own1", "p1", "P1", "", "")
	seedPostWithTitle(t, repo, "own2", "p2", "P2", "", "")
	ownID := seedDMTarget(t, repo, "p1", "ownuser", 5.0, "new", "")
	otherID := seedDMTarget(t, repo, "p2", "otheruser", 5.0, "new", "")

	// PATCH against own1 with both ids — only ownID belongs to own1.
	rr := doRequest(t, router, http.MethodPatch, "/api/projects/own1/dm-targets/bulk",
		map[string]any{"ids": []int64{ownID, otherID}, "dm_status": "sent"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["updated"].(float64) != 1 {
		t.Fatalf("updated = %v, want 1 (cross-project id must be ignored)", result["updated"])
	}
	// The other project's target must still be 'new'.
	var status string
	if err := repo.DB.QueryRow("SELECT dm_status FROM dm_targets WHERE id = ?", otherID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "new" {
		t.Fatalf("cross-project target dm_status = %q, want new (must not be mutated)", status)
	}
}

// --- Migration: rebuild preserves ids and backfills dm_status_updated_at ---

func TestDMTargets_RebuildPreservesRowsAndBackfillsTimestamp(t *testing.T) {
	// Pre-seed an old-shape table manually (two-state CHECK, no
	// dm_status_updated_at column), then run InitSchema to trigger the rebuild.
	db := testingpkg.OpenTestDB(t)
	// Drop the new table InitSchema created, recreate the old shape.
	if _, err := db.Exec(`DROP TABLE dm_targets`); err != nil {
		t.Fatalf("drop dm_targets: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE dm_targets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id TEXT NOT NULL,
			username TEXT NOT NULL,
			intent_score REAL NOT NULL DEFAULT 0,
			signal TEXT NOT NULL DEFAULT '',
			context TEXT NOT NULL DEFAULT '',
			approach TEXT NOT NULL DEFAULT '',
			draft_dm TEXT,
			draft_provider TEXT,
			dm_status TEXT NOT NULL DEFAULT 'new' CHECK (dm_status IN ('new', 'sent')),
			profile_score REAL,
			profile_signals TEXT
		)`); err != nil {
		t.Fatalf("recreate old shape: %v", err)
	}
	// Seed the parent project and post so the foreign key in the rebuilt table is satisfied.
	if _, err := db.Exec(
		`INSERT INTO projects (id, name, mode, created_at, updated_at) VALUES ('px', 'px', 'research', datetime('now'), datetime('now'))`,
	); err != nil {
		t.Fatalf("seed parent project: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO posts (id, project_id, platform, title, body, author, url, post_score, final_score, engagement_type, status, found_at, scouted_at)
		 VALUES ('post-x', 'px', 'reddit', 't', 'b', 'a', 'u', 0, 0, 'karma', 'new', datetime('now'), datetime('now'))`,
	); err != nil {
		t.Fatalf("seed parent post: %v", err)
	}
	// Seed one row with a known id and a known old status.
	_, err := db.Exec(
		`INSERT INTO dm_targets (id, post_id, username, intent_score, signal, context, approach, dm_status) VALUES (42, 'post-x', 'olduser', 6.0, '', '', '', 'sent')`,
	)
	if err != nil {
		t.Fatalf("seed old row: %v", err)
	}
	// Now run InitSchema — the rebuild helper must fire.
	if err := dbpkg.InitSchema(db); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}
	// Assert the row is still there, with the same id, and dm_status_updated_at populated.
	var (
		gotID     int64
		gotStatus string
		gotUpd    string
	)
	if err := db.QueryRow(`SELECT id, dm_status, dm_status_updated_at FROM dm_targets WHERE username = 'olduser'`).Scan(&gotID, &gotStatus, &gotUpd); err != nil {
		t.Fatalf("read rebuilt row: %v", err)
	}
	if gotID != 42 {
		t.Fatalf("id = %d, want 42 (rebuild must preserve primary keys)", gotID)
	}
	if gotStatus != "sent" {
		t.Fatalf("dm_status = %q, want sent (rebuild must preserve status)", gotStatus)
	}
	if gotUpd == "" {
		t.Fatalf("dm_status_updated_at must be backfilled on rebuild, got empty")
	}
	// Assert the new CHECK accepts 'replied' and 'ignored'.
	if _, err := db.Exec(`INSERT INTO projects (id, name, mode, created_at, updated_at) VALUES ('px2', 'px2', 'research', datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("seed check-test project: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO posts (id, project_id, platform, title, body, author, url, post_score, final_score, engagement_type, status, found_at, scouted_at) VALUES ('p', 'px2', 'reddit', 't', 'b', 'a', 'u', 0, 0, 'karma', 'new', datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("seed check-test post: %v", err)
	}
	for _, s := range []string{"replied", "ignored"} {
		if _, err := db.Exec(`INSERT INTO dm_targets (post_id, username, intent_score, signal, context, approach, dm_status) VALUES ('p', 'u-'+?, 0, '', '', '', ?)`, s, s); err != nil {
			t.Fatalf("insert with dm_status=%q must succeed: %v", s, err)
		}
	}
	// And rejects a bogus value (defence-in-depth).
	if _, err := db.Exec(`INSERT INTO dm_targets (post_id, username, intent_score, signal, context, approach, dm_status) VALUES ('p', 'ubogus', 0, '', '', '', 'totally-bogus')`); err == nil {
		t.Fatal("CHECK must reject invalid dm_status after rebuild")
	}
}
