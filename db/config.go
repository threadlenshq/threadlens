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
		cwd, err := os.Getwd()
		if err != nil {
			return Config{}, err
		}
		resolved, err := ResolveDefaultSQLitePathFrom(cwd)
		if err != nil {
			return Config{}, err
		}
		dbPath = resolved
	}

	return Config{Dialect: DialectSQLite, SQLitePath: dbPath}, nil
}

// openCoreRootMarkers are filenames whose presence identifies the open-core
// workspace root. go.work is used in synthetic test fixtures; package.json is
// present in the real open-core checkout.
var openCoreRootMarkers = []string{"go.work", "package.json"}

// ResolveDefaultSQLitePathFrom walks up the directory tree from start until it
// finds a directory that contains one of the openCoreRootMarkers (indicating
// the open-core workspace root) and returns the path to scout.db inside that
// root.
//
// The start parameter makes the resolution testable without changing the global
// working directory.
func ResolveDefaultSQLitePathFrom(start string) (string, error) {
	dir := start
	for {
		for _, marker := range openCoreRootMarkers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return filepath.Join(dir, "scout.db"), nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not locate open-core root (no workspace marker found)")
		}
		dir = parent
	}
}
