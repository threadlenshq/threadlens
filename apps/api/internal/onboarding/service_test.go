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
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// newTestService creates a Service wired to an in-memory DB and the supplied
// Config. projectRepo is passed as nil; the service must accept a nil project
// repository for tests that do not exercise project-count behaviour.
func newTestService(t *testing.T, cfg onboarding.Config) *onboarding.Service {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)
	svc, err := onboarding.NewService(cfg, repo, nil)
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

// ── 1. GetStatus – new install ────────────────────────────────────────────────

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
	if status.CurrentRequiredStep != onboarding.RequiredStepWelcome {
		t.Errorf("CurrentRequiredStep = %q; want %q",
			status.CurrentRequiredStep, onboarding.RequiredStepWelcome)
	}
	if status.RequiredSetupComplete {
		t.Error("want RequiredSetupComplete=false on a fresh install, got true")
	}
}

// ── 2. GetStatus – disabled ───────────────────────────────────────────────────

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

// ── 3. GetStatus – legacy migration ──────────────────────────────────────────

func TestGetStatus_LegacyCompleteMigratesToExploration(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)

	// Simulate a legacy install: the old completion key is set but no v1 state
	// exists in the repository.
	if err := repo.Set(context.Background(), completionKey, "true"); err != nil {
		t.Fatal(err)
	}

	svc, err := onboarding.NewService(
		onboarding.Config{CompletionKey: completionKey, StateKey: stateKey},
		repo, nil,
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
	if !status.RequiredSetupComplete {
		t.Error("want RequiredSetupComplete=true after legacy migration, got false")
	}
}

// ── 4. SaveRequiredStep ───────────────────────────────────────────────────────

func TestSaveRequiredStepPersistsResumePoint(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// Complete the ai_provider step with the chosen provider path.
	stepValues := map[string]string{"AI_PROVIDER": "anthropic"}
	if _, err := svc.SaveRequiredStep(context.Background(), onboarding.RequiredStepAIProvider, stepValues); err != nil {
		t.Fatalf("SaveRequiredStep: %v", err)
	}

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus after SaveRequiredStep: %v", err)
	}
	// After completing ai_provider the resume point must advance to the next step.
	if status.CurrentRequiredStep != onboarding.RequiredStepAppDatabase {
		t.Errorf("CurrentRequiredStep = %q; want %q",
			status.CurrentRequiredStep, onboarding.RequiredStepAppDatabase)
	}
	// The non-secret provider choice must be persisted in the context.
	if status.Context.AIProviderPath != "anthropic" {
		t.Errorf("Context.AIProviderPath = %q; want %q",
			status.Context.AIProviderPath, "anthropic")
	}
}

// ── 5. Save (final required setup) ───────────────────────────────────────────

func TestSaveFinalRequiredSetupWritesEnvAndDoesNotStoreSecretInProgress(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		StateKey:      stateKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	db := testhelpers.OpenTestDB(t)
	repo := settings.NewRepository(db)
	svc, err := onboarding.NewService(cfg, repo, nil)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	secrets := map[string]string{
		"AI_PROVIDER":       "anthropic",
		"ANTHROPIC_API_KEY": "sk-ant-fake-secret",
	}
	if err := svc.Save(context.Background(), secrets); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The env file must contain the secret.
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "ANTHROPIC_API_KEY=sk-ant-fake-secret") {
		t.Errorf("env file does not contain expected key/value:\n%s", string(data))
	}

	// The v1 progress JSON stored in the settings repository must NOT contain
	// the raw secret value — secrets must only be written to the env file.
	rawProgress, found, getErr := repo.Get(context.Background(), stateKey)
	if getErr != nil {
		t.Fatalf("reading progress from repo: %v", getErr)
	}
	if found && strings.Contains(rawProgress, "sk-ant-fake-secret") {
		t.Error("progress JSON must not contain the raw secret; secrets belong only in the env file")
	}

	// Phase must advance to exploration after a successful final Save.
	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus after Save: %v", err)
	}
	if status.Phase != onboarding.PhaseExploration && status.Phase != onboarding.PhaseComplete {
		t.Errorf("Phase = %q; want exploration or complete after final Save", status.Phase)
	}
}

// ── 6. UpdateExploration ─────────────────────────────────────────────────────

func TestUpdateExplorationDismissMarksPendingSkipped(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// Advance through required setup so the service is in exploration phase.
	if err := svc.Save(context.Background(), map[string]string{}); err != nil {
		t.Fatalf("Save (native mode, advance to exploration): %v", err)
	}

	// Dismiss the exploration checklist via the typed update struct.
	update := onboarding.ExplorationUpdate{Dismiss: true}
	if _, err := svc.UpdateExploration(context.Background(), update); err != nil {
		t.Fatalf("UpdateExploration(dismiss=true): %v", err)
	}

	status, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus after UpdateExploration: %v", err)
	}
	// After dismissal, all pending items must be skipped and the exploration
	// must be reported as complete.
	if !status.ExplorationComplete {
		t.Error("want ExplorationComplete=true after dismiss, got false")
	}
	if status.Phase != onboarding.PhaseComplete {
		t.Errorf("Phase = %q; want %q after dismiss", status.Phase, onboarding.PhaseComplete)
	}
}

// ── 7. Reset ─────────────────────────────────────────────────────────────────

func TestResetProgressClearsV1AndLegacyState(t *testing.T) {
	envPath := tempEnvFile(t, "")
	cfg := onboarding.Config{
		CompletionKey: completionKey,
		StateKey:      stateKey,
		DockerMode:    true,
		EnvFilePath:   envPath,
	}
	svc := newTestService(t, cfg)

	// Save to write both the legacy completion flag and v1 progress.
	if err := svc.Save(context.Background(), map[string]string{"ANTHROPIC_API_KEY": "sk-test"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify complete before reset.
	ok, err := svc.IsComplete(context.Background())
	if err != nil {
		t.Fatalf("IsComplete before Reset: %v", err)
	}
	if !ok {
		t.Fatal("expected IsComplete=true before Reset")
	}

	// Reset with the typed constant for "clear everything".
	if err := svc.Reset(context.Background(), onboarding.ResetModeProgress); err != nil {
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

// ── helpers ───────────────────────────────────────────────────────────────────

// contains is a thin wrapper so tests don't need to import strings directly.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
