package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountGeneralQueryRoutes registers general-query routes onto the provided router.
func MountGeneralQueryRoutes(r chi.Router, svc *services.GeneralQueryService) {
	r.Get("/api/projects/{id}/general-queries", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		gqs, status, msg := svc.List(r.Context(), projectID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, gqs)
	})

	r.Post("/api/projects/{id}/general-queries", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		var body services.GeneralQueryRequest
		_ = httpx.DecodeJSON(r, &body)
		gq, status, msg := svc.Create(r.Context(), projectID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, gq)
	})

	r.Patch("/api/projects/{id}/general-queries/{gqid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		gqid, err := strconv.ParseInt(chi.URLParam(r, "gqid"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "General query not found")
			return
		}
		var body services.GeneralQueryRequest
		_ = httpx.DecodeJSON(r, &body)
		gq, status, msg := svc.Update(r.Context(), projectID, gqid, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, gq)
	})

	r.Delete("/api/projects/{id}/general-queries/{gqid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		gqid, err := strconv.ParseInt(chi.URLParam(r, "gqid"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "General query not found")
			return
		}
		status, msg := svc.Delete(r.Context(), projectID, gqid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
