package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountInsightsRoutes registers the insights routes onto the provided router.
func MountInsightsRoutes(r chi.Router, svc *services.InsightsService) {
	// GET /api/insights
	r.Get("/api/insights", func(w http.ResponseWriter, r *http.Request) {
		q := extractQueryParams(r)

		f := services.InsightsFilter{
			ProjectID: q["project_id"],
			Since:     q["since"],
		}

		if minScoreStr, ok := q["min_score"]; ok && minScoreStr != "" {
			score, err := strconv.ParseFloat(minScoreStr, 64)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "Invalid min_score")
				return
			}
			f.MinScore = &score
		}

		result, err := svc.BuildInsights(r.Context(), f)
		if err != nil {
			if services.IsInvalidSince(err) {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, result)
	})
}
