package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountQueryRoutes registers all query routes onto the provided router.
func MountQueryRoutes(r chi.Router, svc *services.QueryService) {
	r.Get("/api/projects/{id}/queries", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		queries, status, msg := svc.List(r.Context(), projectID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, queries)
	})

	r.Post("/api/projects/{id}/queries", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		var body services.QueryRequest
		_ = httpx.DecodeJSON(r, &body)
		query, status, msg := svc.Create(r.Context(), projectID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, query)
	})

	r.Post("/api/projects/{id}/queries/suggest", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		var body services.SuggestRequest
		_ = httpx.DecodeJSON(r, &body)
		resp, status, msg := svc.Suggest(r.Context(), projectID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, resp)
	})

	r.Post("/api/projects/{id}/queries/refine", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		var body services.RefineRequest
		_ = httpx.DecodeJSON(r, &body)
		resp, status, msg := svc.Refine(r.Context(), projectID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, resp)
	})

	r.Patch("/api/projects/{id}/queries/{qid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		qidStr := chi.URLParam(r, "qid")
		qid, err := strconv.ParseInt(qidStr, 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "Query not found")
			return
		}
		var body map[string]any
		_ = httpx.DecodeJSON(r, &body)
		if body == nil {
			body = map[string]any{}
		}
		query, status, msg := svc.Patch(r.Context(), projectID, qid, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, query)
	})

	r.Delete("/api/projects/{id}/queries/{qid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		qidStr := chi.URLParam(r, "qid")
		qid, err := strconv.ParseInt(qidStr, 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "Query not found")
			return
		}
		status, msg := svc.Delete(r.Context(), projectID, qid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
