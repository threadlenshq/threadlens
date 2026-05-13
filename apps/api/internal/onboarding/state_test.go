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
	"strings"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
)

// ── 1. NewProgress defaults ───────────────────────────────────────────────────

func TestNewProgress_DefaultsToRequiredSetupWelcome(t *testing.T) {
	p := onboarding.NewProgress()

	if p.Version != 1 {
		t.Errorf("Version = %d; want 1", p.Version)
	}
	if p.RequiredSetup.Status != onboarding.RequiredStatusNotStarted {
		t.Errorf("RequiredSetup.Status = %v; want RequiredStatusNotStarted", p.RequiredSetup.Status)
	}
	if p.RequiredSetup.CurrentStep != onboarding.RequiredStepWelcome {
		t.Errorf("RequiredSetup.CurrentStep = %v; want RequiredStepWelcome", p.RequiredSetup.CurrentStep)
	}
	if value, ok := p.Exploration.Items[onboarding.ExplorationItemStarterProject]; !ok {
		t.Error("Exploration.Items missing key ExplorationItemStarterProject")
	} else if value != onboarding.ItemStatePending {
		t.Errorf("Exploration.Items[ExplorationItemStarterProject] = %v; want ItemStatePending", value)
	}
}

// ── 2. Phase calculation ──────────────────────────────────────────────────────

func TestProgressStatus_Phases(t *testing.T) {
	p := onboarding.NewProgress()

	// Enabled + fresh progress: should be in required-setup phase.
	if got := onboarding.PhaseForProgress(true, p); got != onboarding.PhaseRequiredSetup {
		t.Errorf("PhaseForProgress(true, new) = %v; want PhaseRequiredSetup", got)
	}

	// Disabled: phase must be PhaseDisabled regardless of progress.
	if got := onboarding.PhaseForProgress(false, p); got != onboarding.PhaseDisabled {
		t.Errorf("PhaseForProgress(false, new) = %v; want PhaseDisabled", got)
	}

	// Mark required setup complete and exploration active.
	p.RequiredSetup.Status = onboarding.RequiredStatusComplete
	if got := onboarding.PhaseForProgress(true, p); got != onboarding.PhaseExploration {
		t.Errorf("PhaseForProgress(true, required-complete) = %v; want PhaseExploration", got)
	}

	// Mark all exploration items skipped.
	for k := range p.Exploration.Items {
		p.Exploration.Items[k] = onboarding.ItemStateSkipped
	}
	if got := onboarding.PhaseForProgress(true, p); got != onboarding.PhaseComplete {
		t.Errorf("PhaseForProgress(true, all-skipped) = %v; want PhaseComplete", got)
	}
}

// ── 3. JSON marshaling does not contain secrets ───────────────────────────────

func TestProgressJSONDoesNotContainSecrets(t *testing.T) {
	p := onboarding.NewProgress()
	p.Context.AIProviderPath = "anthropic"

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)
	for _, forbidden := range []string{"sk-ant-secret", "GEMINI_API_KEY=", "ANTHROPIC_API_KEY="} {
		if strings.Contains(jsonStr, forbidden) {
			t.Errorf("marshaled JSON contains forbidden string %q: %s", forbidden, jsonStr)
		}
	}
	if !strings.Contains(jsonStr, `"aiProviderPath":"anthropic"`) {
		t.Errorf("marshaled JSON missing expected field aiProviderPath: %s", jsonStr)
	}
}

// ── 4. ExplorationComplete ────────────────────────────────────────────────────

func TestExplorationCompleteWhenEveryItemCompletedOrSkipped(t *testing.T) {
	p := onboarding.NewProgress()

	// Not complete when exploration has pending items.
	if onboarding.ExplorationComplete(p.Exploration.Items) {
		t.Error("ExplorationComplete should be false for a fresh progress value")
	}

	// Mark every item completed or skipped (alternating).
	i := 0
	for k := range p.Exploration.Items {
		if i%2 == 0 {
			p.Exploration.Items[k] = onboarding.ItemStateCompleted
		} else {
			p.Exploration.Items[k] = onboarding.ItemStateSkipped
		}
		i++
	}

	if !onboarding.ExplorationComplete(p.Exploration.Items) {
		t.Error("ExplorationComplete should be true when every item is completed or skipped")
	}
}
