package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountScheduleRoutes registers all schedule routes onto the provided router.
func MountScheduleRoutes(r chi.Router, svc *services.ScheduleService) {
	r.Get("/api/projects/{id}/schedules", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		schedules, status, msg := svc.List(r.Context(), projectID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, schedules)
	})

	r.Post("/api/projects/{id}/schedules", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		var body services.ScheduleRequest
		_ = httpx.DecodeJSON(r, &body)
		sch, status, msg := svc.Create(r.Context(), projectID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, sch)
	})

	r.Patch("/api/projects/{id}/schedules/{sid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		sidStr := chi.URLParam(r, "sid")
		sid, err := strconv.ParseInt(sidStr, 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "Schedule not found")
			return
		}
		var body map[string]any
		_ = httpx.DecodeJSON(r, &body)
		if body == nil {
			body = map[string]any{}
		}
		sch, status, msg := svc.Patch(r.Context(), projectID, sid, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, status, sch)
	})

	r.Delete("/api/projects/{id}/schedules/{sid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		sidStr := chi.URLParam(r, "sid")
		sid, err := strconv.ParseInt(sidStr, 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "Schedule not found")
			return
		}
		status, msg := svc.Delete(r.Context(), projectID, sid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
