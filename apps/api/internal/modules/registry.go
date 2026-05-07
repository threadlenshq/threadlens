// Package modules provides the compiled trusted module registry for the ThreadLens API.
// Modules are compiled into the binary — there is no runtime plugin loading or filesystem discovery.
package modules

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
)

// Module is the contract that every compiled module must satisfy.
type Module interface {
	// ID returns a unique, stable identifier for this module (e.g. "core_research").
	ID() string
	// Name returns a human-readable display name for this module.
	Name() string
	// Capabilities returns the entitlement capabilities this module provides.
	Capabilities() []entitlements.Capability
	// MountRoutes registers the module's HTTP routes on the provided router.
	MountRoutes(r chi.Router)
}

// Registry holds an ordered, deduplicated set of compiled modules.
type Registry struct {
	modules []Module
}

// NewRegistry constructs a Registry from the provided modules, sorted deterministically by ID.
// It panics if any module is nil, has an empty ID, or if duplicate IDs are registered.
func NewRegistry(mods ...Module) *Registry {
	seen := make(map[string]struct{}, len(mods))
	copyMods := make([]Module, 0, len(mods))
	for _, m := range mods {
		if m == nil {
			panic("modules.NewRegistry: nil module is not allowed")
		}
		id := m.ID()
		if id == "" {
			panic("modules.NewRegistry: module with empty ID is not allowed")
		}
		if _, exists := seen[id]; exists {
			panic(fmt.Sprintf("modules.NewRegistry: duplicate module ID %q", id))
		}
		seen[id] = struct{}{}
		copyMods = append(copyMods, m)
	}
	sort.Slice(copyMods, func(i, j int) bool { return copyMods[i].ID() < copyMods[j].ID() })
	return &Registry{modules: copyMods}
}

// Modules returns a copy of the registered module slice.
func (r *Registry) Modules() []Module {
	return append([]Module(nil), r.modules...)
}

// Statuses returns an entitlement ModuleStatus for every registered module.
// All registered modules are considered enabled by the compiled registry.
func (r *Registry) Statuses() []entitlements.ModuleStatus {
	statuses := make([]entitlements.ModuleStatus, 0, len(r.modules))
	for _, module := range r.modules {
		statuses = append(statuses, entitlements.ModuleStatus{ID: module.ID(), Name: module.Name(), Enabled: true})
	}
	return statuses
}

// Capabilities aggregates every capability contributed by all registered modules.
func (r *Registry) Capabilities() []entitlements.Capability {
	var caps []entitlements.Capability
	for _, module := range r.modules {
		caps = append(caps, module.Capabilities()...)
	}
	return caps
}

// MountRoutes calls MountRoutes on each registered module in deterministic order.
func (r *Registry) MountRoutes(router chi.Router) {
	for _, module := range r.modules {
		module.MountRoutes(router)
	}
}

// CoreResearchModule is the built-in open-core research module. It contributes all core
// entitlement capabilities and does not register additional HTTP routes.
type CoreResearchModule struct{}

func (CoreResearchModule) ID() string   { return "core_research" }
func (CoreResearchModule) Name() string { return "Core Research" }

// Capabilities returns a defensive copy of the core capability set.
func (CoreResearchModule) Capabilities() []entitlements.Capability {
	return append([]entitlements.Capability(nil), entitlements.CoreCapabilities...)
}

func (CoreResearchModule) MountRoutes(chi.Router) {}

// RouteFuncModule is a lightweight Module implementation that registers a single GET
// route returning HTTP 204 No Content. It is intended for testing and simple extension seams.
type RouteFuncModule struct {
	ModuleID   string
	ModuleName string
	// RoutePath is the GET path to register. If empty, no route is mounted.
	RoutePath string
}

func (m RouteFuncModule) ID() string   { return m.ModuleID }
func (m RouteFuncModule) Name() string { return m.ModuleName }
func (m RouteFuncModule) Capabilities() []entitlements.Capability { return nil }
func (m RouteFuncModule) MountRoutes(r chi.Router) {
	if m.RoutePath == "" {
		return
	}
	r.Get(m.RoutePath, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
}
