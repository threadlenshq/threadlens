package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
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

// dialectFromDB infers the Dialect by comparing the driver instance returned
// by database.Driver() against the instances registered under the known driver
// names ("sqlite" and "pgx"). This avoids fragile type-name substring matching
// and works correctly even when drivers are wrapped (as long as they are
// registered under one of the canonical names).
//
// The dialect is never surfaced to callers; it is an internal implementation
// detail of EnsureCoreSeedMarker.
func dialectFromDB(database *sql.DB) (Dialect, error) {
	drv := database.Driver()

	knownDrivers := map[string]Dialect{
		"sqlite": DialectSQLite,
		"pgx":    DialectPostgres,
	}

	for name, dialect := range knownDrivers {
		// sql.Open registers and caches the driver instance; comparing the
		// interface values (which hold the same concrete pointer) is reliable.
		if registered := driverByName(name); registered != nil && registered == drv {
			return dialect, nil
		}
	}

	return "", fmt.Errorf("dialectFromDB: unrecognised driver (not registered under a known name)")
}

// driverByName returns the driver.Driver registered under name, or nil if the
// name is not in sql.Drivers(). It opens a throwaway *sql.DB solely to obtain
// the cached Driver() value; no actual connection is made.
func driverByName(name string) driver.Driver {
	for _, n := range sql.Drivers() {
		if n == name {
			db, err := sql.Open(name, "")
			if err != nil {
				return nil
			}
			d := db.Driver()
			_ = db.Close()
			return d
		}
	}
	return nil
}
