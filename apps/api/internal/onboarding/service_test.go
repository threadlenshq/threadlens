package onboarding_test

// service_test.go specifies the expected behaviour of the onboarding Service
// before the implementation exists. All tests in this file are intentionally
// failing until onboarding.NewService (and the Service type) are implemented.
//
// Design constraints kept here:
//   - IsComplete reads the "onboarding.complete" key from the settings repository.
//   - GetStatus reports completion, enabled state, and the effective env file path.
//   - Save writes selected config values to the env file and marks onboarding complete.
//   - Reset clears the completion key (and optionally removes managed env values).
//   - The service refuses to save when onboarding is disabled.
//   - The service refuses to save when required config is missing in Docker mode.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// newTestService creates a Service wired to an in-memory DB and the supplied
// Config. The caller may further customise the Config before passing it in.
func newTestService(t *testing.T, cfg onboarding.Config) *onboarding.Service {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)
	svc, err := onboarding.NewService(cfg, repo)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

// tempEnvFile writes initial content to a temp .env file and returns its path.
func tempEnvFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// ── 1. IsComplete ─────────────────────────────────────────────────────────────

func TestIsComplete_FalseWhenKeyAbsent(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey}
	svc := newTestService(t, cfg)

	ok, err := svc.IsComplete(context.Background())
	if err != nil {
		t.Fatalf("IsComplete: %v", err)
	}
	if ok {
		t.Error("want IsComplete=false when key is absent, got true")
	}
}

func TestIsComplete_TrueAfterKeySet(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)

	// Pre-seed the completion key directly via the repository.
	if err := repo.Set(context.Background(), completionKey, "true"); err != nil {
		t.Fatal(err)
	}

	svc, err := onboarding.NewService(onboarding.Config{CompletionKey: completionKey}, repo)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	ok, err := svc.IsComplete(context.Background())
	if err != nil {
		t.Fatalf("IsComplete: %v", err)
	}
	if !ok {
		t.Error("want IsComplete=true after key is set, got false")
	}
}

// ── 2. GetStatus ──────────────────────────────────────────────────────────────

func TestGetStatus_ReportsNotCompleteByDefault(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey}
	svc := newTestService(t, cfg)

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.Complete {
		t.Error("want Complete=false by default, got true")
	}
}

func TestGetStatus_ReportsEnabled(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, Disabled: false}
	svc := newTestService(t, cfg)

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if !status.Enabled {
		t.Error("want Enabled=true when Disabled=false, got false")
	}
}

func TestGetStatus_ReportsDisabled(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, Disabled: true}
	svc := newTestService(t, cfg)

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.Enabled {
		t.Error("want Enabled=false when Disabled=true, got true")
	}
}

func TestGetStatus_ExposesEnvFilePath(t *testing.T) {
	envPath := "/custom/.env"
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.EnvFilePath != envPath {
		t.Errorf("EnvFilePath = %q; want %q", status.EnvFilePath, envPath)
	}
}

func TestGetStatus_CompleteAfterSave(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	if err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": "sk-test"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if !status.Complete {
		t.Error("want Complete=true after Save, got false")
	}
}

// ── 3. Save ───────────────────────────────────────────────────────────────────

func TestSave_WritesValuesToEnvFileInDockerMode(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": "sk-ant-test"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), "ANTHROPIC_API_KEY=sk-ant-test") {
		t.Errorf("env file does not contain expected key/value:\n%s", string(data))
	}
}

func TestSave_MarksOnboardingComplete(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	if err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": "sk-test"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ok, err := svc.IsComplete(context.Background())
	if err != nil {
		t.Fatalf("IsComplete: %v", err)
	}
	if !ok {
		t.Error("want IsComplete=true after Save, got false")
	}
}

func TestSave_InNativeModeDoesNotRequireEnvFile(t *testing.T) {
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		DockerMode:    false,
		// EnvFilePath intentionally empty
	}
	svc := newTestService(t, cfg)

	// In native mode, Save should not attempt to write an env file.
	err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": "sk-native"})
	if err != nil {
		t.Fatalf("Save in native mode: %v", err)
	}

	// Save must still mark onboarding complete even in native mode.
	ok, err := svc.IsComplete(context.Background())
	if err != nil {
		t.Fatalf("IsComplete after native Save: %v", err)
	}
	if !ok {
		t.Error("want IsComplete=true after Save in native mode, got false")
	}
}

func TestSave_ReturnsErrorForEmptyValueInDockerMode(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	// A key present but with an empty string value should be rejected in Docker
	// mode — the user hasn't provided a meaningful API key.
	err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": ""})
	if err == nil {
		t.Fatal("want error when saving an empty-valued key in Docker mode, got nil")
	}
}

// ── 4. Reset ──────────────────────────────────────────────────────────────────

func TestReset_ClearsCompletionKey(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	// Save first to mark complete.
	if err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": "sk-test"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := svc.Reset(context.Background()); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	ok, err := svc.IsComplete(context.Background())
	if err != nil {
		t.Fatalf("IsComplete: %v", err)
	}
	if ok {
		t.Error("want IsComplete=false after Reset, got true")
	}
}

func TestReset_IsIdempotent(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey}
	svc := newTestService(t, cfg)

	// Reset on a service that has never been saved should not error.
	if err := svc.Reset(context.Background()); err != nil {
		t.Fatalf("Reset on fresh service: %v", err)
	}
	if err := svc.Reset(context.Background()); err != nil {
		t.Fatalf("second Reset: %v", err)
	}
}

// ── 5. Gating ─────────────────────────────────────────────────────────────────

func TestSave_ReturnsErrorWhenDisabled(t *testing.T) {
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		Disabled:      true,
	}
	svc := newTestService(t, cfg)

	err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": "sk-test"})
	if err == nil {
		t.Fatal("want error when onboarding is disabled, got nil")
	}
}

func TestSave_ReturnsErrorForEmptyValuesInDockerMode(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	// Passing an empty values map in Docker mode should be rejected because
	// at least one key is required to configure the environment.
	err := svc.Save(context.Background(), map[string]string{})
	if err == nil {
		t.Fatal("want error when saving empty values in Docker mode, got nil")
	}
}

func TestSave_ReturnsErrorForNilValuesInDockerMode(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	err := svc.Save(context.Background(), nil)
	if err == nil {
		t.Fatal("want error when saving nil values in Docker mode, got nil")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// contains is a thin wrapper so tests don't need to import strings directly.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
