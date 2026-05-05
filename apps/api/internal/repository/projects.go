package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

func (r *Repository) ListProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT * FROM projects ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if projects == nil {
		projects = []domain.Project{}
	}
	return projects, nil
}

func (r *Repository) CreateProject(ctx context.Context, p domain.Project) (domain.Project, error) {
	if p.ID == "" || p.Name == "" || p.Mode == "" {
		return domain.Project{}, fmt.Errorf("%w: id, name, and mode are required", ErrValidation)
	}
	if p.Mode != "research" && p.Mode != "marketing" {
		return domain.Project{}, fmt.Errorf("%w: mode must be \"research\" or \"marketing\"", ErrValidation)
	}

	var desc *string
	if p.Description != nil && *p.Description != "" {
		desc = p.Description
	}

	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO projects (id, name, mode, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
		p.ID, p.Name, p.Mode, desc,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return domain.Project{}, fmt.Errorf("%w: Project with that id already exists", ErrConflict)
		}
		return domain.Project{}, err
	}

	return r.GetProject(ctx, p.ID)
}

func (r *Repository) GetProject(ctx context.Context, id string) (domain.Project, error) {
	row := r.DB.QueryRowContext(ctx, "SELECT * FROM projects WHERE id = ?", id)
	p, err := scanProjectRow(row)
	if err == sql.ErrNoRows {
		return domain.Project{}, fmt.Errorf("%w: Project not found", ErrNotFound)
	}
	return p, err
}

func (r *Repository) GetProjectWithStats(ctx context.Context, id string) (domain.ProjectWithStats, error) {
	p, err := r.GetProject(ctx, id)
	if err != nil {
		return domain.ProjectWithStats{}, err
	}

	var totalPosts, newPosts, totalQueries int64

	if err := r.DB.QueryRowContext(ctx,
		"SELECT COUNT(*) as count FROM posts WHERE project_id = ?", id,
	).Scan(&totalPosts); err != nil {
		return domain.ProjectWithStats{}, err
	}

	if err := r.DB.QueryRowContext(ctx,
		"SELECT COUNT(*) as count FROM posts WHERE project_id = ? AND status = 'new'", id,
	).Scan(&newPosts); err != nil {
		return domain.ProjectWithStats{}, err
	}

	if err := r.DB.QueryRowContext(ctx,
		"SELECT COUNT(*) as count FROM project_queries WHERE project_id = ?", id,
	).Scan(&totalQueries); err != nil {
		return domain.ProjectWithStats{}, err
	}

	var lastRun *string
	row := r.DB.QueryRowContext(ctx,
		"SELECT completed_at FROM scout_runs WHERE project_id = ? ORDER BY completed_at DESC LIMIT 1", id,
	)
	var completedAt sql.NullString
	if err := row.Scan(&completedAt); err != nil && err != sql.ErrNoRows {
		return domain.ProjectWithStats{}, err
	}
	if completedAt.Valid {
		lastRun = &completedAt.String
	}

	return domain.ProjectWithStats{
		Project: p,
		Stats: domain.ProjectStats{
			TotalPosts:   totalPosts,
			NewPosts:     newPosts,
			TotalQueries: totalQueries,
			LastRun:      lastRun,
		},
	}, nil
}

func (r *Repository) PatchProject(ctx context.Context, id string, body map[string]any) (domain.Project, error) {
	existing, err := r.GetProject(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}

	allowed := []string{"name", "scoring_prompt", "description"}
	var updates []string
	var values []any

	for _, field := range allowed {
		if val, ok := body[field]; ok {
			updates = append(updates, field+" = ?")
			values = append(values, val)
		}
	}

	if len(updates) == 0 {
		return existing, nil
	}

	updates = append(updates, "updated_at = datetime('now')")
	values = append(values, id)

	query := "UPDATE projects SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	if _, err := r.DB.ExecContext(ctx, query, values...); err != nil {
		return domain.Project{}, err
	}

	return r.GetProject(ctx, id)
}

func (r *Repository) DeleteProject(ctx context.Context, id string) error {
	if _, err := r.GetProject(ctx, id); err != nil {
		return err
	}

	_, err := r.DB.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", id)
	return err
}

