package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// SeedResult describes the outcome of a core seed marker operation.
type SeedResult struct {
	Status  string
	Name    string
	Version int
}

// EnsureCoreSeedMarker marks a named seed as applied in app_settings. It is
// idempotent and concurrency-safe: if the key already exists the function
// returns Status "noop" rather than an error.
//
// The function is compatible with both SQLite and Postgres: it auto-detects
// the driver and selects the appropriate placeholder style without requiring
// the caller to pass a dialect parameter.
func EnsureCoreSeedMarker(ctx context.Context, database *sql.DB, name string, version int) (SeedResult, error) {
	dialect, err := dialectFromDB(database)
	if err != nil {
		return SeedResult{}, fmt.Errorf("EnsureCoreSeedMarker: detect dialect: %w", err)
	}

	key := fmt.Sprintf("seed.core.%s", name)
	value, err := json.Marshal(map[string]any{"name": name, "version": version})
	if err != nil {
		return SeedResult{}, err
	}

	// Use a WHERE NOT EXISTS sub-select so the same SQL shape works on both
	// SQLite and Postgres. Only the placeholder tokens differ between dialects.
	q, err := seedInsertSQL(dialect)
	if err != nil {
		return SeedResult{}, err
	}

	result, err := database.ExecContext(ctx, q, key, string(value), key)
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

// seedInsertSQL returns the INSERT … WHERE NOT EXISTS statement with the
// correct placeholder style for the given dialect.
func seedInsertSQL(dialect Dialect) (string, error) {
	switch dialect {
	case DialectSQLite:
		return `INSERT INTO app_settings (key, value, updated_at)
		 SELECT ?, ?, CURRENT_TIMESTAMP
		 WHERE NOT EXISTS (SELECT 1 FROM app_settings WHERE key = ?)`, nil
	case DialectPostgres:
		return `INSERT INTO app_settings (key, value, updated_at)
		 SELECT $1, $2, NOW()
		 WHERE NOT EXISTS (SELECT 1 FROM app_settings WHERE key = $3)`, nil
	default:
		return "", fmt.Errorf("seedInsertSQL: unsupported dialect %q", dialect)
	}
}

// dialectFromDB infers the Dialect from the underlying driver registered in
// the *sql.DB. It uses the driver type name, which is stable for the two
// drivers this package imports:
//
//   - modernc.org/sqlite  → "*sqlite.Driver"
//   - jackc/pgx/v5/stdlib → "*stdlib.Driver"
//
// The dialect is never surfaced to callers; it is an internal implementation
// detail of EnsureCoreSeedMarker.
func dialectFromDB(database *sql.DB) (Dialect, error) {
	typeName := fmt.Sprintf("%T", database.Driver())
	switch {
	case strings.Contains(typeName, "sqlite"):
		return DialectSQLite, nil
	case strings.Contains(typeName, "stdlib"), strings.Contains(typeName, "pgx"):
		return DialectPostgres, nil
	default:
		return "", fmt.Errorf("dialectFromDB: unrecognised driver type %q", typeName)
	}
}
