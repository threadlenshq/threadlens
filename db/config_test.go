package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeOpenCoreFixture creates a minimal directory tree that mimics the
// open-core layout used by findOpenCoreRoot:
//
//	<root>/apps/api/go.mod
//	<root>/package.json
//
// It returns the root directory path.
func makeOpenCoreFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	apiDir := filepath.Join(root, "apps", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	for _, f := range []string{
		filepath.Join(apiDir, "go.mod"),
		filepath.Join(root, "package.json"),
	} {
		if err := os.WriteFile(f, []byte("{}"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", f, err)
		}
	}
	return root
}

// withCwdResolver temporarily overrides the package-level cwdResolver and
// restores it when the test finishes.
func withCwdResolver(t *testing.T, dir string) {
	t.Helper()
	orig := cwdResolver
	cwdResolver = func() (string, error) { return dir, nil }
	t.Cleanup(func() { cwdResolver = orig })
}

// ---------------------------------------------------------------------------
// resolveDefaultSQLitePathFrom (pure helper — no process state)
// ---------------------------------------------------------------------------

func TestResolveDefaultSQLitePathFrom_OpenCoreRoot(t *testing.T) {
	root := makeOpenCoreFixture(t)
	got := resolveDefaultSQLitePathFrom(root)
	want := filepath.Join(root, "scout.db")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolveDefaultSQLitePathFrom_SubdirectoryOfOpenCore(t *testing.T) {
	root := makeOpenCoreFixture(t)
	subDir := filepath.Join(root, "apps", "api")
	got := resolveDefaultSQLitePathFrom(subDir)
	want := filepath.Join(root, "scout.db")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolveDefaultSQLitePathFrom_FallbackWhenNoOpenCoreRoot(t *testing.T) {
	dir := t.TempDir() // plain dir, no open-core markers
	got := resolveDefaultSQLitePathFrom(dir)
	want := filepath.Join(dir, "scout.db")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// ResolveDefaultSQLitePath (uses cwdResolver seam)
// ---------------------------------------------------------------------------

func TestResolveDefaultSQLitePath_FromOpenCoreSubdir(t *testing.T) {
	root := makeOpenCoreFixture(t)
	subDir := filepath.Join(root, "apps", "api")
	withCwdResolver(t, subDir)

	got, err := ResolveDefaultSQLitePath()
	if err != nil {
		t.Fatalf("ResolveDefaultSQLitePath() error = %v", err)
	}
	want := filepath.Join(root, "scout.db")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolveDefaultSQLitePath_FallbackDir(t *testing.T) {
	dir := t.TempDir()
	withCwdResolver(t, dir)

	got, err := ResolveDefaultSQLitePath()
	if err != nil {
		t.Fatalf("ResolveDefaultSQLitePath() error = %v", err)
	}
	want := filepath.Join(dir, "scout.db")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// LoadConfigFromEnv
// ---------------------------------------------------------------------------

func TestLoadConfigDefaultsToSQLiteWithOpenCorePath(t *testing.T) {
	root := makeOpenCoreFixture(t)
	withCwdResolver(t, filepath.Join(root, "apps", "api"))

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
	want := filepath.Join(root, "scout.db")
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
