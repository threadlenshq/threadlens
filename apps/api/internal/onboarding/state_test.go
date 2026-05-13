package onboarding_test

// state_test.go defines the expected state model API before implementation.
// These tests will compile-fail until Task 3 (state.go) is implemented.
//
// Invariants asserted here:
//   - NewProgress returns version 1 with required setup not started and current step welcome.
//   - PhaseForProgress correctly categorises a progress value.
//   - JSON marshaling of a progress object never leaks secret strings.
//   - ExplorationComplete is satisfied when every item is completed or skipped.

import (
	"encoding/json"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
)

// containsAny returns true if s contains any of the given substrings.
func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		for i := 0; i+len(n) <= len(s); i++ {
			if s[i:i+len(n)] == n {
				return true
			}
		}
	}
	return false
}

// ── 1. NewProgress defaults ───────────────────────────────────────────────────

func TestNewProgress_DefaultsToRequiredSetupWelcome(t *testing.T) {
	p := onboarding.NewProgress()

	if p.Version != 1 {
		t.Errorf("Version = %d; want 1", p.Version)
	}
	if p.RequiredSetup.Status != onboarding.StatusNotStarted {
		t.Errorf("RequiredSetup.Status = %v; want StatusNotStarted", p.RequiredSetup.Status)
	}
	if p.CurrentStep != onboarding.StepWelcome {
		t.Errorf("CurrentStep = %v; want StepWelcome", p.CurrentStep)
	}
}

// ── 2. Phase calculation ──────────────────────────────────────────────────────

func TestProgressStatus_Phases(t *testing.T) {
	p := onboarding.NewProgress()

	// Onboarding enabled: phase should be PhaseRequiredSetup (initial).
	phase := onboarding.PhaseForProgress(false, p)
	if phase != onboarding.PhaseRequiredSetup {
		t.Errorf("PhaseForProgress(enabled, new) = %v; want PhaseRequiredSetup", phase)
	}

	// Onboarding disabled: phase should be PhaseDisabled regardless of progress.
	phase = onboarding.PhaseForProgress(true, p)
	if phase != onboarding.PhaseDisabled {
		t.Errorf("PhaseForProgress(disabled, new) = %v; want PhaseDisabled", phase)
	}
}

// ── 3. JSON marshaling does not contain secrets ───────────────────────────────

func TestProgressJSONDoesNotContainSecrets(t *testing.T) {
	p := onboarding.NewProgress()

	// Inject fake secret values that must never appear in serialised state.
	p.AnthropicAPIKey = "sk-ant-secret"
	p.GeminiAPIKey = "gemini-secret-value"

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	secrets := []string{
		"sk-ant-secret",
		"GEMINI_API_KEY=",
		"ANTHROPIC_API_KEY=",
		"gemini-secret-value",
	}
	jsonStr := string(data)
	if containsAny(jsonStr, secrets) {
		t.Errorf("marshaled JSON contains a secret string: %s", jsonStr)
	}
}

// ── 4. ExplorationComplete ────────────────────────────────────────────────────

func TestExplorationCompleteWhenEveryItemCompletedOrSkipped(t *testing.T) {
	p := onboarding.NewProgress()

	// Not complete when exploration has pending items.
	if onboarding.ExplorationComplete(p) {
		t.Error("ExplorationComplete should be false for a fresh progress value")
	}

	// Mark every exploration item as completed or skipped.
	for i := range p.Exploration.Items {
		if i%2 == 0 {
			p.Exploration.Items[i].Status = onboarding.StatusCompleted
		} else {
			p.Exploration.Items[i].Status = onboarding.StatusSkipped
		}
	}

	if !onboarding.ExplorationComplete(p) {
		t.Error("ExplorationComplete should be true when every item is completed or skipped")
	}
}
