package ai

import (
	"strings"
	"testing"
)

func TestEveryTaskHasDefaultByProvider(t *testing.T) {
	for _, task := range Tasks {
		if task.DefaultByProvider == nil {
			t.Errorf("task %q has nil DefaultByProvider", task.ID)
		}
		if len(task.DefaultByProvider) == 0 {
			t.Errorf("task %q has empty DefaultByProvider", task.ID)
		}
	}
}

func TestDefaultByProviderReferencesRealModels(t *testing.T) {
	for _, task := range Tasks {
		for provider, modelID := range task.DefaultByProvider {
			if GetModel(modelID) == nil {
				t.Errorf("task %q DefaultByProvider[%q] = %q, but GetModel returns nil",
					task.ID, provider, modelID)
			}
		}
	}
}

func TestDefaultByProviderHasAllFiveProviders(t *testing.T) {
	requiredProviders := []string{"copilot", "claude-cli", "opencode", "sdk", "gemini"}
	for _, task := range Tasks {
		for _, p := range requiredProviders {
			if _, ok := task.DefaultByProvider[p]; !ok {
				t.Errorf("task %q DefaultByProvider missing provider %q", task.ID, p)
			}
		}
	}
}

func TestOpencodeDefaultsUseGoPrefix(t *testing.T) {
	for _, task := range Tasks {
		modelID := task.DefaultByProvider["opencode"]
		if modelID == "" {
			t.Errorf("task %q has no opencode default", task.ID)
			continue
		}
		// The opencode onboarding provider must map to opencode-go: entries,
		// not opencode: (free tier).
		if !strings.HasPrefix(modelID, "opencode-go:") {
			t.Errorf("task %q opencode default = %q; want opencode-go: prefix", task.ID, modelID)
		}
	}
}
