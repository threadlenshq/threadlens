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

func ResolveDefaultSQLitePath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	openCoreRoot, ok := findOpenCoreRoot(wd)
	if !ok {
		return filepath.Join(wd, "scout.db"), nil
	}
	return filepath.Join(openCoreRoot, "scout.db"), nil
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
