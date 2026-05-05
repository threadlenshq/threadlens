package db

import (
	"os"
	"path/filepath"
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

// buildOpenCoreFixture creates a temp directory that mimics the open-core
// subtree layout used by ResolveDefaultSQLitePathFrom:
//
//	<root>/           ← open-core root (has go.work or open-core marker)
//	  apps/
//	    api/          ← a nested subdirectory to start the walk from
//
// It returns the path of the nested subdirectory so tests can use it as the
// start argument without touching the real checkout or the process-wide cwd.
func buildOpenCoreFixture(t *testing.T) (startDir, openCoreRoot string) {
	t.Helper()
	root := t.TempDir()

	// Marker file that ResolveDefaultSQLitePathFrom will use to identify the
	// open-core root (mirrors what the production implementation will look for).
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(root, "apps", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	return nested, root
}

func TestResolveDefaultSQLitePathWorksFromRepoSubdirectories(t *testing.T) {
	t.Parallel()

	startDir, openCoreRoot := buildOpenCoreFixture(t)

	path, err := ResolveDefaultSQLitePathFrom(startDir)
	if err != nil {
		t.Fatalf("ResolveDefaultSQLitePathFrom(%q) error = %v", startDir, err)
	}
	want := filepath.Join(openCoreRoot, "scout.db")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}
