package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

func (r *Repository) CreateQueryReviewJob(ctx context.Context, projectID string, kind domain.QueryReviewKind, step, refinement string) (domain.QueryReviewJob, error) {
	res, err := r.DB.ExecContext(ctx, `
		INSERT INTO query_review_jobs (project_id, kind, status, step, refinement)
		VALUES (?, ?, 'running', ?, ?)
	`, projectID, kind, step, refinement)
	if err != nil {
		return domain.QueryReviewJob{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.QueryReviewJob{}, err
	}
	return r.GetQueryReviewJob(ctx, projectID, id)
}

func (r *Repository) ListQueryReviewJobs(ctx context.Context, projectID string, limit int) ([]domain.QueryReviewJob, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, project_id, kind, status, COALESCE(step, ''), COALESCE(refinement, ''),
		       started_at, completed_at, reviewed_at, resolution, error, result_json
		FROM query_review_jobs
		WHERE project_id = ?
		  AND (status = 'running' OR reviewed_at IS NULL)
		ORDER BY started_at DESC, id DESC
		LIMIT ?
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []domain.QueryReviewJob{}
	for rows.Next() {
		job, err := scanQueryReviewJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *Repository) GetQueryReviewJob(ctx context.Context, projectID string, jobID int64) (domain.QueryReviewJob, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, project_id, kind, status, COALESCE(step, ''), COALESCE(refinement, ''),
		       started_at, completed_at, reviewed_at, resolution, error, result_json
		FROM query_review_jobs
		WHERE id = ? AND project_id = ?
	`, jobID, projectID)
	job, err := scanQueryReviewJobRow(row)
	if err == sql.ErrNoRows {
		return domain.QueryReviewJob{}, fmt.Errorf("Query review job not found: %w", ErrNotFound)
	}
	return job, err
}

func (r *Repository) CompleteQueryReviewJob(ctx context.Context, projectID string, jobID int64, result any) (domain.QueryReviewJob, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return domain.QueryReviewJob{}, err
	}
	res, err := r.DB.ExecContext(ctx, `
		UPDATE query_review_jobs
		SET status = 'completed', completed_at = datetime('now'), result_json = ?, error = NULL
		WHERE id = ? AND project_id = ?
	`, string(payload), jobID, projectID)
	if err != nil {
		return domain.QueryReviewJob{}, err
	}
	if n, err2 := res.RowsAffected(); err2 != nil {
		return domain.QueryReviewJob{}, err2
	} else if n == 0 {
		return domain.QueryReviewJob{}, fmt.Errorf("Query review job not found: %w", ErrNotFound)
	}
	return r.GetQueryReviewJob(ctx, projectID, jobID)
}

func (r *Repository) FailQueryReviewJob(ctx context.Context, projectID string, jobID int64, message string) (domain.QueryReviewJob, error) {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE query_review_jobs
		SET status = 'failed', completed_at = datetime('now'), error = ?, result_json = NULL
		WHERE id = ? AND project_id = ?
	`, message, jobID, projectID)
	if err != nil {
		return domain.QueryReviewJob{}, err
	}
	if n, err2 := res.RowsAffected(); err2 != nil {
		return domain.QueryReviewJob{}, err2
	} else if n == 0 {
		return domain.QueryReviewJob{}, fmt.Errorf("Query review job not found: %w", ErrNotFound)
	}
	return r.GetQueryReviewJob(ctx, projectID, jobID)
}

func (r *Repository) MarkQueryReviewJobReviewed(ctx context.Context, projectID string, jobID int64, resolution domain.QueryReviewResolution) (domain.QueryReviewJob, error) {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE query_review_jobs
		SET reviewed_at = datetime('now'), resolution = ?
		WHERE id = ? AND project_id = ?
	`, resolution, jobID, projectID)
	if err != nil {
		return domain.QueryReviewJob{}, err
	}
	if n, err2 := res.RowsAffected(); err2 != nil {
		return domain.QueryReviewJob{}, err2
	} else if n == 0 {
		return domain.QueryReviewJob{}, fmt.Errorf("Query review job not found: %w", ErrNotFound)
	}
	return r.GetQueryReviewJob(ctx, projectID, jobID)
}

func (r *Repository) MarkStaleQueryReviewJobsFailed(ctx context.Context) (int64, error) {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE query_review_jobs
		SET status = 'failed', completed_at = datetime('now'), error = 'Server restarted before query review finished'
		WHERE status = 'running'
		  AND started_at < datetime('now', '-15 minutes')
	`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type queryReviewScanner interface {
	Scan(dest ...any) error
}

func scanQueryReviewJob(rows *sql.Rows) (domain.QueryReviewJob, error) {
	return scanQueryReviewJobAny(rows)
}

func scanQueryReviewJobRow(row *sql.Row) (domain.QueryReviewJob, error) {
	return scanQueryReviewJobAny(row)
}

func scanQueryReviewJobAny(scanner queryReviewScanner) (domain.QueryReviewJob, error) {
	var job domain.QueryReviewJob
	var completedAt, reviewedAt, resolution, errStr, resultJSON sql.NullString
	err := scanner.Scan(
		&job.ID, &job.ProjectID, &job.Kind, &job.Status, &job.Step, &job.Refinement,
		&job.StartedAt, &completedAt, &reviewedAt, &resolution, &errStr, &resultJSON,
	)
	if err != nil {
		return domain.QueryReviewJob{}, err
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.String
	}
	if reviewedAt.Valid {
		job.ReviewedAt = &reviewedAt.String
	}
	if resolution.Valid {
		r := domain.QueryReviewResolution(resolution.String)
		job.Resolution = &r
	}
	if errStr.Valid {
		job.Error = &errStr.String
	}
	if resultJSON.Valid {
		job.Result = json.RawMessage(resultJSON.String)
	}
	return job, nil
}
