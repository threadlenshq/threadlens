package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
)

// GetSetting retrieves a raw string value from app_settings.
// Returns (value, true, nil) if found, ("", false, nil) if not found, or ("", false, err) on DB error.
func (r *Repository) GetSetting(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := r.DB.QueryRowContext(ctx, "SELECT value FROM app_settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	return value, true, nil
}

// SetSetting inserts or updates an arbitrary key/value string pair in
// app_settings, mirroring GetSetting. Used for non-model settings such as
// the active AI provider chosen at onboarding.
func (r *Repository) SetSetting(ctx context.Context, key, value string) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO app_settings (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE
		SET value = excluded.value, updated_at = datetime('now')
	`, key, value)
	return err
}

// SetModelSetting saves the user-chosen model for a task using the same JSON format as Express:
// { "modelId": "provider:model" }.
func (r *Repository) SetModelSetting(ctx context.Context, taskID string, modelID string) error {
	task := ai.GetTask(taskID)
	if task == nil {
		return fmt.Errorf("unknown task id: %s", taskID)
	}

	model := ai.GetModel(modelID)
	if model == nil {
		return fmt.Errorf("unknown model id: %s", modelID)
	}

	value, err := json.Marshal(map[string]string{"modelId": model.ID})
	if err != nil {
		return err
	}

	key := "model." + taskID
	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO app_settings (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE
		SET value = excluded.value, updated_at = datetime('now')
	`, key, string(value))
	return err
}

// DeleteModelSetting removes the user-chosen model for a task, reverting to the default.
func (r *Repository) DeleteModelSetting(ctx context.Context, taskID string) error {
	task := ai.GetTask(taskID)
	if task == nil {
		return fmt.Errorf("unknown task id: %s", taskID)
	}

	_, err := r.DB.ExecContext(ctx, "DELETE FROM app_settings WHERE key = ?", "model."+taskID)
	return err
}
