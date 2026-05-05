package scheduler

import (
	"context"
	"log"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// Scheduler manages cron jobs for scout runs.
type Scheduler struct {
	repo   *repository.Repository
	runner *pipeline.Runner
	cron   *cron.Cron
	mu     sync.Mutex
	jobs   map[int64]cron.EntryID
}

// New creates a new Scheduler backed by the given repository and runner.
func New(repo *repository.Repository, runner *pipeline.Runner) *Scheduler {
	return &Scheduler{
		repo:   repo,
		runner: runner,
		cron:   cron.New(),
		jobs:   make(map[int64]cron.EntryID),
	}
}

// LoadAll loads all enabled schedules from the database and registers them.
func (s *Scheduler) LoadAll(ctx context.Context) error {
	schedules, err := s.repo.EnabledSchedules(ctx)
	if err != nil {
		return err
	}
	for _, sch := range schedules {
		if err := s.Register(sch); err != nil {
			log.Printf("scheduler: failed to register schedule %d (%s/%s): %v", sch.ID, sch.ProjectID, sch.Platform, err)
		}
	}
	return nil
}

// Register adds a cron job for the given schedule. If a job for this schedule
// already exists it is replaced.
func (s *Scheduler) Register(schedule domain.Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry if present.
	if entryID, ok := s.jobs[schedule.ID]; ok {
		s.cron.Remove(entryID)
		delete(s.jobs, schedule.ID)
	}

	schedID := schedule.ID
	projectID := schedule.ProjectID
	platform := schedule.Platform

	entryID, err := s.cron.AddFunc(schedule.CronExpr, func() {
		ctx := context.Background()
		log.Printf("scheduler: running schedule %d project=%s platform=%s", schedID, projectID, platform)
		_, runErr := s.runner.Run(ctx, projectID, platform, nil)
		if runErr != nil {
			log.Printf("scheduler: schedule %d run error: %v", schedID, runErr)
		}
		if markErr := s.repo.MarkScheduleRun(ctx, schedID); markErr != nil {
			log.Printf("scheduler: failed to mark schedule %d run: %v", schedID, markErr)
		}
	})
	if err != nil {
		return err
	}

	s.jobs[schedID] = entryID
	return nil
}

// Unregister removes the cron job for the given schedule ID (if any).
func (s *Scheduler) Unregister(scheduleID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.jobs[scheduleID]; ok {
		s.cron.Remove(entryID)
		delete(s.jobs, scheduleID)
	}
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop gracefully stops the scheduler, waiting for running jobs.
func (s *Scheduler) Stop(ctx context.Context) error {
	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
