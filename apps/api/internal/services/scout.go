package services

import (
	"context"
	"net/http"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// ScoutService handles business logic for scout runs.
type ScoutService struct {
	repo   *repository.Repository
	runner *pipeline.Runner
}

// NewScoutService creates a new ScoutService.
func NewScoutService(repo *repository.Repository, runner *pipeline.Runner) *ScoutService {
	return &ScoutService{repo: repo, runner: runner}
}

// StartRun validates inputs, creates a scout_run row, and starts the pipeline asynchronously.
// Returns the new runID and status "running", or an HTTP status code and error message.
func (s *ScoutService) StartRun(ctx context.Context, projectID string, platform string) (domain.ScoutRun, int, string) {
	if !validPlatforms[platform] {
		return domain.ScoutRun{}, http.StatusBadRequest, `platform must be "reddit", "bluesky", or "google"`
	}

	// Verify project exists.
	_, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Project not found"
		}
		return domain.ScoutRun{}, code, msg
	}

	runID, err := s.repo.CreateScoutRun(ctx, projectID, platform)
	if err != nil {
		return domain.ScoutRun{}, http.StatusInternalServerError, "Internal server error"
	}

	run, err := s.repo.GetScoutRun(ctx, projectID, runID)
	if err != nil {
		return domain.ScoutRun{}, http.StatusInternalServerError, "Internal server error"
	}

	// Start pipeline in background (run row already created).
	s.runner.StartAsync(projectID, platform, runID)

	return run, http.StatusCreated, ""
}

// ListRuns returns the latest 20 scout runs for a project.
func (s *ScoutService) ListRuns(ctx context.Context, projectID string) ([]domain.ScoutRun, int, string) {
	runs, err := s.repo.ListScoutRuns(ctx, projectID, 20)
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal server error"
	}
	return runs, http.StatusOK, ""
}

// GetRun returns a single scout run by runID within a project.
func (s *ScoutService) GetRun(ctx context.Context, projectID string, runID int64) (domain.ScoutRun, int, string) {
	run, err := s.repo.GetScoutRun(ctx, projectID, runID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Run not found"
		}
		return domain.ScoutRun{}, code, msg
	}
	return run, http.StatusOK, ""
}

// CancelRun cancels the given run. If tracked, cancels its context; if untracked but
// still running in the DB, marks it failed with "Cancelled".
func (s *ScoutService) CancelRun(ctx context.Context, projectID string, runID int64) (int, string) {
	run, err := s.repo.GetScoutRun(ctx, projectID, runID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Run not found"
		}
		return code, msg
	}

	tracked := s.runner.Cancel(runID)
	if !tracked && run.Status == "running" {
		// Untracked running row — mark failed.
		_ = s.repo.FailScoutRun(ctx, runID, "Cancelled")
	}

	return http.StatusOK, ""
}
