package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

func marshalPlatforms(platforms []string) string {
	if platforms == nil {
		platforms = []string{}
	}
	b, _ := json.Marshal(platforms)
	return string(b)
}

func scanGeneralQuery(s rowScanner) (domain.GeneralQuery, error) {
	var g domain.GeneralQuery
	var platformsJSON string
	if err := s.Scan(&g.ID, &g.ProjectID, &g.QueryText, &g.Angle, &platformsJSON, &g.Enabled, &g.CreatedAt); err != nil {
		return domain.GeneralQuery{}, err
	}
	if platformsJSON == "" {
		platformsJSON = "[]"
	}
	if err := json.Unmarshal([]byte(platformsJSON), &g.Platforms); err != nil {
		return domain.GeneralQuery{}, err
	}
	if g.Platforms == nil {
		g.Platforms = []string{}
	}
	return g, nil
}

const generalQueryCols = "id, project_id, query_text, angle, platforms, enabled, created_at"

func (r *Repository) CreateGeneralQuery(ctx context.Context, projectID, queryText, angle string, platforms []string, enabled bool) (domain.GeneralQuery, error) {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO general_queries (project_id, query_text, angle, platforms, enabled, created_at)
		 VALUES (?, ?, ?, ?, ?, datetime('now'))`,
		projectID, queryText, angle, marshalPlatforms(platforms), enabledVal,
	)
	if err != nil {
		return domain.GeneralQuery{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.GeneralQuery{}, err
	}
	return r.GetGeneralQuery(ctx, projectID, id)
}

func (r *Repository) ListGeneralQueries(ctx context.Context, projectID string) ([]domain.GeneralQuery, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT "+generalQueryCols+" FROM general_queries WHERE project_id = ? ORDER BY created_at", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GeneralQuery{}
	for rows.Next() {
		g, err := scanGeneralQuery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repository) GetGeneralQuery(ctx context.Context, projectID string, id int64) (domain.GeneralQuery, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT "+generalQueryCols+" FROM general_queries WHERE id = ? AND project_id = ?", id, projectID)
	g, err := scanGeneralQuery(row)
	if err == sql.ErrNoRows {
		return domain.GeneralQuery{}, fmt.Errorf("%w: General query not found", ErrNotFound)
	}
	return g, err
}

func (r *Repository) UpdateGeneralQueryMeta(ctx context.Context, projectID string, id int64, queryText, angle string, platforms []string, enabled bool) (domain.GeneralQuery, error) {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	if _, err := r.DB.ExecContext(ctx,
		`UPDATE general_queries SET query_text = ?, angle = ?, platforms = ?, enabled = ?
		 WHERE id = ? AND project_id = ?`,
		queryText, angle, marshalPlatforms(platforms), enabledVal, id, projectID,
	); err != nil {
		return domain.GeneralQuery{}, err
	}
	return r.GetGeneralQuery(ctx, projectID, id)
}

func (r *Repository) DeleteGeneralQuery(ctx context.Context, projectID string, id int64) error {
	if _, err := r.GetGeneralQuery(ctx, projectID, id); err != nil {
		return err
	}
	// Materialized project_queries rows cascade via ON DELETE CASCADE.
	_, err := r.DB.ExecContext(ctx, "DELETE FROM general_queries WHERE id = ? AND project_id = ?", id, projectID)
	return err
}

func (r *Repository) InsertMaterializedQuery(ctx context.Context, projectID string, generalQueryID int64, platform, queryURL, angle string, enabled bool) (domain.Query, error) {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO project_queries (project_id, platform, query_url, angle, enabled, created_at, general_query_id)
		 VALUES (?, ?, ?, ?, ?, datetime('now'), ?)`,
		projectID, platform, queryURL, angle, enabledVal, generalQueryID,
	)
	if err != nil {
		return domain.Query{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Query{}, err
	}
	return r.getQueryByID(ctx, id)
}

func (r *Repository) DeleteMaterializedQueries(ctx context.Context, generalQueryID int64) error {
	_, err := r.DB.ExecContext(ctx, "DELETE FROM project_queries WHERE general_query_id = ?", generalQueryID)
	return err
}
