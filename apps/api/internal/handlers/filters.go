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
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/telemetry"
)

type filterRecoverRequest struct {
	FindingType string                   `json:"finding_type"`
	ID          string                   `json:"id"`
	Mode        string                   `json:"mode"`
	Trust       *domain.FilterTrustOption `json:"trust,omitempty"`
}

type filterTrustCreateRequest struct {
	Platform   string `json:"platform"`
	TrustType  string `json:"trust_type"`
	SourceKind string `json:"source_kind"`
	SourceKey  string `json:"source_key"`
	Reason     string `json:"reason"`
}

type filterJobCreateRequest struct {
	RequestedScope string                 `json:"requested_scope"`
	Targets        []domain.FilterJobTarget `json:"targets"`
}

// MountFilterRoutes registers owner-facing filter endpoints on the given router.
func MountFilterRoutes(r chi.Router, repo *repository.Repository, classifier *pipeline.FilterClassifier, rec *telemetry.Recorder) {

	// GET /api/projects/{id}/filters/findings
	r.Get("/api/projects/{id}/filters/findings", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		if _, err := repo.GetProject(req.Context(), projectID); err != nil {
			writeRepoError(w, err)
			return
		}

		q := req.URL.Query()
		filters := repository.FilteredFindingFilters{
			Platform: q.Get("platform"),
			Reason:   q.Get("reason"),
			Source:   q.Get("source"),
		}
		if v := q.Get("ai_used"); v != "" {
			b := v == "true"
			filters.AIUsed = &b
		}
		if v := q.Get("min_confidence"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				filters.MinConfidence = &f
			}
		}
		if v := q.Get("max_confidence"); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				filters.MaxConfidence = &f
			}
		}

		page := 1
		limit := 20
		if v := q.Get("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		}
		if v := q.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		findings, err := repo.ListFilteredFindings(req.Context(), projectID, filters, page, limit)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, findings)
	})

	// POST /api/projects/{id}/filters/findings/recover
	r.Post("/api/projects/{id}/filters/findings/recover", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		if _, err := repo.GetProject(req.Context(), projectID); err != nil {
			writeRepoError(w, err)
			return
		}

		var body filterRecoverRequest
		if err := httpx.DecodeJSON(req, &body); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		body.FindingType = strings.TrimSpace(body.FindingType)
		body.ID = strings.TrimSpace(body.ID)
		body.Mode = strings.TrimSpace(body.Mode)

		if body.FindingType != domain.FindingTypePost && body.FindingType != domain.FindingTypeGoogleResult {
			httpx.WriteError(w, http.StatusBadRequest, "invalid finding_type: must be post or google_result")
			return
		}
		if body.Mode != "restore_visibility" && body.Mode != "restore_and_trust" {
			httpx.WriteError(w, http.StatusBadRequest, "invalid mode: must be restore_visibility or restore_and_trust")
			return
		}
		if body.Mode == "restore_and_trust" {
			if body.Trust == nil {
				httpx.WriteError(w, http.StatusBadRequest, "trust is required for restore_and_trust mode")
				return
			}
			if body.Trust.TrustType != domain.TrustTypeSource && body.Trust.TrustType != domain.TrustTypeFilterSignature {
				httpx.WriteError(w, http.StatusBadRequest, "invalid trust_type")
				return
			}
			if strings.TrimSpace(body.Trust.SourceKind) == "" || strings.TrimSpace(body.Trust.SourceKey) == "" {
				httpx.WriteError(w, http.StatusBadRequest, "trust source_kind and source_key are required")
				return
			}
		}

		// Verify the finding belongs to this project before mutating
		if err := verifyFindingBelongsToProject(req.Context(), repo, projectID, body.FindingType, body.ID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				httpx.WriteError(w, http.StatusNotFound, "Finding not found")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		var createdTrust *domain.TrustRecord
		if body.Mode == "restore_and_trust" {
			tr, err := repo.CreateTrustRecord(req.Context(), domain.TrustRecord{
				ProjectID:  projectID,
				Platform:   body.Trust.Platform,
				TrustType:  body.Trust.TrustType,
				SourceKind: body.Trust.SourceKind,
				SourceKey:  body.Trust.SourceKey,
				Reason:     "owner recovery",
			})
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
			createdTrust = &tr
		}

		if err := repo.RestoreFindingVisibility(req.Context(), projectID, body.FindingType, body.ID, "owner restored"); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"finding_type": body.FindingType,
			"id":           body.ID,
			"filter_state": domain.FilterStateVisible,
			"trust_record": createdTrust,
		})
	})

	// GET /api/projects/{id}/filters/trust-records
	r.Get("/api/projects/{id}/filters/trust-records", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		if _, err := repo.GetProject(req.Context(), projectID); err != nil {
			writeRepoError(w, err)
			return
		}
		records, err := repo.ListTrustRecords(req.Context(), projectID)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, records)
	})

	// POST /api/projects/{id}/filters/trust-records
	r.Post("/api/projects/{id}/filters/trust-records", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		if _, err := repo.GetProject(req.Context(), projectID); err != nil {
			writeRepoError(w, err)
			return
		}

		var body filterTrustCreateRequest
		if err := httpx.DecodeJSON(req, &body); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if body.TrustType != domain.TrustTypeSource && body.TrustType != domain.TrustTypeFilterSignature {
			httpx.WriteError(w, http.StatusBadRequest, "invalid trust_type")
			return
		}
		if strings.TrimSpace(body.SourceKind) == "" || strings.TrimSpace(body.SourceKey) == "" {
			httpx.WriteError(w, http.StatusBadRequest, "source_kind and source_key are required")
			return
		}

		rec, err := repo.CreateTrustRecord(req.Context(), domain.TrustRecord{
			ProjectID:  projectID,
			Platform:   body.Platform,
			TrustType:  body.TrustType,
			SourceKind: body.SourceKind,
			SourceKey:  body.SourceKey,
			Reason:     body.Reason,
		})
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, rec)
	})

	// POST /api/projects/{id}/filters/jobs
	r.Post("/api/projects/{id}/filters/jobs", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		if _, err := repo.GetProject(req.Context(), projectID); err != nil {
			writeRepoError(w, err)
			return
		}

		var body filterJobCreateRequest
		if err := httpx.DecodeJSON(req, &body); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		validScopes := map[string]bool{
			domain.FilterJobScopeSelectedVisiblePosts: true,
			domain.FilterJobScopeSelectedFiltered:     true,
			domain.FilterJobScopeSelectedGoogle:       true,
		}
		if !validScopes[body.RequestedScope] {
			httpx.WriteError(w, http.StatusBadRequest, "invalid requested_scope")
			return
		}
		if len(body.Targets) == 0 {
			httpx.WriteError(w, http.StatusBadRequest, "targets must not be empty")
			return
		}

		job, err := repo.CreateFilterJob(req.Context(), projectID, body.RequestedScope, body.Targets)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		go runFilterJob(context.Background(), repo, classifier, projectID, job.ID, body)
		if rec != nil {
			rec.Record(telemetry.EventFeatureFilterJob)
		}
		httpx.WriteJSON(w, http.StatusAccepted, job)
	})

	// GET /api/projects/{id}/filters/jobs
	r.Get("/api/projects/{id}/filters/jobs", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		if _, err := repo.GetProject(req.Context(), projectID); err != nil {
			writeRepoError(w, err)
			return
		}
		jobs, err := repo.ListFilterJobs(req.Context(), projectID, 20)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, jobs)
	})

	// GET /api/projects/{id}/filters/jobs/{jobId}
	r.Get("/api/projects/{id}/filters/jobs/{jobId}", func(w http.ResponseWriter, req *http.Request) {
		projectID := chi.URLParam(req, "id")
		jobID, err := strconv.ParseInt(chi.URLParam(req, "jobId"), 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusNotFound, "Filter job not found")
			return
		}
		if _, err := repo.GetProject(req.Context(), projectID); err != nil {
			writeRepoError(w, err)
			return
		}
		job, err := repo.GetFilterJob(req.Context(), projectID, jobID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				httpx.WriteError(w, http.StatusNotFound, "Filter job not found")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, job)
	})
}

