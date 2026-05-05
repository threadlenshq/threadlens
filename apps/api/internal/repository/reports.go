package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

// ListReports returns all research reports for a project (with council status joined).
func (r *Repository) ListReports(ctx context.Context, projectID string) ([]domain.ResearchReport, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT rr.id, rr.project_id, rr.title, rr.status, rr.post_count, rr.clusters, rr.assessment,
		       rr.model_used, rr.created_at, rr.completed_at, rr.error,
		       rc.status AS council_status, rc.completed_at AS council_completed_at, rc.error AS council_error
		FROM research_reports rr
		LEFT JOIN report_councils rc ON rc.report_id = rr.id
		WHERE rr.project_id = ?
		ORDER BY rr.created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []domain.ResearchReport
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if reports == nil {
		reports = []domain.ResearchReport{}
	}
	return reports, nil
}

// GetReport returns a single report by projectID and reportID (with council status joined).
func (r *Repository) GetReport(ctx context.Context, projectID string, reportID int64) (domain.ResearchReport, error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT rr.id, rr.project_id, rr.title, rr.status, rr.post_count, rr.clusters, rr.assessment,
		       rr.model_used, rr.created_at, rr.completed_at, rr.error,
		       rc.status AS council_status, rc.completed_at AS council_completed_at, rc.error AS council_error
		FROM research_reports rr
		LEFT JOIN report_councils rc ON rc.report_id = rr.id
		WHERE rr.id = ? AND rr.project_id = ?
	`, reportID, projectID)

	rep, err := scanReportRow(row)
	if err == sql.ErrNoRows {
		return domain.ResearchReport{}, fmt.Errorf("%w: Report not found", ErrNotFound)
	}
	return rep, err
}

// GetReportCouncilJSON returns the raw council_json for a report's council row.
func (r *Repository) GetReportCouncilJSON(ctx context.Context, projectID string, reportID int64) (json.RawMessage, error) {
	var councilJSON string
	err := r.DB.QueryRowContext(ctx,
		`SELECT council_json FROM report_councils WHERE report_id = ? AND project_id = ?`,
		reportID, projectID,
	).Scan(&councilJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: Council not found", ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	// Validate JSON – fall back to empty object on parse failure.
	if !json.Valid([]byte(councilJSON)) {
		return json.RawMessage(`{}`), nil
	}
	return json.RawMessage(councilJSON), nil
}

// StartAnalysis inserts a new research_reports row with status='running' and returns its ID.
func (r *Repository) StartAnalysis(ctx context.Context, projectID string, modelID string) (int64, error) {
	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO research_reports (project_id, status, model_used) VALUES (?, 'running', ?)`,
		projectID, modelID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// MarkReportFailed marks a report as failed with the given message.
func (r *Repository) MarkReportFailed(ctx context.Context, reportID int64, message string) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE research_reports SET status = 'failed', error = ? WHERE id = ?`,
		message, reportID,
	)
	return err
}

// CompleteReport marks a report as completed and stores all analysis fields.
func (r *Repository) CompleteReport(ctx context.Context, reportID int64, title string, postCount int64, clusters []byte, assessment string, modelUsed string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE research_reports
		SET status = 'completed',
		    title = ?,
		    post_count = ?,
		    clusters = ?,
		    assessment = ?,
		    model_used = ?,
		    completed_at = datetime('now')
		WHERE id = ?
	`, title, postCount, string(clusters), assessment, modelUsed, reportID)
	return err
}

// scanReport scans a report row from sql.Rows (list query with council join).
func scanReport(rows *sql.Rows) (domain.ResearchReport, error) {
	var rep domain.ResearchReport
	var clustersStr string
	var completedAt, councilStatus, councilCompletedAt, councilError, reportError sql.NullString
	err := rows.Scan(
		&rep.ID, &rep.ProjectID, &rep.Title, &rep.Status, &rep.PostCount,
		&clustersStr, &rep.Assessment, &rep.ModelUsed, &rep.CreatedAt,
		&completedAt, &reportError,
		&councilStatus, &councilCompletedAt, &councilError,
	)
	if err != nil {
		return domain.ResearchReport{}, err
	}
	// Parse clusters JSON or default to empty array.
	if json.Valid([]byte(clustersStr)) {
		rep.Clusters = json.RawMessage(clustersStr)
	} else {
		rep.Clusters = json.RawMessage(`[]`)
	}
	if completedAt.Valid {
		rep.CompletedAt = &completedAt.String
	}
	if reportError.Valid {
		rep.Error = &reportError.String
	}
	if councilStatus.Valid {
		rep.CouncilStatus = &councilStatus.String
	}
	if councilCompletedAt.Valid {
		rep.CouncilCompletedAt = &councilCompletedAt.String
	}
	if councilError.Valid {
		rep.CouncilError = &councilError.String
	}
	return rep, nil
}

// scanReportRow scans from sql.Row (single-row query with council join).
func scanReportRow(row *sql.Row) (domain.ResearchReport, error) {
	var rep domain.ResearchReport
	var clustersStr string
	var completedAt, councilStatus, councilCompletedAt, councilError, reportError sql.NullString
	err := row.Scan(
		&rep.ID, &rep.ProjectID, &rep.Title, &rep.Status, &rep.PostCount,
		&clustersStr, &rep.Assessment, &rep.ModelUsed, &rep.CreatedAt,
		&completedAt, &reportError,
		&councilStatus, &councilCompletedAt, &councilError,
	)
	if err != nil {
		return domain.ResearchReport{}, err
	}
	if json.Valid([]byte(clustersStr)) {
		rep.Clusters = json.RawMessage(clustersStr)
	} else {
		rep.Clusters = json.RawMessage(`[]`)
	}
	if completedAt.Valid {
		rep.CompletedAt = &completedAt.String
	}
	if reportError.Valid {
		rep.Error = &reportError.String
	}
	if councilStatus.Valid {
		rep.CouncilStatus = &councilStatus.String
	}
	if councilCompletedAt.Valid {
		rep.CouncilCompletedAt = &councilCompletedAt.String
	}
	if councilError.Valid {
		rep.CouncilError = &councilError.String
	}
	return rep, nil
}
