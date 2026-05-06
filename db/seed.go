package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// SeedResult describes the outcome of a core seed marker operation.
type SeedResult struct {
	Status  string
	Name    string
	Version int
}

// EnsureCoreSeedMarker marks a named seed as applied in app_settings for the
// given dialect. It is idempotent and concurrency-safe: a concurrent insert
// that races to the same key returns Status "noop" rather than an error.
func EnsureCoreSeedMarker(ctx context.Context, database *sql.DB, dialect Dialect, name string, version int) (SeedResult, error) {
	key := fmt.Sprintf("seed.core.%s", name)
	value, err := json.Marshal(map[string]any{"name": name, "version": version})
	if err != nil {
		return SeedResult{}, err
	}

	var result sql.Result
	switch dialect {
	case DialectPostgres:
		result, err = database.ExecContext(ctx,
			`INSERT INTO app_settings (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO NOTHING`,
			key, string(value),
		)
	default: // DialectSQLite
		result, err = database.ExecContext(ctx,
			`INSERT OR IGNORE INTO app_settings (key, value, updated_at) VALUES (?, ?, datetime('now'))`,
			key, string(value),
		)
	}
	if err != nil {
		return SeedResult{}, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return SeedResult{}, err
	}
	if rows == 0 {
		return SeedResult{Status: "noop", Name: name, Version: version}, nil
	}
	return SeedResult{Status: "seeded", Name: name, Version: version}, nil
}