func (r *Repository) CloneProject(ctx context.Context, sourceID string, newID string, name string) (domain.Project, error) {
	src, err := r.GetProject(ctx, sourceID)
	if err != nil {
		return domain.Project{}, err
	}

	if newID == "" || name == "" {
		return domain.Project{}, fmt.Errorf("%w: id and name are required", ErrValidation)
	}

	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO projects (id, name, mode, scoring_prompt, description, created_at, updated_at)
		 VALUES (?, ?, 'research', ?, ?, datetime('now'), datetime('now'))`,
		newID, name, src.ScoringPrompt, src.Description,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return domain.Project{}, fmt.Errorf("%w: Project with that id already exists", ErrConflict)
		}
		return domain.Project{}, err
	}

	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO project_queries (project_id, platform, query_url, angle, enabled)
		 SELECT ?, platform, query_url, angle, enabled
		 FROM project_queries WHERE project_id = ?`,
		newID, sourceID,
	)
	if err != nil {
		return domain.Project{}, err
	}

	return r.GetProject(ctx, newID)
}

func (r *Repository) SelectAngle(ctx context.Context, projectID string, reportID int64, clusterIndex int64) (domain.Project, error) {
	if _, err := r.GetProject(ctx, projectID); err != nil {
		return domain.Project{}, err
	}

	_, err := r.DB.ExecContext(ctx,
		`UPDATE projects SET selected_report_id = ?, selected_cluster_index = ?, updated_at = datetime('now') WHERE id = ?`,
		reportID, clusterIndex, projectID,
	)
	if err != nil {
		return domain.Project{}, err
	}

	return r.GetProject(ctx, projectID)
}

func (r *Repository) GraduateProject(ctx context.Context, projectID string) (domain.Project, error) {
	p, err := r.GetProject(ctx, projectID)
	if err != nil {
		return domain.Project{}, err
	}

	if p.Mode != "research" {
		return domain.Project{}, fmt.Errorf("%w: Project is already in marketing mode", ErrValidation)
	}

	if p.SelectedReportID == nil {
		return domain.Project{}, fmt.Errorf("%w: Select a product angle before graduating to marketing mode", ErrValidation)
	}

	_, err = r.DB.ExecContext(ctx,
		`UPDATE projects SET mode = 'marketing', updated_at = datetime('now') WHERE id = ?`,
		projectID,
	)
	if err != nil {
		return domain.Project{}, err
	}

	return r.GetProject(ctx, projectID)
}

// scanProject scans from sql.Rows
func scanProject(rows *sql.Rows) (domain.Project, error) {
	var p domain.Project
	var scoringPrompt, description sql.NullString
	var selectedReportID, selectedClusterIndex sql.NullInt64
	err := rows.Scan(
		&p.ID, &p.Name, &p.Mode,
		&scoringPrompt, &description,
		&selectedReportID, &selectedClusterIndex,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return domain.Project{}, err
	}
	if scoringPrompt.Valid {
		p.ScoringPrompt = &scoringPrompt.String
	}
	if description.Valid {
		p.Description = &description.String
	}
	if selectedReportID.Valid {
		p.SelectedReportID = &selectedReportID.Int64
	}
	if selectedClusterIndex.Valid {
		p.SelectedClusterIndex = &selectedClusterIndex.Int64
	}
	return p, nil
}

// scanProjectRow scans from sql.Row
func scanProjectRow(row *sql.Row) (domain.Project, error) {
	var p domain.Project
	var scoringPrompt, description sql.NullString
	var selectedReportID, selectedClusterIndex sql.NullInt64
	err := row.Scan(
		&p.ID, &p.Name, &p.Mode,
		&scoringPrompt, &description,
		&selectedReportID, &selectedClusterIndex,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return domain.Project{}, err
	}
	if scoringPrompt.Valid {
		p.ScoringPrompt = &scoringPrompt.String
	}
	if description.Valid {
		p.Description = &description.String
	}
	if selectedReportID.Valid {
		p.SelectedReportID = &selectedReportID.Int64
	}
	if selectedClusterIndex.Valid {
		p.SelectedClusterIndex = &selectedClusterIndex.Int64
	}
	return p, nil
}
