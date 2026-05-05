package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountPromptRoutes registers all prompt routes onto the provided router.
func MountPromptRoutes(r chi.Router, svc *services.PromptService) {
	r.Get("/api/projects/{id}/prompts", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		prompts, status, msg := svc.List(r.Context(), projectID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, prompts)
	})

	r.Post("/api/projects/{id}/prompts", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		var body services.PromptRequest
		_ = httpx.DecodeJSON(r, &body)
		prompt, status, msg := svc.Create(r.Context(), projectID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, prompt)
	})

	r.Patch("/api/projects/{id}/prompts/{pid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		pidStr := chi.URLParam(r, "pid")
		pid, err := strconv.ParseInt(pidStr, 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "Prompt not found")
			return
		}
		var body map[string]any
		_ = httpx.DecodeJSON(r, &body)
		if body == nil {
			body = map[string]any{}
		}
		prompt, status, msg := svc.Patch(r.Context(), projectID, pid, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, prompt)
	})

	r.Delete("/api/projects/{id}/prompts/{pid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		pidStr := chi.URLParam(r, "pid")
		pid, err := strconv.ParseInt(pidStr, 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "Prompt not found")
			return
		}
		status, msg := svc.Delete(r.Context(), projectID, pid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
