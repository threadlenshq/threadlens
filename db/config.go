package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
//  1. Normalise SCOUT_DB_DIALECT (trim whitespace, lowercase).
//  2. If dialect == "postgres" → postgres (requires DATABASE_URL).
//  3. If dialect == "sqlite"   → sqlite (use SCOUT_DB_PATH or default path).
//  4. If dialect == ""         → infer: postgres when DATABASE_URL is set,
//     otherwise sqlite.
//  5. Any other dialect value  → error.
func LoadConfigFromEnv() (Config, error) {
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	dialect := Dialect(strings.TrimSpace(strings.ToLower(os.Getenv("SCOUT_DB_DIALECT"))))
	dbPath := strings.TrimSpace(os.Getenv("SCOUT_DB_PATH"))

	switch dialect {
	case DialectPostgres:
		if dbURL == "" {
			return Config{}, errors.New("DATABASE_URL is required when SCOUT_DB_DIALECT=postgres")
		}
		return Config{Dialect: DialectPostgres, DatabaseURL: dbURL}, nil

	case DialectSQLite:
		// falls through to SQLite path resolution below

	case "":
		// No explicit dialect — infer from DATABASE_URL.
		if dbURL != "" {
			return Config{Dialect: DialectPostgres, DatabaseURL: dbURL}, nil
		}
		// Otherwise default to SQLite.

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

	return Config{Dialect: DialectSQLite, SQLitePath: filepath.Clean(dbPath)}, nil
}

// ResolveDefaultSQLitePath returns the default SQLite database path for the
// open-core checkout by walking up from the current working directory to find
// the open-core root (identified by go.work), then returning <root>/scout.db.
//
// If the root cannot be located, it falls back to <cwd>/scout.db rather than
// returning an error.
func ResolveDefaultSQLitePath() (string, error) {
	cwd, err := cwdResolver()
	if err != nil {
		return "", err
	}
	return resolveDefaultSQLitePathFrom(cwd), nil
}

// resolveDefaultSQLitePathFrom walks up the directory tree from start until it
// finds a directory that contains go.work (which precisely identifies the
// open-core workspace root) and returns the path to scout.db inside that root.
//
// go.work is used as the sole marker because it is unique to the open-core
// checkout root and is not present in any ancestor repository directory,
// preventing false positives on shared package.json ancestors.
//
// If no root is found, it falls back to <start>/scout.db.
func resolveDefaultSQLitePathFrom(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return filepath.Join(dir, "scout.db")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding a marker; fall back.
			return filepath.Join(start, "scout.db")
		}
		dir = parent
	}
}
