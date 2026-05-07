package services

import (
	"context"
	"net/http"

	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/templates"
	"github.com/kyle/scout/open-core/apps/api/internal/tenant"
)

type RuntimeService struct {
	mode     entitlements.RuntimeMode
	resolver entitlements.Resolver
	catalog  templates.Catalog
}

func NewRuntimeService(mode entitlements.RuntimeMode, resolver entitlements.Resolver, catalog templates.Catalog) *RuntimeService {
	return &RuntimeService{mode: mode, resolver: resolver, catalog: catalog}
}

func (s *RuntimeService) Snapshot(ctx context.Context) (entitlements.Snapshot, int, string) {
	subject := tenant.SubjectFromContext(ctx, s.mode)
	snapshot, err := s.resolver.Snapshot(ctx, subject)
	if err != nil {
		return entitlements.Snapshot{}, http.StatusInternalServerError, "Internal server error"
	}
	return snapshot, http.StatusOK, ""
}

func (s *RuntimeService) ListPromptPacks(ctx context.Context) ([]templates.PromptPack, int, string) {
	subject := tenant.SubjectFromContext(ctx, s.mode)
	packs, err := s.catalog.List(ctx, subject)
	if err != nil {
		return nil, entitlements.StatusCode(err), err.Error()
	}
	return packs, http.StatusOK, ""
}
