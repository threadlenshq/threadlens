package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildOpenCoreFixture creates a temp directory that mimics the open-core
// subtree layout:
//
//	<root>/           ← open-core root (has go.work marker)
//	  apps/
//	    api/          ← a nested subdirectory
//
// It returns (nestedDir, rootDir) so callers can inject nestedDir as the fake
// cwd without mutating the real process cwd.
func buildOpenCoreFixture(t *testing.T) (nestedDir, openCoreRoot string) {
	t.Helper()
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(root, "apps", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	return nested, root
}

// withFakeCwd temporarily replaces the package-level cwdResolver so that
// ResolveDefaultSQLitePath() returns the supplied directory as the cwd.
// It restores the original resolver via t.Cleanup.
func withFakeCwd(t *testing.T, dir string) {
	t.Helper()
	orig := cwdResolver
	cwdResolver = func() (string, error) { return dir, nil }
	t.Cleanup(func() { cwdResolver = orig })
}

func TestLoadConfigDefaultsToOpenCoreSQLitePath(t *testing.T) {
	nestedDir, openCoreRoot := buildOpenCoreFixture(t)
	withFakeCwd(t, nestedDir)

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
	want := filepath.Join(openCoreRoot, "scout.db")
	if cfg.SQLitePath != want {
		t.Fatalf("SQLitePath = %q, want %q", cfg.SQLitePath, want)
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

func TestResolveDefaultSQLitePathFindsRepoRoot(t *testing.T) {
	t.Parallel()

	nestedDir, openCoreRoot := buildOpenCoreFixture(t)
	withFakeCwd(t, nestedDir)

	path, err := ResolveDefaultSQLitePath()
	if err != nil {
		t.Fatalf("ResolveDefaultSQLitePath() error = %v", err)
	}
	want := filepath.Join(openCoreRoot, "scout.db")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestResolveDefaultSQLitePathFallsBackToCwd(t *testing.T) {
	t.Parallel()

	// A temp dir with no marker files — root cannot be found.
	dir := t.TempDir()
	withFakeCwd(t, dir)

	path, err := ResolveDefaultSQLitePath()
	if err != nil {
		t.Fatalf("ResolveDefaultSQLitePath() error = %v (want fallback, not error)", err)
	}
	want := filepath.Join(dir, "scout.db")
	if path != want {
		t.Fatalf("path = %q, want fallback %q", path, want)
	}
}
