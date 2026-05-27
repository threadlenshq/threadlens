package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

type queryReviewJobCreateRequest struct {
	Kind           string `json:"kind"`
	Refinement     string `json:"refinement"`
	SocialReportID int64  `json:"social_report_id"`
	GoogleReportID int64  `json:"google_report_id"`
}

type queryReviewJobReviewedRequest struct {
	Resolution string `json:"resolution"`
}

func MountQueryReviewJobRoutes(r chi.Router, repo *repository.Repository, querySvc *services.QueryService) {
	r.Post("/api/projects/{id}/query-review-jobs", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		var body queryReviewJobCreateRequest
		_ = httpx.DecodeJSON(req, &body)
		body.Kind = strings.TrimSpace(body.Kind)
		body.Refinement = strings.TrimSpace(body.Refinement)

		if body.Kind != string(domain.QueryReviewKindSuggest) && body.Kind != string(domain.QueryReviewKindRefine) {
			httpx.WriteError(w, http.StatusBadRequest, "kind must be suggest or refine")
			return
		}
		if _, err := repo.GetProject(req.Context(), projectID); err != nil {
			writeRepoError(w, err)
			return
		}

		step := "Generating query suggestions"
		if body.Kind == string(domain.QueryReviewKindRefine) {
			step = "Reviewing current queries"
		}
		job, err := repo.CreateQueryReviewJob(req.Context(), projectID, domain.QueryReviewKind(body.Kind), step, body.Refinement)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		go runQueryReviewJob(context.Background(), repo, querySvc, job.ID, projectID, body)
		httpx.WriteJSON(w, http.StatusAccepted, job)
	})

	r.Get("/api/projects/{id}/query-review-jobs", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		if _, err := repo.GetProject(req.Context(), projectID); err != nil {
			writeRepoError(w, err)
			return
		}
		jobs, err := repo.ListQueryReviewJobs(req.Context(), projectID, 20)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, jobs)
	})

	r.Get("/api/projects/{id}/query-review-jobs/{jobId}", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		jobID, ok := parseJobID(w, req)
		if !ok {
			return
		}
		job, err := repo.GetQueryReviewJob(req.Context(), projectID, jobID)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, job)
	})

	r.Post("/api/projects/{id}/query-review-jobs/{jobId}/reviewed", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		jobID, ok := parseJobID(w, req)
		if !ok {
			return
		}
		var body queryReviewJobReviewedRequest
		_ = httpx.DecodeJSON(req, &body)
		if body.Resolution != string(domain.QueryReviewResolutionApplied) && body.Resolution != string(domain.QueryReviewResolutionDenied) {
			httpx.WriteError(w, http.StatusBadRequest, "resolution must be applied or denied")
			return
		}
		job, err := repo.MarkQueryReviewJobReviewed(req.Context(), projectID, jobID, body.Resolution)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, job)
	})
}

func runQueryReviewJob(parent context.Context, repo *repository.Repository, querySvc *services.QueryService, jobID int64, projectID string, body queryReviewJobCreateRequest) {
	timeout := 10 * time.Minute
	if body.Kind == string(domain.QueryReviewKindRefine) {
		timeout = 15 * time.Minute
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	if body.Kind == string(domain.QueryReviewKindSuggest) {
		resp, status, msg := querySvc.Suggest(ctx, projectID, services.SuggestRequest{Refinement: body.Refinement})
		if msg != "" || status >= 400 {
			if msg == "" {
				msg = "Failed to generate suggestions, try again"
			}
			_, _ = repo.FailQueryReviewJob(context.Background(), projectID, jobID, msg)
			return
		}
		_, _ = repo.CompleteQueryReviewJob(context.Background(), projectID, jobID, resp)
		return
	}

	resp, status, msg := querySvc.Refine(ctx, projectID, services.RefineRequest{
		Refinement:     body.Refinement,
		SocialReportID: body.SocialReportID,
		GoogleReportID: body.GoogleReportID,
	})
	if msg != "" || status >= 400 {
		if msg == "" {
			msg = "Failed to generate refinement suggestions, try again"
		}
		_, _ = repo.FailQueryReviewJob(context.Background(), projectID, jobID, msg)
		return
	}
	_, _ = repo.CompleteQueryReviewJob(context.Background(), projectID, jobID, resp)
}

func parseJobID(w http.ResponseWriter, req *http.Request) (int64, bool) {
	jobID, err := strconv.ParseInt(chi.URLParam(req, "jobId"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Query review job not found")
		return 0, false
	}
	return jobID, true
}

func writeRepoError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		message := err.Error()
		if strings.Contains(message, "Project not found") {
			httpx.WriteError(w, http.StatusNotFound, "Project not found")
			return
		}
		httpx.WriteError(w, http.StatusNotFound, "Query review job not found")
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
}