// runFilterJob runs a background filter classification job.
// It never updates post status, post_score, final_score, drafts, report rows, or Google relevance fields.
func runFilterJob(parent context.Context, repo *repository.Repository, classifier *pipeline.FilterClassifier, projectID string, jobID int64, body filterJobCreateRequest) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()

	_ = repo.UpdateFilterJobStep(ctx, projectID, jobID, "Classifying selected findings")

	var filtered, restoredByTrust, unchanged, failed int64
	errs := map[string]string{}

	for _, target := range body.Targets {
		target.FindingType = strings.TrimSpace(target.FindingType)
		target.ID = strings.TrimSpace(target.ID)

		if target.FindingType != domain.FindingTypePost && target.FindingType != domain.FindingTypeGoogleResult {
			failed++
			errs[target.ID] = "unknown finding_type"
			continue
		}

		var input pipeline.FilterInput

		if target.FindingType == domain.FindingTypePost {
			post, err := repo.GetPost(ctx, projectID, target.ID)
			if err != nil {
				failed++
				errs[target.ID] = err.Error()
				continue
			}
			// Normalize to FilterInput without recalculating scores
			input = pipeline.FilterInput{
				FindingType:    domain.FindingTypePost,
				Platform:       post.Platform,
				ID:             post.ID,
				Title:          post.Title,
				Body:           post.Body,
				URL:            post.URL,
				Author:         post.Author,
				SourceIdentity: post.SourceIdentity,
			}
		} else {
			// google_result targets are not currently loadable by string ID in one call;
			// skip them gracefully since no GetGoogleResult(ctx, projectID, stringID) helper exists
			failed++
			errs[target.ID] = "google_result targets not supported in manual job runner"
			continue
		}

		decision, err := classifier.Classify(ctx, projectID, input)
		if err != nil {
			failed++
			errs[target.ID] = err.Error()
			continue
		}

		// Apply only filter metadata; never touch post_score, final_score, status, drafts, or reports
		if target.FindingType == domain.FindingTypePost {
			if err := repo.ApplyPostFilterDecision(ctx, projectID, target.ID, decision, &jobID); err != nil {
				failed++
				errs[target.ID] = err.Error()
				continue
			}
		}

		switch decision.State {
		case domain.FilterStateFiltered:
			filtered++
		case domain.FilterStateVisible:
			if decision.Source == domain.FilterSourceTrustedOverride {
				restoredByTrust++
			} else {
				unchanged++
			}
		default:
			unchanged++
		}
	}

	result := domain.FilterJobResult{
		Filtered:        filtered,
		RestoredByTrust: restoredByTrust,
		Unchanged:       unchanged,
		Failed:          failed,
		Errors:          errs,
	}
	_, _ = repo.CompleteFilterJob(context.Background(), projectID, jobID, result)
}

// verifyFindingBelongsToProject checks that the finding ID belongs to the given project.
func verifyFindingBelongsToProject(ctx context.Context, repo *repository.Repository, projectID, findingType, id string) error {
	if findingType == domain.FindingTypePost {
		_, err := repo.GetPost(ctx, projectID, id)
		return err
	}
	// google_result IDs are integers stored as TEXT via CAST; verify ownership directly in DB
	var count int
	err := repo.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM google_results WHERE CAST(id AS TEXT) = ? AND project_id = ?`,
		id, projectID,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return repository.ErrNotFound
	}
	return nil
}
