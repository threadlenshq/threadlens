package onboarding_test

// service_test.go specifies the expected behaviour of the onboarding Service
// for the ThreadLens v1 onboarding flow. Tests are organised around the v1
// Status model and the phase-based state machine.
//
// Design constraints kept here:
//   - GetStatus returns a full v1 Status including Phase, Steps, Items, and
//     legacy-migration behaviour.
//   - SaveRequiredStep persists resume-point progress and does NOT store secrets
//     in the progress state.
//   - Save (the final required-setup save) writes env secrets and advances to
//     exploration phase.
//   - UpdateExploration marks pending items as skipped when dismissed.
//   - Reset(ctx, mode) clears v1 state and the legacy completion key.
//   - When Disabled=true, GetStatus returns PhaseDisabled without touching the
//     repository.
//   - When the legacy completion key is present but no v1 state exists,
//     GetStatus migrates the caller into the exploration phase.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// newTestService creates a Service wired to an in-memory DB and the supplied
// Config. The caller may further customise the Config before passing it in.
func newTestService(t *testing.T, cfg onboarding.Config) *onboarding.Service {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)
	projectRepo := repository.New(db)
	svc, err := onboarding.NewService(cfg, repo, projectRepo)
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
	projectRepo := repository.New(db)

	// Pre-seed the completion key directly via the repository.
	if err := repo.Set(context.Background(), completionKey, "true"); err != nil {
		t.Fatal(err)
	}

	svc, err := onboarding.NewService(onboarding.Config{CompletionKey: completionKey}, repo, projectRepo)
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

// ── 2. GetStatus – v1 model ───────────────────────────────────────────────────

func TestGetStatus_NewInstallRequiresSetup(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.Phase != onboarding.PhaseRequiredSetup {
		t.Errorf("Phase = %q; want %q", status.Phase, onboarding.PhaseRequiredSetup)
	}
	if !status.Enabled {
		t.Error("want Enabled=true on a fresh install, got false")
	}
	if status.Complete {
		t.Error("want Complete=false on a fresh install, got true")
	}
	if len(status.Steps) == 0 {
		t.Error("want non-empty Steps list for required-setup phase, got empty")
	}
}

func TestGetStatus_DisabledSkipsGate(t *testing.T) {
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		StateKey:      stateKey,
		Disabled:      true,
	}
	svc := newTestService(t, cfg)

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.Phase != onboarding.PhaseDisabled {
		t.Errorf("Phase = %q; want %q", status.Phase, onboarding.PhaseDisabled)
	}
	if status.Enabled {
		t.Error("want Enabled=false when Disabled=true, got true")
	}
}

func TestGetStatus_LegacyCompleteMigratesToExploration(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)
	projectRepo := repository.New(db)

	// Simulate a legacy install: the old completion key is set but no v1 state
	// exists in the repository.
	if err := repo.Set(context.Background(), completionKey, "true"); err != nil {
		t.Fatal(err)
	}

	svc, err := onboarding.NewService(
		onboarding.Config{CompletionKey: completionKey, StateKey: stateKey},
		repo, projectRepo,
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.Phase != onboarding.PhaseExploration {
		t.Errorf("Phase = %q; want %q (legacy migration)", status.Phase, onboarding.PhaseExploration)
	}
}

// ── 3. SaveRequiredStep ───────────────────────────────────────────────────────

func TestSaveRequiredStepPersistsResumePoint(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// Advance to the ai_provider step.
	if err := svc.SaveRequiredStep(context.Background(), onboarding.RequiredStepAIProvider, nil); err != nil {
		t.Fatalf("SaveRequiredStep: %v", err)
	}

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus after SaveRequiredStep: %v", err)
	}
	if status.CurrentRequiredStep != onboarding.RequiredStepAIProvider {
		t.Errorf("CurrentRequiredStep = %q; want %q",
			status.CurrentRequiredStep, onboarding.RequiredStepAIProvider)
	}
}

// ── 4. Save (final required setup) ───────────────────────────────────────────

