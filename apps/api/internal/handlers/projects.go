package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountProjectRoutes registers all project routes onto the provided router.
func MountProjectRoutes(r chi.Router, svc *services.ProjectService) {
	r.Get("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		projects, err := svc.List(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, projects)
	})

	r.Post("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		var body services.CreateProjectRequest
		// Decode into zero-value struct; ignore EOF (empty body is valid zero-value)
		_ = httpx.DecodeJSON(r, &body)
		project, status, msg := svc.Create(r.Context(), body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, project)
	})

	r.Get("/api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		project, status, msg := svc.Get(r.Context(), id)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, project)
	})

	r.Patch("/api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var body map[string]any
		_ = httpx.DecodeJSON(r, &body)
		if body == nil {
			body = map[string]any{}
		}
		project, status, msg := svc.Patch(r.Context(), id, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, project)
	})

	r.Delete("/api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		status, msg := svc.Delete(r.Context(), id)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	r.Post("/api/projects/{id}/clone", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var body services.CloneProjectRequest
		_ = httpx.DecodeJSON(r, &body)
		project, status, msg := svc.Clone(r.Context(), id, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, project)
	})

	r.Post("/api/projects/{id}/select-angle", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var body services.SelectAngleRequest
		_ = httpx.DecodeJSON(r, &body)
		project, status, msg := svc.SelectAngle(r.Context(), id, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, project)
	})

	r.Post("/api/projects/{id}/graduate", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		project, status, msg := svc.Graduate(r.Context(), id)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, project)
	})
}
