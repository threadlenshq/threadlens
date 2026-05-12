// Package settings provides a generic key/value repository backed by the
// app_settings table. It is intentionally minimal: Get, Set, Delete, and
// ListPrefix. No JSON serialisation, no domain knowledge — callers own that.
package settings

import (
	"context"
	"database/sql"
	"errors"
)

// Repository reads and writes rows in the app_settings table.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a Repository that uses the provided *sql.DB.
// The app_settings table must already exist (it is created by db.InitSchema).
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Get returns the value stored for key.
// Returns (value, true, nil) when found, ("", false, nil) when absent,
// or ("", false, err) on a database error.
func (r *Repository) Get(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := r.db.QueryRowContext(ctx,
		"SELECT value FROM app_settings WHERE key = ?", key,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// Set inserts or updates the key with the given value.
func (r *Repository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE
		SET value = excluded.value,
		    updated_at = datetime('now')
	`, key, value)
	return err
}

// Delete removes the key. It is not an error if the key does not exist.
func (r *Repository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM app_settings WHERE key = ?", key,
	)
	return err
}

// ListPrefix returns all key/value pairs whose key starts with prefix.
// An empty map (not nil) is returned when there are no matches.
func (r *Repository) ListPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT key, value FROM app_settings WHERE key LIKE ? ESCAPE '\\'",
		escapeLike(prefix)+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

// escapeLike escapes SQLite LIKE special characters in s so it can be used
// safely as a prefix pattern.
func escapeLike(s string) string {
	out := make([]byte, 0, len(s)+4)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '%', '_', '\\':
			out = append(out, '\\')
		}
		out = append(out, s[i])
	}
	return string(out)
}
