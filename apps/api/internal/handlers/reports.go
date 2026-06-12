package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	"github.com/kyle/scout/open-core/apps/api/internal/telemetry"
)

// MountReportRoutes registers all report routes onto the provided router.
func MountReportRoutes(r chi.Router, svc *services.ReportService, rec *telemetry.Recorder) {
	// GET /api/projects/{id}/reports
	r.Get("/api/projects/{id}/reports", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		reports, status, msg := svc.ListReports(r.Context(), projectID)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, reports)
	})

	// GET /api/projects/{id}/reports/{rid}/council
	r.Get("/api/projects/{id}/reports/{rid}/council", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		rid, err := strconv.ParseInt(chi.URLParam(r, "rid"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid report id")
			return
		}
		council, status, msg := svc.GetReportCouncil(r.Context(), projectID, rid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		// council is already JSON; write it directly.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(council)
	})

	// GET /api/projects/{id}/reports/{rid}
	r.Get("/api/projects/{id}/reports/{rid}", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		rid, err := strconv.ParseInt(chi.URLParam(r, "rid"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid report id")
			return
		}
		rep, status, msg := svc.GetReport(r.Context(), projectID, rid)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, rep)
	})

	// POST /api/projects/{id}/reports
		r.Post("/api/projects/{id}/reports", func(w http.ResponseWriter, r *http.Request) {
		projectID := chi.URLParam(r, "id")
		var body services.CreateReportRequest
		_ = httpx.DecodeJSON(r, &body)
		rep, status, msg := svc.StartReport(r.Context(), projectID, body)
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		if rec != nil {
			rec.Record(telemetry.EventFeatureReportCreate)
		}
		httpx.WriteJSON(w, status, rep)
	})
}
