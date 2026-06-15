package services

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
	"github.com/kyle/scout/open-core/apps/api/internal/tenant"
)

// newTestModelService creates a ModelService wired to an in-memory DB.
func newTestModelService(t *testing.T) *ModelService {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	return NewModelService(repo, entitlements.RuntimeModeSelfHosted, resolver)
}

// ctxWithTenant returns a context with a self-hosted tenant subject.
func ctxWithTenant() context.Context {
	return tenant.WithSubject(context.Background(), entitlements.Subject{
		RuntimeMode: entitlements.RuntimeModeSelfHosted,
	})
}

func TestCatalog_ReturnsDefaultProviderWhenMissing(t *testing.T) {
	svc := newTestModelService(t)
	ctx := ctxWithTenant()

	result, err := svc.Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	provider, ok := result["currentProvider"].(string)
	if !ok {
		t.Fatal("currentProvider missing or not a string")
	}
	if provider != "sdk" {
		t.Errorf("currentProvider = %q; want %q", provider, "sdk")
	}
}

func TestCatalog_ReturnsStoredProvider(t *testing.T) {
	svc := newTestModelService(t)
	ctx := ctxWithTenant()

	if err := svc.StoreProvider(ctx, "opencode"); err != nil {
		t.Fatalf("StoreProvider: %v", err)
	}

	result, err := svc.Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	provider := result["currentProvider"].(string)
	if provider != "opencode" {
		t.Errorf("currentProvider = %q; want %q", provider, "opencode")
	}
}

func TestCatalog_OverridesTaskDefaultsPerProvider(t *testing.T) {
	svc := newTestModelService(t)
	ctx := ctxWithTenant()

	// Store "opencode" as the provider.
	if err := svc.StoreProvider(ctx, "opencode"); err != nil {
		t.Fatalf("StoreProvider: %v", err)
	}

	result, err := svc.Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	tasks, ok := result["tasks"].([]ai.TaskEntry)
	if !ok {
		t.Fatal("tasks missing or wrong type")
	}

	// post_scoring should have opencode-go:deepseek-v4-flash as default.
	for _, task := range tasks {
		if task.ID == "post_scoring" {
			if task.Default != "opencode-go:deepseek-v4-flash" {
				t.Errorf("post_scoring default = %q; want %q", task.Default, "opencode-go:deepseek-v4-flash")
			}
			return
		}
	}
	t.Error("post_scoring task not found in catalog response")
}

func TestCatalog_CopilotProviderDefaults(t *testing.T) {
	svc := newTestModelService(t)
	ctx := ctxWithTenant()

	if err := svc.StoreProvider(ctx, "copilot"); err != nil {
		t.Fatalf("StoreProvider: %v", err)
	}

	result, err := svc.Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	tasks := result["tasks"].([]ai.TaskEntry)
	for _, task := range tasks {
		if task.ID == "post_scoring" {
			if task.Default != "copilot:gpt-5-mini" {
				t.Errorf("post_scoring default = %q; want %q", task.Default, "copilot:gpt-5-mini")
			}
			return
		}
	}
	t.Error("post_scoring task not found")
}

func TestCatalog_FallsBackToDefaultWhenModelNotInCatalog(t *testing.T) {
	svc := newTestModelService(t)
	ctx := ctxWithTenant()

	// Temporarily mutate the first task's DefaultByProvider to reference a
	// model that does not exist in the catalog, then restore it on cleanup.
	original := ai.Tasks[0].DefaultByProvider["sdk"]
	ai.Tasks[0].DefaultByProvider["sdk"] = "nonexistent:model"
	t.Cleanup(func() {
		ai.Tasks[0].DefaultByProvider["sdk"] = original
	})

	// Store "sdk" as the provider (it's the default anyway).
	if err := svc.StoreProvider(ctx, "sdk"); err != nil {
		t.Fatalf("StoreProvider: %v", err)
	}

	result, err := svc.Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	tasks := result["tasks"].([]ai.TaskEntry)
	for _, task := range tasks {
		if task.ID == ai.Tasks[0].ID {
			if task.Default != ai.Tasks[0].Default {
				t.Errorf("task %q default = %q; want fallback %q when model not in catalog", task.ID, task.Default, ai.Tasks[0].Default)
			}
			return
		}
	}
	t.Error("first task not found in catalog response")
}

func TestStoreProvider_RejectsEmptyString(t *testing.T) {
	svc := newTestModelService(t)
	ctx := ctxWithTenant()

	err := svc.StoreProvider(ctx, "")
	if err == nil {
		t.Fatal("StoreProvider(\"\"): expected error, got nil")
	}
	modelErr, ok := err.(*ModelError)
	if !ok {
		t.Fatalf("expected *ModelError, got %T", err)
	}
	if modelErr.Kind != "invalidProvider" {
		t.Errorf("ModelError.Kind = %q; want %q", modelErr.Kind, "invalidProvider")
	}
}

func TestStoreProvider_RoundTripsThroughGetSetting(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	svc := NewModelService(repo, entitlements.RuntimeModeSelfHosted, resolver)
	ctx := ctxWithTenant()

	if err := svc.StoreProvider(ctx, "opencode"); err != nil {
		t.Fatalf("StoreProvider: %v", err)
	}

	val, ok, err := repo.GetSetting(ctx, "ai_provider")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if !ok {
		t.Fatal("ai_provider key not found after StoreProvider")
	}
	if val != "opencode" {
		t.Errorf("ai_provider = %q; want %q", val, "opencode")
	}
}
