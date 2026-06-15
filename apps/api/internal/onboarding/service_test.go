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
//   - SaveRequiredStep rejects steps that are not the current resume point.
//   - Save (the final required-setup save) writes env secrets and advances to
//     exploration phase, but only after all required steps have been completed.
//   - Save rejects calls from a fresh install.
//   - UpdateExploration marks pending items as skipped when dismissed.
//   - Reset(ctx, mode) clears v1 state and the legacy completion key.
//   - When Disabled=true, GetStatus returns PhaseDisabled without touching the
//     repository.
//   - When the legacy completion key is present but no v1 state exists,
//     GetStatus migrates the caller into the exploration phase.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
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
	svc, err := onboarding.NewService(cfg, repo, nil, nil)
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

// completeRequiredSteps walks the service through every required step in order
// so that Save can be called. stepValues is merged into the values map for each
// call, which is the right place to inject AI_PROVIDER etc. for the relevant
// step.
func completeRequiredSteps(t *testing.T, svc *onboarding.Service, stepValues map[string]string) {
	t.Helper()
	ctx := context.Background()
	for _, step := range onboarding.RequiredSteps {
		vals := map[string]string{}
		for k, v := range stepValues {
			vals[k] = v
		}
		if _, err := svc.SaveRequiredStep(ctx, step, vals); err != nil {
			t.Fatalf("completeRequiredSteps: SaveRequiredStep(%q): %v", step, err)
		}
	}
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
		repo, nil, nil,
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

// ── 4. SaveRequiredStep – happy path ─────────────────────────────────────────

func TestSaveRequiredStepPersistsResumePoint(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// Complete the welcome step first (the linear flow starts there).
	if _, err := svc.SaveRequiredStep(context.Background(), onboarding.RequiredStepWelcome, nil); err != nil {
		t.Fatalf("SaveRequiredStep(welcome): %v", err)
	}

	// Now complete the ai_provider step with the chosen provider path.
	stepValues := map[string]string{"AI_PROVIDER": "anthropic"}
	if _, err := svc.SaveRequiredStep(context.Background(), onboarding.RequiredStepAIProvider, stepValues); err != nil {
		t.Fatalf("SaveRequiredStep(ai_provider): %v", err)
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

// ── 4b. SaveRequiredStep – out-of-order rejection ────────────────────────────

func TestSaveRequiredStep_RejectsOutOfOrder(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// On a fresh install the current step is welcome. Attempting to save
	// ai_provider (the second step) must be rejected.
	_, err := svc.SaveRequiredStep(
		context.Background(),
		onboarding.RequiredStepAIProvider,
		map[string]string{"AI_PROVIDER": "anthropic"},
	)
	if err == nil {
		t.Fatal("SaveRequiredStep(ai_provider) on fresh install: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not the current step") {
		t.Errorf("error %q does not mention 'not the current step'", err.Error())
	}

	// Skipping ahead to review must also be rejected.
	_, err = svc.SaveRequiredStep(context.Background(), onboarding.RequiredStepReview, nil)
	if err == nil {
		t.Fatal("SaveRequiredStep(review) on fresh install: expected error, got nil")
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
	svc, err := onboarding.NewService(cfg, repo, nil, nil)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	// Walk through all required steps before calling Save.
	completeRequiredSteps(t, svc, map[string]string{"AI_PROVIDER": "anthropic"})

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

// ── 5b. Save – rejected before required setup is complete ────────────────────

func TestSave_RejectsBeforeRequiredSetupComplete(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// Calling Save on a brand-new install (no steps completed) must fail.
	err := svc.Save(context.Background(), map[string]string{})
	if err == nil {
		t.Fatal("Save on fresh install: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "review step") {
		t.Errorf("error %q does not mention 'review step'", err.Error())
	}

	// Phase must remain at required_setup — Save must not have mutated state.
	status, statusErr := svc.GetStatus(context.Background())
	if statusErr != nil {
		t.Fatalf("GetStatus: %v", statusErr)
	}
	if status.Phase != onboarding.PhaseRequiredSetup {
		t.Errorf("Phase = %q after rejected Save; want %q", status.Phase, onboarding.PhaseRequiredSetup)
	}
}

// ── 6. UpdateExploration ─────────────────────────────────────────────────────

func TestUpdateExplorationDismissMarksPendingSkipped(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// Advance through all required steps, then call Save to enter exploration.
	completeRequiredSteps(t, svc, map[string]string{"AI_PROVIDER": "anthropic"})
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

	// Walk through all required steps then Save to write both the legacy
	// completion flag and v1 progress.
	completeRequiredSteps(t, svc, map[string]string{"AI_PROVIDER": "anthropic"})
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

// ── 8. Reopening exploration ──────────────────────────────────────────────────

// TestUpdateExploration_ReopenItemBecomesActive verifies that moving an already-
// terminal exploration item back to pending or blocked clears the completion
// state and returns the phase to exploration (active).
func TestUpdateExploration_ReopenItemBecomesActive(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	// Enter exploration phase.
	completeRequiredSteps(t, svc, map[string]string{"AI_PROVIDER": "anthropic"})
	if err := svc.Save(context.Background(), map[string]string{}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ctx := context.Background()

	// Complete every exploration item so the phase advances to complete.
	for _, item := range onboarding.ExplorationItems {
		_, err := svc.UpdateExploration(ctx, onboarding.ExplorationUpdate{
			Item:  item,
			State: onboarding.ItemStateCompleted,
		})
		if err != nil {
			t.Fatalf("UpdateExploration(complete %q): %v", item, err)
		}
	}

	status, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus after completing all items: %v", err)
	}
	if status.Phase != onboarding.PhaseComplete {
		t.Fatalf("Phase = %q before reopen; want complete", status.Phase)
	}

	// Reopen the first item back to pending — exploration should become active.
	_, err = svc.UpdateExploration(ctx, onboarding.ExplorationUpdate{
		Item:  onboarding.ExplorationItemStarterProject,
		State: onboarding.ItemStatePending,
	})
	if err != nil {
		t.Fatalf("UpdateExploration(reopen to pending): %v", err)
	}

	status, err = svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus after reopen: %v", err)
	}
	if status.Phase != onboarding.PhaseExploration {
		t.Errorf("Phase = %q after reopen; want %q", status.Phase, onboarding.PhaseExploration)
	}
	if status.ExplorationComplete {
		t.Error("want ExplorationComplete=false after reopening an item, got true")
	}
}

// ── 9. Stable CompletedAt ─────────────────────────────────────────────────────

// TestUpdateExploration_RepeatedDismissPreservesCompletedAt verifies that
// calling UpdateExploration with Dismiss=true a second time does not overwrite
// the original CompletedAt timestamp.
func TestUpdateExploration_RepeatedDismissPreservesCompletedAt(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	svc := newTestService(t, cfg)

	completeRequiredSteps(t, svc, map[string]string{"AI_PROVIDER": "anthropic"})
	if err := svc.Save(context.Background(), map[string]string{}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ctx := context.Background()

	// First dismiss.
	if _, err := svc.UpdateExploration(ctx, onboarding.ExplorationUpdate{Dismiss: true}); err != nil {
		t.Fatalf("first dismiss: %v", err)
	}

	// Capture the CompletedAt from the stored progress JSON.
	firstJSON := svc.DebugProgressJSONForTest(ctx)
	if firstJSON == "" {
		t.Fatal("no progress JSON found after first dismiss")
	}

	// Second dismiss — must not overwrite CompletedAt.
	if _, err := svc.UpdateExploration(ctx, onboarding.ExplorationUpdate{Dismiss: true}); err != nil {
		t.Fatalf("second dismiss: %v", err)
	}

	secondJSON := svc.DebugProgressJSONForTest(ctx)

	// Extract completedAt from the exploration block of both snapshots via
	// simple string search so the test does not depend on a separate JSON
	// unmarshal helper.
	extractExplorationCompletedAt := func(raw string) string {
		const explKey = `"exploration":`
		ei := strings.Index(raw, explKey)
		if ei == -1 {
			return ""
		}
		expl := raw[ei:]
		const key = `"completedAt":"`
		idx := strings.Index(expl, key)
		if idx == -1 {
			return ""
		}
		rest := expl[idx+len(key):]
		end := strings.Index(rest, `"`)
		if end == -1 {
			return ""
		}
		return rest[:end]
	}

	first := extractExplorationCompletedAt(firstJSON)
	second := extractExplorationCompletedAt(secondJSON)
	if first == "" {
		t.Fatal("CompletedAt not set after first dismiss")
	}
	if first != second {
		t.Errorf("CompletedAt changed on repeated dismiss: first=%q second=%q", first, second)
	}
}

// ── 10. Auto-assign model overrides on SaveRequiredStep(ai_provider) ─────────

func TestSaveRequiredStep_AIProvider_WritesOpencodeDefaults(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	db := testhelpers.OpenTestDB(t)
	settingsRepo := settings.NewRepository(db)
	projectRepo := repository.New(db)
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	modelSvc := services.NewModelService(projectRepo, entitlements.RuntimeModeSelfHosted, resolver)
	svc, err := onboarding.NewService(cfg, settingsRepo, projectRepo, modelSvc)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()

	// Complete welcome step first.
	if _, err := svc.SaveRequiredStep(ctx, onboarding.RequiredStepWelcome, nil); err != nil {
		t.Fatalf("SaveRequiredStep(welcome): %v", err)
	}

	// Save ai_provider with "opencode".
	if _, err := svc.SaveRequiredStep(ctx, onboarding.RequiredStepAIProvider, map[string]string{
		"AI_PROVIDER": "opencode",
	}); err != nil {
		t.Fatalf("SaveRequiredStep(ai_provider): %v", err)
	}

	// Verify all 7 model.<taskID> rows exist with opencode-go defaults.
	for _, task := range ai.Tasks {
		key := "model." + task.ID
		raw, ok, getErr := projectRepo.GetSetting(ctx, key)
		if getErr != nil {
			t.Fatalf("GetSetting(%q): %v", key, getErr)
		}
		if !ok {
			t.Errorf("model.%s row not found after SaveRequiredStep(opencode)", task.ID)
			continue
		}
		var obj map[string]string
		if jsonErr := json.Unmarshal([]byte(raw), &obj); jsonErr != nil {
			t.Errorf("model.%s value is not valid JSON: %v", task.ID, jsonErr)
			continue
		}
		expectedModel := task.DefaultByProvider["opencode"]
		if obj["modelId"] != expectedModel {
			t.Errorf("model.%s = %q; want %q", task.ID, obj["modelId"], expectedModel)
		}
	}

	// Verify ai_provider was stored.
	val, ok, getErr := projectRepo.GetSetting(ctx, "ai_provider")
	if getErr != nil {
		t.Fatalf("GetSetting(ai_provider): %v", getErr)
	}
	if !ok || val != "opencode" {
		t.Errorf("ai_provider = %q (ok=%v); want %q", val, ok, "opencode")
	}
}

func TestSaveRequiredStep_AIProvider_WritesClaudeCliDefaults(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	db := testhelpers.OpenTestDB(t)
	settingsRepo := settings.NewRepository(db)
	projectRepo := repository.New(db)
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	modelSvc := services.NewModelService(projectRepo, entitlements.RuntimeModeSelfHosted, resolver)
	svc, err := onboarding.NewService(cfg, settingsRepo, projectRepo, modelSvc)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()

	// Complete welcome step first.
	if _, err := svc.SaveRequiredStep(ctx, onboarding.RequiredStepWelcome, nil); err != nil {
		t.Fatalf("SaveRequiredStep(welcome): %v", err)
	}

	// Save ai_provider with "claude-cli".
	if _, err := svc.SaveRequiredStep(ctx, onboarding.RequiredStepAIProvider, map[string]string{
		"AI_PROVIDER": "claude-cli",
	}); err != nil {
		t.Fatalf("SaveRequiredStep(ai_provider): %v", err)
	}

	// Verify all 7 model.<taskID> rows exist with claude-cli defaults.
	for _, task := range ai.Tasks {
		key := "model." + task.ID
		raw, ok, getErr := projectRepo.GetSetting(ctx, key)
		if getErr != nil {
			t.Fatalf("GetSetting(%q): %v", key, getErr)
		}
		if !ok {
			t.Errorf("model.%s row not found after SaveRequiredStep(claude-cli)", task.ID)
			continue
		}
		var obj map[string]string
		if jsonErr := json.Unmarshal([]byte(raw), &obj); jsonErr != nil {
			t.Errorf("model.%s value is not valid JSON: %v", task.ID, jsonErr)
			continue
		}
		expectedModel := task.DefaultByProvider["claude-cli"]
		if obj["modelId"] != expectedModel {
			t.Errorf("model.%s = %q; want %q", task.ID, obj["modelId"], expectedModel)
		}
	}
}

func TestSaveRequiredStep_AIProvider_IdempotentOnResave(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	db := testhelpers.OpenTestDB(t)
	settingsRepo := settings.NewRepository(db)
	projectRepo := repository.New(db)
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	modelSvc := services.NewModelService(projectRepo, entitlements.RuntimeModeSelfHosted, resolver)
	svc, err := onboarding.NewService(cfg, settingsRepo, projectRepo, modelSvc)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()

	// Walk through welcome → ai_provider with "claude-cli".
	if _, err := svc.SaveRequiredStep(ctx, onboarding.RequiredStepWelcome, nil); err != nil {
		t.Fatalf("SaveRequiredStep(welcome): %v", err)
	}
	if _, err := svc.SaveRequiredStep(ctx, onboarding.RequiredStepAIProvider, map[string]string{
		"AI_PROVIDER": "claude-cli",
	}); err != nil {
		t.Fatalf("SaveRequiredStep(ai_provider, claude-cli): %v", err)
	}

	// Capture the model rows after first save.
	firstValues := make(map[string]string, len(ai.Tasks))
	for _, task := range ai.Tasks {
		raw, ok, _ := projectRepo.GetSetting(ctx, "model."+task.ID)
		if ok {
			firstValues[task.ID] = raw
		}
	}

	// Re-save the same provider (idempotent).
	if _, err := svc.SaveRequiredStep(ctx, onboarding.RequiredStepAIProvider, map[string]string{
		"AI_PROVIDER": "claude-cli",
	}); err != nil {
		t.Fatalf("SaveRequiredStep(ai_provider, claude-cli) re-save: %v", err)
	}

	// Verify rows are unchanged.
	for _, task := range ai.Tasks {
		raw, ok, _ := projectRepo.GetSetting(ctx, "model."+task.ID)
		if !ok {
			t.Errorf("model.%s row missing after re-save", task.ID)
			continue
		}
		if raw != firstValues[task.ID] {
			t.Errorf("model.%s changed on re-save: got %q, had %q", task.ID, raw, firstValues[task.ID])
		}
	}
}

func TestSaveRequiredStep_AIProvider_UnrecognizedProviderWritesNoRows(t *testing.T) {
	cfg := onboarding.Config{CompletionKey: completionKey, StateKey: stateKey}
	db := testhelpers.OpenTestDB(t)
	settingsRepo := settings.NewRepository(db)
	projectRepo := repository.New(db)
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	modelSvc := services.NewModelService(projectRepo, entitlements.RuntimeModeSelfHosted, resolver)
	svc, err := onboarding.NewService(cfg, settingsRepo, projectRepo, modelSvc)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()

	// Complete welcome step first.
	if _, err := svc.SaveRequiredStep(ctx, onboarding.RequiredStepWelcome, nil); err != nil {
		t.Fatalf("SaveRequiredStep(welcome): %v", err)
	}

	// Save ai_provider with an unrecognized provider.
	if _, err := svc.SaveRequiredStep(ctx, onboarding.RequiredStepAIProvider, map[string]string{
		"AI_PROVIDER": "unknown-provider",
	}); err != nil {
		t.Fatalf("SaveRequiredStep(ai_provider): %v", err)
	}

	// Verify no model.<taskID> rows were written.
	for _, task := range ai.Tasks {
		key := "model." + task.ID
		_, ok, getErr := projectRepo.GetSetting(ctx, key)
		if getErr != nil {
			t.Fatalf("GetSetting(%q): %v", key, getErr)
		}
		if ok {
			t.Errorf("model.%s row was written for unrecognized provider, expected none", task.ID)
		}
	}

	// Verify ai_provider was still stored.
	val, ok, getErr := projectRepo.GetSetting(ctx, "ai_provider")
	if getErr != nil {
		t.Fatalf("GetSetting(ai_provider): %v", getErr)
	}
	if !ok || val != "unknown-provider" {
		t.Errorf("ai_provider = %q (ok=%v); want %q", val, ok, "unknown-provider")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────


