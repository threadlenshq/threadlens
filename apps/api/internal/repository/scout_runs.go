package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

// CreateScoutRun inserts a new scout_runs row with status='running' and returns its ID.
func (r *Repository) CreateScoutRun(ctx context.Context, projectID string, platform string) (int64, error) {
	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO scout_runs (project_id, platform, status) VALUES (?, ?, 'running')`,
		projectID, platform,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListScoutRuns returns up to limit scout runs for a project, ordered by started_at DESC.
func (r *Repository) ListScoutRuns(ctx context.Context, projectID string, limit int) ([]domain.ScoutRun, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, project_id, platform, started_at, completed_at,
		       posts_checked, posts_found, status, error, step, warnings
		FROM scout_runs
		WHERE project_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []domain.ScoutRun
	for rows.Next() {
		run, err := scanScoutRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if runs == nil {
		runs = []domain.ScoutRun{}
	}
	return runs, nil
}

// GetScoutRun returns a single scout run by project and run ID.
func (r *Repository) GetScoutRun(ctx context.Context, projectID string, runID int64) (domain.ScoutRun, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, project_id, platform, started_at, completed_at,
		       posts_checked, posts_found, status, error, step, warnings
		FROM scout_runs
		WHERE id = ? AND project_id = ?
	`, runID, projectID)

	run, err := scanScoutRunRow(row)
	if err == sql.ErrNoRows {
		return domain.ScoutRun{}, fmt.Errorf("%w: Run not found", ErrNotFound)
	}
	return run, err
}

// UpdateScoutStep sets the current step string on a running scout run.
func (r *Repository) UpdateScoutStep(ctx context.Context, runID int64, step string) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE scout_runs SET step = ? WHERE id = ?`,
		step, runID,
	)
	return err
}

// CompleteScoutRun marks a run as completed with final counts.
func (r *Repository) CompleteScoutRun(ctx context.Context, runID int64, postsChecked int64, postsFound int64, warnings *string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE scout_runs
		SET status = 'completed',
		    completed_at = datetime('now'),
		    posts_checked = ?,
		    posts_found = ?,
		    warnings = ?
		WHERE id = ?
	`, postsChecked, postsFound, warnings, runID)
	return err
}

// FailScoutRun marks a run as failed with an error message.
func (r *Repository) FailScoutRun(ctx context.Context, runID int64, message string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE scout_runs
		SET status = 'failed',
		    completed_at = datetime('now'),
		    error = ?
		WHERE id = ?
	`, message, runID)
	return err
}

// PlatformRunStats holds aggregated run performance for a single platform.
type PlatformRunStats struct {
	Platform       string
	TotalRuns      int
	RunsWithZero   int
	LastPostsFound int64
}

// RecentPlatformStats returns run performance stats per platform for the last N completed runs.
// Only completed runs (status='completed') are considered.
func (r *Repository) RecentPlatformStats(ctx context.Context, projectID string, lookback int) ([]PlatformRunStats, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT platform,
		       COUNT(*) AS total_runs,
		       SUM(CASE WHEN posts_found = 0 THEN 1 ELSE 0 END) AS runs_with_zero,
		       MAX(posts_found) AS last_posts_found
		FROM (
			SELECT platform, posts_found
			FROM scout_runs
			WHERE project_id = ? AND status = 'completed'
			ORDER BY started_at DESC
			LIMIT ?
		)
		GROUP BY platform
	`, projectID, lookback)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []PlatformRunStats
	for rows.Next() {
		var s PlatformRunStats
		if err := rows.Scan(&s.Platform, &s.TotalRuns, &s.RunsWithZero, &s.LastPostsFound); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// MarkOrphanedRunsFailed marks any running runs older than 5 minutes as failed.
// Returns the number of rows updated.
func (r *Repository) MarkOrphanedRunsFailed(ctx context.Context) (int64, error) {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE scout_runs
		SET status = 'failed',
		    completed_at = datetime('now'),
		    error = 'Server restarted'
		WHERE status = 'running'
		  AND started_at < datetime('now', '-5 minutes')
	`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// scanScoutRun scans from sql.Rows (list query).
func scanScoutRun(rows *sql.Rows) (domain.ScoutRun, error) {
	var run domain.ScoutRun
	var completedAt, errStr, step, warnings sql.NullString
	err := rows.Scan(
		&run.ID, &run.ProjectID, &run.Platform, &run.StartedAt,
		&completedAt, &run.PostsChecked, &run.PostsFound,
		&run.Status, &errStr, &step, &warnings,
	)
	if err != nil {
		return domain.ScoutRun{}, err
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.String
	}
	if errStr.Valid {
		run.Error = &errStr.String
	}
	if step.Valid {
		run.Step = &step.String
	}
	if warnings.Valid {
		run.Warnings = &warnings.String
	}
	return run, nil
}

// scanScoutRunRow scans from sql.Row (single-row query).
func scanScoutRunRow(row *sql.Row) (domain.ScoutRun, error) {
	var run domain.ScoutRun
	var completedAt, errStr, step, warnings sql.NullString
	err := row.Scan(
		&run.ID, &run.ProjectID, &run.Platform, &run.StartedAt,
		&completedAt, &run.PostsChecked, &run.PostsFound,
		&run.Status, &errStr, &step, &warnings,
	)
	if err != nil {
		return domain.ScoutRun{}, err
	}
	if completedAt.Valid {
		run.CompletedAt = &completedAt.String
	}
	if errStr.Valid {
		run.Error = &errStr.String
	}
	if step.Valid {
		run.Step = &step.String
	}
	if warnings.Valid {
		run.Warnings = &warnings.String
	}
	return run, nil
}
