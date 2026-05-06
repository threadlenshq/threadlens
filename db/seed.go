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
// returns Status "noop" rather than an error.
//
// The function is compatible with both SQLite and Postgres: it auto-detects
// the dialect by probing the live connection with dialect-specific SQL, which
// is robust for any *sql.DB regardless of how the driver was registered.
func EnsureCoreSeedMarker(ctx context.Context, database *sql.DB, name string, version int) (SeedResult, error) {
	dialect, err := dialectFromDB(ctx, database)
	if err != nil {
		return SeedResult{}, fmt.Errorf("EnsureCoreSeedMarker: detect dialect: %w", err)
	}

	key := fmt.Sprintf("seed.core.%s", name)
	value, err := json.Marshal(map[string]any{"name": name, "version": version})
	if err != nil {
		return SeedResult{}, err
	}

	// Use a single-statement, truly atomic upsert that avoids the TOCTOU race
	// window of the old INSERT … WHERE NOT EXISTS approach.
	//
	// SQLite:   INSERT OR IGNORE deduplicates on the unique key column in one
	//           atomic operation; no separate read needed.
	// Postgres: INSERT … ON CONFLICT DO NOTHING is the equivalent single-
	//           statement approach guaranteed by the engine.
	//
	// RowsAffected() == 1 means the row was newly inserted ("seeded"); 0 means
	// it already existed ("noop").
	q, err := seedInsertSQL(dialect)
	if err != nil {
		return SeedResult{}, err
	}

	result, err := database.ExecContext(ctx, q, key, string(value))
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

// seedInsertSQL returns the single-statement atomic INSERT with the correct
// placeholder style for the given dialect.
//
// Both variants rely on the UNIQUE constraint on app_settings.key to make the
// operation safe under concurrent callers without any read-then-write gap.
func seedInsertSQL(dialect Dialect) (string, error) {
	switch dialect {
	case DialectSQLite:
		// INSERT OR IGNORE is a single atomic operation in SQLite; it silently
		// skips the insert when a row with the same key already exists.
		return `INSERT OR IGNORE INTO app_settings (key, value, updated_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)`, nil
	case DialectPostgres:
		// ON CONFLICT DO NOTHING is the Postgres equivalent single-statement
		// atomic guard; pgx/stdlib requires $N positional placeholders.
		return `INSERT INTO app_settings (key, value, updated_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (key) DO NOTHING`, nil
	default:
		return "", fmt.Errorf("seedInsertSQL: unsupported dialect %q", dialect)
	}
}

// dialectFromDB probes the live connection with dialect-specific SQL to infer
// the Dialect without relying on driver names, registered driver instances, or
// type-name string matching.
//
// Detection order:
//  1. SELECT sqlite_version() — succeeds only on SQLite.
//  2. SELECT version()        — succeeds on Postgres (and most other SQL engines).
//
// Using a live probe means the result reflects the actual database in use,
// regardless of driver wrappers, registration aliases, or custom sql.Open calls.
// The dialect is never surfaced to callers; it is an internal implementation
// detail of EnsureCoreSeedMarker.
func dialectFromDB(ctx context.Context, database *sql.DB) (Dialect, error) {
	// Probe for SQLite first: sqlite_version() is a SQLite-only function.
	var discard string
	err := database.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&discard)
	if err == nil {
		return DialectSQLite, nil
	}

	// Probe for Postgres: version() is available in Postgres.
	err = database.QueryRowContext(ctx, "SELECT version()").Scan(&discard)
	if err == nil {
		return DialectPostgres, nil
	}

	return "", fmt.Errorf("dialectFromDB: unable to identify dialect via probe queries")
}
