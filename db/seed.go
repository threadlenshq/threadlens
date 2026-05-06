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

// EnsureCoreSeedMarker marks a named seed as applied in app_settings. It is
// idempotent and concurrency-safe: if the key already exists the function
// returns Status "noop" rather than an error. It works with both SQLite and
// Postgres without requiring a dialect parameter.
func EnsureCoreSeedMarker(ctx context.Context, database *sql.DB, name string, version int) (SeedResult, error) {
	key := fmt.Sprintf("seed.core.%s", name)
	value, err := json.Marshal(map[string]any{"name": name, "version": version})
	if err != nil {
		return SeedResult{}, err
	}

	// Use a WHERE NOT EXISTS sub-select so the same SQL compiles on both
	// SQLite and Postgres — no dialect-specific INSERT OR IGNORE / ON CONFLICT
	// syntax required. The subquery acts as the idempotency guard.
	result, err := database.ExecContext(ctx,
		`INSERT INTO app_settings (key, value, updated_at)
		 SELECT ?, ?, CURRENT_TIMESTAMP
		 WHERE NOT EXISTS (SELECT 1 FROM app_settings WHERE key = ?)`,
		key, string(value), key,
	)
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
