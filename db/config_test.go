package db

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadConfigDefaultsToOpenCoreSQLitePath(t *testing.T) {
	t.Setenv("SCOUT_DB_DIALECT", "")
	t.Setenv("SCOUT_DB_PATH", "")
	t.Setenv("DATABASE_URL", "")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.Dialect != DialectSQLite {
		t.Fatalf("Dialect = %q, want %q", cfg.Dialect, DialectSQLite)
	}
	if !strings.HasSuffix(filepath.ToSlash(cfg.SQLitePath), "/open-core/scout.db") {
		t.Fatalf("SQLitePath = %q, want suffix /open-core/scout.db", cfg.SQLitePath)
	}
	if filepath.Base(cfg.SQLitePath) != "scout.db" {
		t.Fatalf("SQLitePath base = %q, want scout.db", filepath.Base(cfg.SQLitePath))
	}
}

func TestLoadConfigUsesSQLitePathOverride(t *testing.T) {
	t.Setenv("SCOUT_DB_DIALECT", "sqlite")
	t.Setenv("SCOUT_DB_PATH", filepath.Join(t.TempDir(), "custom.db"))
	t.Setenv("DATABASE_URL", "")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.Dialect != DialectSQLite {
		t.Fatalf("Dialect = %q", cfg.Dialect)
	}
	if !strings.HasSuffix(cfg.SQLitePath, "custom.db") {
		t.Fatalf("SQLitePath = %q, want custom.db suffix", cfg.SQLitePath)
	}
}

func TestLoadConfigUsesPostgresWhenDatabaseURLIsSet(t *testing.T) {
	t.Setenv("SCOUT_DB_DIALECT", "")
	t.Setenv("SCOUT_DB_PATH", "")
	t.Setenv("DATABASE_URL", "postgres://scout:secret@localhost:5432/scout_dev?sslmode=disable")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.Dialect != DialectPostgres {
		t.Fatalf("Dialect = %q, want %q", cfg.Dialect, DialectPostgres)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("DatabaseURL must be populated")
	}
}

func TestLoadConfigRejectsPostgresWithoutURL(t *testing.T) {
	t.Setenv("SCOUT_DB_DIALECT", "postgres")
	t.Setenv("SCOUT_DB_PATH", "")
	t.Setenv("DATABASE_URL", "")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatal("LoadConfigFromEnv() error = nil, want missing DATABASE_URL error")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("error = %v, want DATABASE_URL is required", err)
	}
}

func TestResolveDefaultSQLitePathWorksFromRepoSubdirectories(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoOpenCore := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	apiDir := filepath.Join(repoOpenCore, "apps", "api")
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	if err := os.Chdir(apiDir); err != nil {
		t.Fatal(err)
	}

	path, err := ResolveDefaultSQLitePath()
	if err != nil {
		t.Fatalf("ResolveDefaultSQLitePath() error = %v", err)
	}
	want := filepath.Join(repoOpenCore, "scout.db")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}
