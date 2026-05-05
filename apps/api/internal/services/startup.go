package services

import (
	"context"
	"log"
	"os"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// RunStartupTasks runs the server boot-time reconciliation steps:
//  1. Mark orphaned scout_runs (running for >5 min) as failed with "Server restarted".
//  2. Kick off council reconciliation for completed reports missing a council row.
//  3. Optionally seed demo data when SCOUT_INIT_DEMO=1.
//
// aiSvc may be nil; it is only used for background council generation (reconciliation
// will insert the row but the background AI call will fail gracefully if nil).
func RunStartupTasks(ctx context.Context, repo *repository.Repository, aiSvc *ai.Service) error {
	// Step 1: Mark orphaned runs failed.
	n, err := repo.MarkOrphanedRunsFailed(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		log.Printf("[startup] marked %d orphaned scout run(s) as failed", n)
	}

	// Step 2: Council reconciliation (non-blocking; errors logged internally).
	pipeline.ReconcileCouncils(ctx, repo.DB, aiSvc, repo)

	// Step 3: Demo seed when opted-in.
	if os.Getenv("SCOUT_INIT_DEMO") == "1" {
		result, err := SeedDemoData(ctx, repo)
		if err != nil {
			log.Printf("[startup] demo seed error: %v", err)
			// Non-fatal — don't prevent server start.
		} else if result.Status == "seeded" {
			log.Printf("[startup] demo data seeded (project=%s version=%d)", result.ProjectID, result.Version)
		}
	}

	return nil
}
