package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Dialect string

const (
	DialectSQLite   Dialect = "sqlite"
	DialectPostgres Dialect = "postgres"
)

type Config struct {
	Dialect     Dialect
	SQLitePath  string
	DatabaseURL string
}

func LoadConfigFromEnv() (Config, error) {
	dialect := strings.TrimSpace(strings.ToLower(os.Getenv("SCOUT_DB_DIALECT")))
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	sqlitePath := strings.TrimSpace(os.Getenv("SCOUT_DB_PATH"))

	if dialect == "" {
		if databaseURL != "" {
			dialect = string(DialectPostgres)
		} else {
			dialect = string(DialectSQLite)
		}
	}

	switch Dialect(dialect) {
	case DialectSQLite:
		if sqlitePath == "" {
			resolved, err := ResolveDefaultSQLitePath()
			if err != nil {
				return Config{}, err
			}
			sqlitePath = resolved
		}
		return Config{Dialect: DialectSQLite, SQLitePath: filepath.Clean(sqlitePath)}, nil
	case DialectPostgres:
		if databaseURL == "" {
			return Config{}, errors.New("DATABASE_URL is required when SCOUT_DB_DIALECT=postgres")
		}
		return Config{Dialect: DialectPostgres, DatabaseURL: databaseURL}, nil
	default:
		return Config{}, fmt.Errorf("unsupported SCOUT_DB_DIALECT %q", dialect)
	}
}

// cwdResolver is a package-level seam that tests can override to inject a
// fake working directory without mutating the real process cwd.
var cwdResolver = os.Getwd

// ResolveDefaultSQLitePath returns the default SQLite database path.
// It walks up from the current working directory looking for the open-core
// root (identified by apps/api/go.mod + package.json). When found it returns
// <open-core-root>/scout.db; otherwise it falls back to <cwd>/scout.db.
func ResolveDefaultSQLitePath() (string, error) {
	wd, err := cwdResolver()
	if err != nil {
		return "", err
	}
	return resolveDefaultSQLitePathFrom(wd), nil
}

// resolveDefaultSQLitePathFrom is the pure, testable core of the resolution
// logic. It accepts an explicit start directory so tests can inject any path
// without touching the process working directory.
func resolveDefaultSQLitePathFrom(start string) string {
	openCoreRoot, ok := findOpenCoreRoot(start)
	if !ok {
		return filepath.Join(start, "scout.db")
	}
	return filepath.Join(openCoreRoot, "scout.db")
}

func findOpenCoreRoot(start string) (string, bool) {
	current := filepath.Clean(start)
	for {
		if fileExists(filepath.Join(current, "apps", "api", "go.mod")) && fileExists(filepath.Join(current, "package.json")) {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
