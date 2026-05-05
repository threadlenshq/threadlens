package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

func (r *Repository) ListPrompts(ctx context.Context, projectID string) ([]domain.Prompt, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, project_id, type, platform, prompt_text, created_at, updated_at FROM project_prompts WHERE project_id = ? ORDER BY platform, type",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []domain.Prompt
	for rows.Next() {
		p, err := scanPrompt(rows)
		if err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if prompts == nil {
		prompts = []domain.Prompt{}
	}
	return prompts, nil
}

func (r *Repository) CreatePrompt(ctx context.Context, projectID string, typ string, platform string, promptText string) (domain.Prompt, error) {
	result, err := r.DB.ExecContext(ctx,
		`INSERT INTO project_prompts (project_id, type, platform, prompt_text, created_at, updated_at)
		 VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
		projectID, typ, platform, promptText,
	)
	if err != nil {
		return domain.Prompt{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.Prompt{}, err
	}
	return r.getPromptByID(ctx, id)
}

func (r *Repository) PatchPrompt(ctx context.Context, projectID string, promptID int64, body map[string]any) (domain.Prompt, error) {
	existing, err := r.GetPrompt(ctx, projectID, promptID)
	if err != nil {
		return domain.Prompt{}, err
	}

	allowed := []string{"type", "platform", "prompt_text"}
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
	values = append(values, promptID)
	query := "UPDATE project_prompts SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	if _, err := r.DB.ExecContext(ctx, query, values...); err != nil {
		return domain.Prompt{}, err
	}

	return r.getPromptByID(ctx, promptID)
}

func (r *Repository) DeletePrompt(ctx context.Context, projectID string, promptID int64) error {
	if _, err := r.GetPrompt(ctx, projectID, promptID); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx, "DELETE FROM project_prompts WHERE id = ?", promptID)
	return err
}

func (r *Repository) GetPrompt(ctx context.Context, projectID string, promptID int64) (domain.Prompt, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT id, project_id, type, platform, prompt_text, created_at, updated_at FROM project_prompts WHERE id = ? AND project_id = ?",
		promptID, projectID,
	)
	p, err := scanPromptRow(row)
	if err == sql.ErrNoRows {
		return domain.Prompt{}, fmt.Errorf("%w: Prompt not found", ErrNotFound)
	}
	return p, err
}

func (r *Repository) GetPromptForPost(ctx context.Context, projectID string, typ string, platform string) (domain.Prompt, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT id, project_id, type, platform, prompt_text, created_at, updated_at FROM project_prompts WHERE project_id = ? AND type = ? AND platform = ?",
		projectID, typ, platform,
	)
	p, err := scanPromptRow(row)
	if err == sql.ErrNoRows {
		return domain.Prompt{}, fmt.Errorf("%w: Prompt not found", ErrNotFound)
	}
	return p, err
}

func (r *Repository) getPromptByID(ctx context.Context, id int64) (domain.Prompt, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT id, project_id, type, platform, prompt_text, created_at, updated_at FROM project_prompts WHERE id = ?",
		id,
	)
	p, err := scanPromptRow(row)
	if err == sql.ErrNoRows {
		return domain.Prompt{}, fmt.Errorf("%w: Prompt not found", ErrNotFound)
	}
	return p, err
}

func scanPrompt(rows *sql.Rows) (domain.Prompt, error) {
	var p domain.Prompt
	err := rows.Scan(&p.ID, &p.ProjectID, &p.Type, &p.Platform, &p.PromptText, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func scanPromptRow(row *sql.Row) (domain.Prompt, error) {
	var p domain.Prompt
	err := row.Scan(&p.ID, &p.ProjectID, &p.Type, &p.Platform, &p.PromptText, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}
