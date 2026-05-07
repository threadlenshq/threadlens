package entitlements

import (
	"context"
	"net/http"
	"testing"
)

func TestLocalResolverGrantsCoreCapabilities(t *testing.T) {
	resolver := NewLocalResolver(RuntimeModeSelfHosted, nil)
	snapshot, err := resolver.Snapshot(context.Background(), Subject{})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if snapshot.RuntimeMode != RuntimeModeSelfHosted {
		t.Fatalf("RuntimeMode = %q, want %q", snapshot.RuntimeMode, RuntimeModeSelfHosted)
	}
	for _, cap := range CoreCapabilities {
		if !snapshot.Capabilities[cap] {
			t.Fatalf("capability %s must be granted in local resolver", cap)
		}
	}
	if snapshot.Capabilities[CapabilityManagedAIUse] {
		t.Fatal("managed AI capability must not be granted by the local open-core resolver")
	}
}

func TestLocalResolverDeniesHostedOnlyCapability(t *testing.T) {
	resolver := NewLocalResolver(RuntimeModeSelfHosted, nil)
	decision, err := resolver.Check(context.Background(), CheckRequest{Capability: CapabilityTenantsManage})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if decision.Allowed {
		t.Fatal("hosted tenant management must be denied by the local resolver")
	}
	if decision.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("StatusCode = %d, want %d", decision.StatusCode, http.StatusPaymentRequired)
	}
}

func TestEnsureAllowedReturnsDeniedError(t *testing.T) {
	err := EnsureAllowed(Decision{Allowed: false, Capability: CapabilityManagedAIUse, Reason: "capability_not_granted"})
	denied, ok := err.(*DeniedError)
	if !ok {
		t.Fatalf("error type = %T, want *DeniedError", err)
	}
	if denied.Decision.Capability != CapabilityManagedAIUse {
		t.Fatalf("Capability = %q, want %q", denied.Decision.Capability, CapabilityManagedAIUse)
	}
}
