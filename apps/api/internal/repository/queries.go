package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

func (r *Repository) ListQueries(ctx context.Context, projectID string) ([]domain.Query, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, project_id, platform, query_url, angle, enabled, created_at FROM project_queries WHERE project_id = ? ORDER BY created_at",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []domain.Query
	for rows.Next() {
		q, err := scanQuery(rows)
		if err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if queries == nil {
		queries = []domain.Query{}
	}
	return queries, nil
}

func (r *Repository) CreateQuery(ctx context.Context, projectID string, platform string, queryURL string, angle string, enabled bool) (domain.Query, error) {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	result, err := r.DB.ExecContext(ctx,
		`INSERT INTO project_queries (project_id, platform, query_url, angle, enabled, created_at)
		 VALUES (?, ?, ?, ?, ?, datetime('now'))`,
		projectID, platform, queryURL, angle, enabledVal,
	)
	if err != nil {
		return domain.Query{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.Query{}, err
	}
	return r.getQueryByID(ctx, id)
}

func (r *Repository) PatchQuery(ctx context.Context, projectID string, queryID int64, body map[string]any) (domain.Query, error) {
	existing, err := r.GetQuery(ctx, projectID, queryID)
	if err != nil {
		return domain.Query{}, err
	}

	allowed := []string{"platform", "query_url", "angle", "enabled"}
	var updates []string
	var values []any

	for _, field := range allowed {
		if val, ok := body[field]; ok {
			updates = append(updates, field+" = ?")
			if field == "enabled" {
				// convert bool/number to int
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

	values = append(values, queryID)
	query := "UPDATE project_queries SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	if _, err := r.DB.ExecContext(ctx, query, values...); err != nil {
		return domain.Query{}, err
	}

	return r.getQueryByID(ctx, queryID)
}

func (r *Repository) DeleteQuery(ctx context.Context, projectID string, queryID int64) error {
	if _, err := r.GetQuery(ctx, projectID, queryID); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx, "DELETE FROM project_queries WHERE id = ?", queryID)
	return err
}

// SocialReportForAI holds the columns needed by query AI helpers.
type SocialReportForAI struct {
	ID          int64
	Title       string
	Assessment  string
	Clusters    string
	CreatedAt   string
	CompletedAt sql.NullString
}

// GoogleReportForAI holds the columns needed by query AI helpers.
type GoogleReportForAI struct {
	ID                   int64
	RunID                int64
	ExecutiveSummaryJSON string
	KeywordSummaryJSON   string
	OpportunitiesJSON    string
	RisksJSON            string
	NextActionsJSON      string
	CreatedAt            string
	UpdatedAt            string
}

// GoogleKeywordRow holds summary data for a single keyword in a google run.
type GoogleKeywordRow struct {
	RootKeyword        string
	TotalResults       int64
	RelevantResults    int64
	OutreachCandidates int64
	AvgRelevanceScore  float64
	AvgConfidenceScore float64
}

// PickSocialReport looks up the appropriate social report for the given project.
// If requestedID is non-zero it fetches that report. If selectedID is non-zero it
// tries that next. Falls back to the most recent completed report.
func (r *Repository) PickSocialReport(ctx context.Context, projectID string, requestedID, selectedID int64) (report *SocialReportForAI, source string, err error) {
	scan := func(row *sql.Row) (*SocialReportForAI, error) {
		var rep SocialReportForAI
		if err := row.Scan(&rep.ID, &rep.Title, &rep.Assessment, &rep.Clusters, &rep.CreatedAt, &rep.CompletedAt); err != nil {
			return nil, err
		}
		return &rep, nil
	}

	if requestedID != 0 {
		row := r.DB.QueryRowContext(ctx,
			`SELECT id, title, assessment, clusters, created_at, completed_at
			 FROM research_reports
			 WHERE id = ? AND project_id = ? AND status = 'completed'`,
			requestedID, projectID)
		rep, err := scan(row)
		if err == sql.ErrNoRows {
			return nil, "requested", nil
		}
		if err != nil {
			return nil, "requested", err
		}
		return rep, "requested", nil
	}

	if selectedID != 0 {
		row := r.DB.QueryRowContext(ctx,
			`SELECT id, title, assessment, clusters, created_at, completed_at
			 FROM research_reports
			 WHERE id = ? AND project_id = ? AND status = 'completed'`,
			selectedID, projectID)
		rep, err := scan(row)
		if err == nil {
			return rep, "selected", nil
		}
		if err != sql.ErrNoRows {
			return nil, "selected", err
		}
	}

	row := r.DB.QueryRowContext(ctx,
		`SELECT id, title, assessment, clusters, created_at, completed_at
		 FROM research_reports
		 WHERE project_id = ? AND status = 'completed'
		 ORDER BY completed_at DESC, created_at DESC, id DESC
		 LIMIT 1`,
		projectID)
	rep, err := scan(row)
	if err == sql.ErrNoRows {
		return nil, "none", nil
	}
	if err != nil {
		return nil, "none", err
	}
	return rep, "latest", nil
}

// PickGoogleReport looks up the appropriate google report.
// If requestedID is non-zero it fetches that specific report. Otherwise returns the latest.
func (r *Repository) PickGoogleReport(ctx context.Context, projectID string, requestedID int64) (report *GoogleReportForAI, source string, err error) {
	scan := func(row *sql.Row) (*GoogleReportForAI, error) {
		var rep GoogleReportForAI
		if err := row.Scan(&rep.ID, &rep.RunID, &rep.ExecutiveSummaryJSON, &rep.KeywordSummaryJSON,
			&rep.OpportunitiesJSON, &rep.RisksJSON, &rep.NextActionsJSON, &rep.CreatedAt, &rep.UpdatedAt); err != nil {
			return nil, err
		}
		return &rep, nil
	}

	if requestedID != 0 {
		row := r.DB.QueryRowContext(ctx,
			`SELECT gr.id, gr.run_id, gr.executive_summary_json, gr.keyword_summary_json, gr.opportunities_json,
			        gr.risks_json, gr.next_actions_json, gr.created_at, gr.updated_at
			 FROM google_reports gr
			 INNER JOIN scout_runs sr ON sr.id = gr.run_id
			 WHERE gr.id = ?
			   AND gr.project_id = ?
			   AND sr.project_id = ?
			   AND sr.platform = 'google'
			   AND sr.status = 'completed'
			   AND sr.completed_at IS NOT NULL`,
			requestedID, projectID, projectID)
		rep, err := scan(row)
		if err == sql.ErrNoRows {
			return nil, "requested", nil
		}
		if err != nil {
			return nil, "requested", err
		}
		return rep, "requested", nil
	}

	row := r.DB.QueryRowContext(ctx,
		`SELECT gr.id, gr.run_id, gr.executive_summary_json, gr.keyword_summary_json, gr.opportunities_json,
		        gr.risks_json, gr.next_actions_json, gr.created_at, gr.updated_at
		 FROM google_reports gr
		 INNER JOIN scout_runs sr ON sr.id = gr.run_id
		 WHERE gr.project_id = ?
		   AND sr.project_id = ?
		   AND sr.platform = 'google'
		   AND sr.status = 'completed'
		   AND sr.completed_at IS NOT NULL
		 ORDER BY sr.completed_at DESC, gr.updated_at DESC, gr.created_at DESC, gr.id DESC
		 LIMIT 1`,
		projectID, projectID)
	rep, err := scan(row)
	if err == sql.ErrNoRows {
		return nil, "none", nil
	}
	if err != nil {
		return nil, "none", err
	}
	return rep, "latest", nil
}

// ListGoogleKeywordRows returns keyword summary rows for a given run, used for quality scoring.
func (r *Repository) ListGoogleKeywordRows(ctx context.Context, projectID string, runID int64) ([]GoogleKeywordRow, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT root_keyword, total_results, relevant_results, outreach_candidates, avg_relevance_score, avg_confidence_score
		 FROM google_keyword_summaries
		 WHERE project_id = ? AND run_id = ?`,
		projectID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []GoogleKeywordRow
	for rows.Next() {
		var k GoogleKeywordRow
		var avgRel, avgConf sql.NullFloat64
		if err := rows.Scan(&k.RootKeyword, &k.TotalResults, &k.RelevantResults, &k.OutreachCandidates, &avgRel, &avgConf); err != nil {
			return nil, err
		}
		if avgRel.Valid {
			k.AvgRelevanceScore = avgRel.Float64
		}
		if avgConf.Valid {
			k.AvgConfidenceScore = avgConf.Float64
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// ListAllQueries returns all queries for a project (all platforms, all enabled states) ordered by created_at.
func (r *Repository) ListAllQueries(ctx context.Context, projectID string) ([]domain.Query, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, project_id, platform, query_url, angle, enabled, created_at FROM project_queries WHERE project_id = ? ORDER BY created_at",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []domain.Query
	for rows.Next() {
		q, err := scanQuery(rows)
		if err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if queries == nil {
		queries = []domain.Query{}
	}
	return queries, nil
}

func (r *Repository) EnabledQueries(ctx context.Context, projectID string, platform string) ([]domain.Query, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, project_id, platform, query_url, angle, enabled, created_at FROM project_queries WHERE project_id = ? AND platform = ? AND enabled = 1 ORDER BY created_at",
		projectID, platform,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []domain.Query
	for rows.Next() {
		q, err := scanQuery(rows)
		if err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if queries == nil {
		queries = []domain.Query{}
	}
	return queries, nil
}

func (r *Repository) GetQuery(ctx context.Context, projectID string, queryID int64) (domain.Query, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT id, project_id, platform, query_url, angle, enabled, created_at FROM project_queries WHERE id = ? AND project_id = ?",
		queryID, projectID,
	)
	q, err := scanQuery(row)
	if err == sql.ErrNoRows {
		return domain.Query{}, fmt.Errorf("%w: Query not found", ErrNotFound)
	}
	return q, err
}

func (r *Repository) getQueryByID(ctx context.Context, id int64) (domain.Query, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT id, project_id, platform, query_url, angle, enabled, created_at FROM project_queries WHERE id = ?",
		id,
	)
	q, err := scanQuery(row)
	if err == sql.ErrNoRows {
		return domain.Query{}, fmt.Errorf("%w: Query not found", ErrNotFound)
	}
	return q, err
}

// rowScanner is satisfied by both *sql.Rows and *sql.Row.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanQuery(s rowScanner) (domain.Query, error) {
	var q domain.Query
	err := s.Scan(&q.ID, &q.ProjectID, &q.Platform, &q.QueryURL, &q.Angle, &q.Enabled, &q.CreatedAt)
	return q, err
}
