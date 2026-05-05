package handlers

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountPostRoutes registers all post routes onto the provided router.
func MountPostRoutes(r chi.Router, svc *services.PostService) {
	// GET /api/projects/{id}/posts
	r.Get("/api/projects/{id}/posts", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		q := extractQueryParams(r)
		filters := services.ParseFilters(q)

		pageStr := q["page"]
		limitStr := q["limit"]
		hasPagination := pageStr != "" || limitStr != ""

		if !hasPagination {
			posts, status, msg := svc.ListPosts(r.Context(), projectID, filters)
			if msg != "" {
				httpx.WriteError(w, status, msg)
				return
			}
			httpx.WriteJSON(w, status, posts)
			return
		}

		page := httpx.ParsePositiveInt(pageStr, 1, 0)
		limit := httpx.ParsePositiveInt(limitStr, 20, 100)
		paged, status, msg := svc.ListPostsPage(r.Context(), projectID, filters, page, limit)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, paged)
	})

	// PATCH /api/projects/{id}/posts/bulk - must be before /{pid}
	r.Patch("/api/projects/{id}/posts/bulk", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		var body services.BulkPatchBody
		// decode; on failure body stays zero-value which will fail validation
		_ = httpx.DecodeJSON(r, &body)
		updated, status, msg := svc.BulkPatch(r.Context(), projectID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, map[string]any{"updated": updated})
	})

	// GET /api/projects/{id}/posts/{pid}
	r.Get("/api/projects/{id}/posts/{pid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		postID := unescapePID(r)
		post, status, msg := svc.GetPost(r.Context(), projectID, postID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, post)
	})

	// PATCH /api/projects/{id}/posts/{pid}
	r.Patch("/api/projects/{id}/posts/{pid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		postID := unescapePID(r)
		var body services.PatchPostBody
		_ = httpx.DecodeJSON(r, &body)
		post, status, msg := svc.PatchPost(r.Context(), projectID, postID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, post)
	})

	// PATCH /api/projects/{id}/posts/{pid}/dm/{username}
	r.Patch("/api/projects/{id}/posts/{pid}/dm/{username}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		postID := unescapePID(r)
		username := chi.URLParam(r, "username")
		var body services.PatchDMBody
		_ = httpx.DecodeJSON(r, &body)
		target, status, msg := svc.PatchDMTarget(r.Context(), projectID, postID, username, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, target)
	})

	// POST /api/projects/{id}/posts/{pid}/post-reply
	r.Post("/api/projects/{id}/posts/{pid}/post-reply", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		postID := unescapePID(r)
		var body services.PostReplyBody
		_ = httpx.DecodeJSON(r, &body)
		post, status, msg := svc.PostReply(r.Context(), projectID, postID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, post)
	})

	// POST /api/projects/{id}/posts/{pid}/generate-draft
	r.Post("/api/projects/{id}/posts/{pid}/generate-draft", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		postID := unescapePID(r)
		post, status, msg := svc.GenerateDraft(r.Context(), projectID, postID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, post)
	})

	// POST /api/projects/{id}/posts/{pid}/dm/{username}/generate-draft
	r.Post("/api/projects/{id}/posts/{pid}/dm/{username}/generate-draft", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		postID := unescapePID(r)
		username := chi.URLParam(r, "username")
		target, status, msg := svc.GenerateDMDraft(r.Context(), projectID, postID, username)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, target)
	})
}

// unescapePID applies url.PathUnescape to the {pid} param, matching Express decodeURIComponent.
func unescapePID(r *http.Request) string {
	raw := chi.URLParam(r, "pid")
	unescaped, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return unescaped
}

// extractQueryParams builds a simple string map of query params for filter parsing.
func extractQueryParams(r *http.Request) map[string]string {
	m := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}
