package modules

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
)

func TestRegistrySortsModulesByID(t *testing.T) {
	registry := NewRegistry(RouteFuncModule{ModuleID: "z", ModuleName: "Z"}, CoreResearchModule{})
	statuses := registry.Statuses()
	if len(statuses) != 2 {
		t.Fatalf("len(statuses) = %d, want 2", len(statuses))
	}
	if statuses[0].ID != "core_research" || statuses[1].ID != "z" {
		t.Fatalf("statuses not sorted: %+v", statuses)
	}
}

func TestRegistryMountRoutes(t *testing.T) {
	registry := NewRegistry(RouteFuncModule{ModuleID: "ops", ModuleName: "Ops", RoutePath: "/api/test-module"})
	r := chi.NewRouter()
	registry.MountRoutes(r)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/test-module", nil))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
}

func TestRegistryPanicOnDuplicateID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate module ID")
		}
	}()
	NewRegistry(
		RouteFuncModule{ModuleID: "dup", ModuleName: "A"},
		RouteFuncModule{ModuleID: "dup", ModuleName: "B"},
	)
}

func TestRegistryPanicOnNilModule(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil module")
		}
	}()
	NewRegistry(nil)
}

func TestRegistryPanicOnEmptyID(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty module ID")
		}
	}()
	NewRegistry(RouteFuncModule{ModuleID: "", ModuleName: "NoID"})
}

func TestCoreResearchModuleCapabilitiesDefensiveCopy(t *testing.T) {
	m := CoreResearchModule{}
	caps1 := m.Capabilities()
	if len(caps1) == 0 {
		t.Fatal("expected non-empty capabilities from CoreResearchModule")
	}
	// Mutate the returned slice.
	caps1[0] = entitlements.Capability("mutated")
	caps2 := m.Capabilities()
	if caps2[0] == entitlements.Capability("mutated") {
		t.Fatal("Capabilities() must return a defensive copy; global CoreCapabilities was mutated")
	}
}

func TestRegistryCapabilitiesAggregates(t *testing.T) {
	registry := NewRegistry(CoreResearchModule{})
	caps := registry.Capabilities()
	if len(caps) == 0 {
		t.Fatal("expected capabilities from CoreResearchModule")
	}
}
