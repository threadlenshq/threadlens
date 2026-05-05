package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

func (r *Repository) ListSchedules(ctx context.Context, projectID string) ([]domain.Schedule, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, project_id, platform, cron_expr, enabled, last_run_at, created_at FROM schedules WHERE project_id = ? ORDER BY created_at ASC",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []domain.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if schedules == nil {
		schedules = []domain.Schedule{}
	}
	return schedules, nil
}

func (r *Repository) CreateSchedule(ctx context.Context, projectID string, platform string, cronExpr string) (domain.Schedule, error) {
	result, err := r.DB.ExecContext(ctx,
		`INSERT INTO schedules (project_id, platform, cron_expr, enabled, created_at)
		 VALUES (?, ?, ?, 1, datetime('now'))`,
		projectID, platform, cronExpr,
	)
	if err != nil {
		return domain.Schedule{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.Schedule{}, err
	}
	return r.getScheduleByID(ctx, id)
}

func (r *Repository) GetSchedule(ctx context.Context, projectID string, scheduleID int64) (domain.Schedule, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT id, project_id, platform, cron_expr, enabled, last_run_at, created_at FROM schedules WHERE id = ? AND project_id = ?",
		scheduleID, projectID,
	)
	s, err := scanScheduleRow(row)
	if err == sql.ErrNoRows {
		return domain.Schedule{}, fmt.Errorf("%w: Schedule not found", ErrNotFound)
	}
	return s, err
}

func (r *Repository) PatchSchedule(ctx context.Context, projectID string, scheduleID int64, body map[string]any) (domain.Schedule, error) {
	existing, err := r.GetSchedule(ctx, projectID, scheduleID)
	if err != nil {
		return domain.Schedule{}, err
	}

	allowed := []string{"cron_expr", "enabled"}
	var updates []string
	var values []any

	for _, field := range allowed {
		if val, ok := body[field]; ok {
			updates = append(updates, field+" = ?")
			if field == "enabled" {
				switch v := val.(type) {
				case bool:
					if v {
						values = append(values, int64(1))
					} else {
						values = append(values, int64(0))
					}
				default:
					values = append(values, val)
				}
			} else {
				values = append(values, val)
			}
		}
	}

	if len(updates) == 0 {
		return existing, nil
	}

	values = append(values, scheduleID)
	query := "UPDATE schedules SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	if _, err := r.DB.ExecContext(ctx, query, values...); err != nil {
		return domain.Schedule{}, err
	}

	return r.getScheduleByID(ctx, scheduleID)
}

func (r *Repository) DeleteSchedule(ctx context.Context, projectID string, scheduleID int64) error {
	if _, err := r.GetSchedule(ctx, projectID, scheduleID); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx, "DELETE FROM schedules WHERE id = ?", scheduleID)
	return err
}

func (r *Repository) EnabledSchedules(ctx context.Context) ([]domain.Schedule, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, project_id, platform, cron_expr, enabled, last_run_at, created_at FROM schedules WHERE enabled = 1",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []domain.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if schedules == nil {
		schedules = []domain.Schedule{}
	}
	return schedules, nil
}

func (r *Repository) MarkScheduleRun(ctx context.Context, scheduleID int64) error {
	_, err := r.DB.ExecContext(ctx,
		"UPDATE schedules SET last_run_at = datetime('now') WHERE id = ?",
		scheduleID,
	)
	return err
}

func (r *Repository) getScheduleByID(ctx context.Context, id int64) (domain.Schedule, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT id, project_id, platform, cron_expr, enabled, last_run_at, created_at FROM schedules WHERE id = ?",
		id,
	)
	s, err := scanScheduleRow(row)
	if err == sql.ErrNoRows {
		return domain.Schedule{}, fmt.Errorf("%w: Schedule not found", ErrNotFound)
	}
	return s, err
}

func scanSchedule(rows *sql.Rows) (domain.Schedule, error) {
	var s domain.Schedule
	err := rows.Scan(&s.ID, &s.ProjectID, &s.Platform, &s.CronExpr, &s.Enabled, &s.LastRunAt, &s.CreatedAt)
	return s, err
}

func scanScheduleRow(row *sql.Row) (domain.Schedule, error) {
	var s domain.Schedule
	err := row.Scan(&s.ID, &s.ProjectID, &s.Platform, &s.CronExpr, &s.Enabled, &s.LastRunAt, &s.CreatedAt)
	return s, err
}
