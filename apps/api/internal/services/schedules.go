package services

import (
	"context"
	"net/http"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/scheduler"
)

// ScheduleService handles business logic for schedule CRUD.
type ScheduleService struct {
	repo      *repository.Repository
	scheduler *scheduler.Scheduler
}

func NewScheduleService(repo *repository.Repository, sched *scheduler.Scheduler) *ScheduleService {
	return &ScheduleService{repo: repo, scheduler: sched}
}

type ScheduleRequest struct {
	Platform string `json:"platform"`
	CronExpr string `json:"cron_expr"`
}

func (s *ScheduleService) List(ctx context.Context, projectID string) ([]domain.Schedule, int, string) {
	// Verify project exists.
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		code, msg := mapError(err)
		return nil, code, msg
	}
	schedules, err := s.repo.ListSchedules(ctx, projectID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal server error"
	}
	return schedules, http.StatusOK, ""
}

func (s *ScheduleService) Create(ctx context.Context, projectID string, body ScheduleRequest) (domain.Schedule, int, string) {
	// Verify project exists.
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		code, msg := mapError(err)
		return domain.Schedule{}, code, msg
	}

	platform := strings.TrimSpace(body.Platform)
	cronExpr := strings.TrimSpace(body.CronExpr)

	if platform == "" || cronExpr == "" {
		return domain.Schedule{}, http.StatusBadRequest, "platform and cron_expr are required"
	}
	if !validPlatforms[platform] {
		return domain.Schedule{}, http.StatusBadRequest, "platform must be reddit, bluesky, or google"
	}

	sch, err := s.repo.CreateSchedule(ctx, projectID, platform, cronExpr)
	if err != nil {
		return domain.Schedule{}, http.StatusInternalServerError, "Internal server error"
	}

	if sch.Enabled != 0 {
		if regErr := s.scheduler.Register(sch); regErr != nil {
			// Log but don't fail — the schedule is persisted.
			_ = regErr
		}
	}

	return sch, http.StatusCreated, ""
}

func (s *ScheduleService) Patch(ctx context.Context, projectID string, scheduleID int64, body map[string]any) (domain.Schedule, int, string) {
	updated, err := s.repo.PatchSchedule(ctx, projectID, scheduleID, body)
	if err != nil {
		code, msg := mapError(err)
		return domain.Schedule{}, code, msg
	}

	// Sync cron registration.
	s.scheduler.Unregister(scheduleID)
	if updated.Enabled != 0 {
		_ = s.scheduler.Register(updated)
	}

	return updated, http.StatusOK, ""
}

func (s *ScheduleService) Delete(ctx context.Context, projectID string, scheduleID int64) (int, string) {
	s.scheduler.Unregister(scheduleID)
	err := s.repo.DeleteSchedule(ctx, projectID, scheduleID)
	if err != nil {
		return mapError(err)
	}
	return http.StatusNoContent, ""
}
