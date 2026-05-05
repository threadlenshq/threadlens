package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountGoogleRoutes registers all google report routes onto the provided router.
func MountGoogleRoutes(r chi.Router, svc *services.GoogleService) {
	// GET /api/projects/{id}/google/reports
	r.Get("/api/projects/{id}/google/reports", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		reports, status, msg := svc.ListGoogleReports(req.Context(), projectID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, reports)
	})

	// GET /api/projects/{id}/google/reports/latest
	r.Get("/api/projects/{id}/google/reports/latest", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		rep, status, msg := svc.LatestGoogleReport(req.Context(), projectID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, rep)
	})

	// GET /api/projects/{id}/google/reports/{rid}/keywords
	r.Get("/api/projects/{id}/google/reports/{rid}/keywords", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		rid, err := strconv.ParseInt(chi.URLParam(req, "rid"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid report id")
			return
		}
		// Resolve report to get run_id
		rep, status, msg := svc.GetGoogleReport(req.Context(), projectID, rid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		summaries, status, msg := svc.ListGoogleKeywordSummaries(req.Context(), projectID, rep.RunID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, summaries)
	})

	// GET /api/projects/{id}/google/reports/{rid}/results
	r.Get("/api/projects/{id}/google/reports/{rid}/results", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		rid, err := strconv.ParseInt(chi.URLParam(req, "rid"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid report id")
			return
		}
		mode := req.URL.Query().Get("mode")
		limitStr := req.URL.Query().Get("limit")
		limit, _ := strconv.Atoi(limitStr)

		result, status, msg := svc.GetGoogleResults(req.Context(), projectID, rid, mode, limit)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, result)
	})

	// GET /api/projects/{id}/google/reports/{rid}
	r.Get("/api/projects/{id}/google/reports/{rid}", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		rid, err := strconv.ParseInt(chi.URLParam(req, "rid"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid report id")
			return
		}
		rep, status, msg := svc.GetGoogleReport(req.Context(), projectID, rid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, rep)
	})
}
