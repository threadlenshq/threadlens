package templates

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
)

func TestLocalCatalogListReturnsEmptyCatalog(t *testing.T) {
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	catalog := NewLocalCatalog(resolver)
	packs, err := catalog.List(context.Background(), entitlements.Subject{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(packs) != 0 {
		t.Fatalf("len(packs) = %d, want 0", len(packs))
	}
}

func TestLocalCatalogApplyValidatesRequest(t *testing.T) {
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	catalog := NewLocalCatalog(resolver)
	_, err := catalog.Apply(context.Background(), entitlements.Subject{}, ApplyRequest{})
	if err == nil || err.Error() != "packId is required" {
		t.Fatalf("error = %v, want packId is required", err)
	}
}