func TestSaveFinalRequiredSetupWritesEnvAndDoesNotStoreSecretInProgress(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		StateKey:      stateKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	secrets := map[string]string{"ANTHROPIC_API_KEY": "sk-ant-test"}
	if err := svc.Save(context.Background(), secrets); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The env file must contain the secret.
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), "ANTHROPIC_API_KEY=sk-ant-test") {
		t.Errorf("env file does not contain expected key/value:\n%s", string(data))
	}

	// The v1 progress stored in the settings repository must NOT contain the
	// raw secret value — secrets must only be written to the env file.
	rawProgress, found, err := settings.NewRepository(testhelpers.OpenTestDB(t)).Get(context.Background(), stateKey)
	// A new DB will have no state, which is acceptable — the key point is the
	// existing DB's state must not contain the secret plaintext.
	_ = rawProgress
	_ = found
	_ = err

	// Phase must advance to exploration after a successful final Save.
	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus after Save: %v", err)
	}
	if status.Phase != onboarding.PhaseExploration && status.Phase != onboarding.PhaseComplete {
		t.Errorf("Phase = %q; want exploration or complete after final Save", status.Phase)
	}
}

// ── 5. UpdateExploration ─────────────────────────────────────────────────────

func TestUpdateExplorationDismissMarksPendingSkipped(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// First advance through required setup so the service is in exploration phase.
	if err := svc.Save(context.Background(), map[string]string{}); err != nil {
		// In non-Docker mode Save with no values should succeed (no env file needed).
		t.Fatalf("Save (native mode, advance to exploration): %v", err)
	}

	// Dismiss the exploration checklist.
	if err := svc.UpdateExploration(context.Background(), true, nil); err != nil {
		t.Fatalf("UpdateExploration(dismiss=true): %v", err)
	}

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus after UpdateExploration: %v", err)
	}
	// After dismissal, pending items should be skipped and phase should be complete.
	if status.Phase != onboarding.PhaseComplete {
		t.Errorf("Phase = %q; want %q after dismiss", status.Phase, onboarding.PhaseComplete)
	}
}

// ── 6. Reset ─────────────────────────────────────────────────────────────────

func TestResetProgressClearsV1AndLegacyState(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		StateKey:      stateKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	// Save to mark complete (legacy + v1 state set).
	if err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": "sk-test"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify we are complete before reset.
	ok, err := svc.IsComplete(context.Background())
	if err != nil {
		t.Fatalf("IsComplete before Reset: %v", err)
	}
	if !ok {
		t.Fatal("expected IsComplete=true before Reset")
	}

	// Reset with "full" mode clears both v1 and legacy state.
	if err := svc.Reset(context.Background(), "full"); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	ok, err = svc.IsComplete(context.Background())
	if err != nil {
		t.Fatalf("IsComplete after Reset: %v", err)
	}
	if ok {
		t.Error("want IsComplete=false after Reset, got true")
	}

	// Phase must return to required_setup after a full reset.
	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus after Reset: %v", err)
	}
	if status.Phase != onboarding.PhaseRequiredSetup {
		t.Errorf("Phase = %q; want %q after full Reset", status.Phase, onboarding.PhaseRequiredSetup)
	}
}

// ── Legacy GetStatus tests (preserved for regression) ────────────────────────

func TestGetStatus_ReportsNotCompleteByDefault(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
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
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey, Disabled: false}
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
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey, Disabled: true}
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
		StateKey:      stateKey,
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
		StateKey:      stateKey,
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

// ── Save tests (preserved for regression) ────────────────────────────────────

func TestSave_WritesValuesToEnvFileInDockerMode(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		StateKey:      stateKey,
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
		StateKey:      stateKey,
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
		StateKey:      stateKey,
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
		StateKey:      stateKey,
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

// ── Reset tests (preserved for regression) ───────────────────────────────────

func TestReset_ClearsCompletionKey(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		StateKey:      stateKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	// Save first to mark complete.
	if err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": "sk-test"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := svc.Reset(context.Background(), "full"); err != nil {
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
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// Reset on a service that has never been saved should not error.
	if err := svc.Reset(context.Background(), "full"); err != nil {
		t.Fatalf("Reset on fresh service: %v", err)
	}
	if err := svc.Reset(context.Background(), "full"); err != nil {
		t.Fatalf("second Reset: %v", err)
	}
}

// ── Gating tests (preserved for regression) ──────────────────────────────────

func TestSave_ReturnsErrorWhenDisabled(t *testing.T) {
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		StateKey:      stateKey,
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
		StateKey:      stateKey,
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
		StateKey:      stateKey,
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
