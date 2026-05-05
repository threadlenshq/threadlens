package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountScoutRoutes registers all scout run routes onto the provided router.
func MountScoutRoutes(r chi.Router, svc *services.ScoutService) {
	// POST /api/projects/{id}/scout
	r.Post("/api/projects/{id}/scout", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")

		// Accept optional platform in query string (matching Express behavior).
		platform := r.URL.Query().Get("platform")
		if platform == "" {
			// Fall back to JSON body.
			var body struct {
				Platform string `json:"platform"`
			}
			_ = httpx.DecodeJSON(r, &body)
			platform = body.Platform
		}

		run, status, msg := svc.StartRun(r.Context(), projectID, platform)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, map[string]any{
			"runId":  run.ID,
			"status": run.Status,
		})
	})

	// GET /api/projects/{id}/scout/runs
	r.Get("/api/projects/{id}/scout/runs", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		runs, status, msg := svc.ListRuns(r.Context(), projectID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, runs)
	})

	// GET /api/projects/{id}/scout/runs/{rid}
	r.Get("/api/projects/{id}/scout/runs/{rid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		rid, err := strconv.ParseInt(chi.URLParam(r, "rid"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid run id")
			return
		}
		run, status, msg := svc.GetRun(r.Context(), projectID, rid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, run)
	})

	// POST /api/projects/{id}/scout/runs/{rid}/cancel
	r.Post("/api/projects/{id}/scout/runs/{rid}/cancel", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		rid, err := strconv.ParseInt(chi.URLParam(r, "rid"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid run id")
			return
		}
		status, msg := svc.CancelRun(r.Context(), projectID, rid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
}
