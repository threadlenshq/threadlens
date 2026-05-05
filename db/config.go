package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Dialect identifies the database backend.
type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

// Config holds the resolved database configuration.
type Config struct {
	Dialect     Dialect
	SQLitePath  string // populated when Dialect == DialectSQLite
	DatabaseURL string // populated when Dialect == DialectPostgres
}

// cwdResolver is a package-level injectable hook for obtaining the current
// working directory. Tests may replace it to avoid os.Chdir mutations.
var cwdResolver = os.Getwd

// LoadConfigFromEnv reads configuration from environment variables.
//
// Resolution order:
//  1. If DATABASE_URL is set → postgres dialect.
//  2. If SCOUT_DB_DIALECT == "postgres" → postgres dialect (requires DATABASE_URL).
//  3. Otherwise → sqlite dialect; path from SCOUT_DB_PATH or the default
//     open-core path resolved from the process working directory.
func LoadConfigFromEnv() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	dialect := Dialect(os.Getenv("SCOUT_DB_DIALECT"))
	dbPath := os.Getenv("SCOUT_DB_PATH")

	// Postgres wins when DATABASE_URL is present.
	if dbURL != "" {
		return Config{Dialect: DialectPostgres, DatabaseURL: dbURL}, nil
	}

	// Validate explicit dialect value.
	switch dialect {
	case "", DialectSQLite:
		// ok — falls through to sqlite path resolution below
	case DialectPostgres:
		return Config{}, errors.New("DATABASE_URL is required when SCOUT_DB_DIALECT=postgres")
	default:
		return Config{}, fmt.Errorf("unsupported SCOUT_DB_DIALECT %q: must be %q or %q", dialect, DialectSQLite, DialectPostgres)
	}

	// SQLite path: explicit override or auto-resolved default.
	if dbPath == "" {
		resolved, err := ResolveDefaultSQLitePath()
		if err != nil {
			return Config{}, err
		}
		dbPath = resolved
	}

	return Config{Dialect: DialectSQLite, SQLitePath: dbPath}, nil
}

// ResolveDefaultSQLitePath returns the default SQLite database path for the
// open-core checkout by walking up from the current working directory to find
// the workspace root (identified by a marker file such as go.work or
// package.json), then returning <root>/scout.db.
//
// If the workspace root cannot be located, it falls back to <cwd>/scout.db
// rather than returning an error.
func ResolveDefaultSQLitePath() (string, error) {
	cwd, err := cwdResolver()
	if err != nil {
		return "", err
	}
	return resolveDefaultSQLitePathFrom(cwd), nil
}

// openCoreRootMarkers are filenames whose presence identifies the open-core
// workspace root. go.work is used in synthetic test fixtures; package.json is
// present in the real open-core checkout.
var openCoreRootMarkers = []string{"go.work", "package.json"}

// resolveDefaultSQLitePathFrom walks up the directory tree from start until it
// finds a directory that contains one of the openCoreRootMarkers (indicating
// the open-core workspace root) and returns the path to scout.db inside that
// root. If no root is found, it falls back to <start>/scout.db.
func resolveDefaultSQLitePathFrom(start string) string {
	dir := start
	for {
		for _, marker := range openCoreRootMarkers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return filepath.Join(dir, "scout.db")
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding a marker; fall back.
			return filepath.Join(start, "scout.db")
		}
		dir = parent
	}
}
