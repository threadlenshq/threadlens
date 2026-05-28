package entitlements

import (
	"context"
	"net/http"
	"os"
	"testing"
)

// msgGoogleParallelKeyMissing is the runtime message code emitted when
// PARALLEL_API_KEY is absent or blank.
const msgGoogleParallelKeyMissing = "google_parallel_api_key_missing"

// TestLocalResolverDeniesGoogleScoutWhenParallelKeyIsWhitespace verifies that
// a PARALLEL_API_KEY containing only whitespace is treated as absent: Google
// scout capability is denied and a warning message is emitted, while all other
// core capabilities remain granted.
func TestLocalResolverDeniesGoogleScoutWhenParallelKeyIsWhitespace(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "   ")
	resolver := NewLocalResolver(RuntimeModeSelfHosted, nil)
	snapshot, err := resolver.Snapshot(context.Background(), Subject{})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if snapshot.RuntimeMode != RuntimeModeSelfHosted {
		t.Fatalf("RuntimeMode = %q, want %q", snapshot.RuntimeMode, RuntimeModeSelfHosted)
	}
	for _, cap := range CoreCapabilities {
		if cap == CapabilityScoutRunGoogle {
			continue
		}
		if !snapshot.Capabilities[cap] {
			t.Fatalf("capability %s must be granted in local resolver", cap)
		}
	}
	if snapshot.Capabilities[CapabilityScoutRunGoogle] {
		t.Fatal("google scout capability must be denied when PARALLEL_API_KEY is whitespace-only")
	}
	if snapshot.Capabilities[CapabilityManagedAIUse] {
		t.Fatal("managed AI capability must not be granted by the local open-core resolver")
	}
	if !hasRuntimeMessage(snapshot.Messages, msgGoogleParallelKeyMissing) {
		t.Fatalf("expected runtime message %q: %+v", msgGoogleParallelKeyMissing, snapshot.Messages)
	}
}

// TestLocalResolverDeniesGoogleScoutWhenParallelKeyIsUnset verifies that a
// completely unset PARALLEL_API_KEY (not present in the environment at all)
// also denies Google scout capability and emits the warning message.
func TestLocalResolverDeniesGoogleScoutWhenParallelKeyIsUnset(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "sentinel_to_ensure_setenv_registered") // register for cleanup
	if err := os.Unsetenv("PARALLEL_API_KEY"); err != nil {
		t.Fatalf("failed to unset PARALLEL_API_KEY: %v", err)
	}
	resolver := NewLocalResolver(RuntimeModeSelfHosted, nil)
	snapshot, err := resolver.Snapshot(context.Background(), Subject{})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if snapshot.Capabilities[CapabilityScoutRunGoogle] {
		t.Fatal("google scout capability must be denied when PARALLEL_API_KEY is unset")
	}
	if !hasRuntimeMessage(snapshot.Messages, msgGoogleParallelKeyMissing) {
		t.Fatalf("expected runtime message %q: %+v", msgGoogleParallelKeyMissing, snapshot.Messages)
	}
}

func TestLocalResolverGrantsGoogleScoutCapabilityWithParallelKey(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "parallel_test_key")
	resolver := NewLocalResolver(RuntimeModeSelfHosted, nil)
	snapshot, err := resolver.Snapshot(context.Background(), Subject{})
	if err != nil {
		t.Fatalf("Snapshot returned error: %v", err)
	}
	if !snapshot.Capabilities[CapabilityScoutRunGoogle] {
		t.Fatal("google scout capability must be granted when PARALLEL_API_KEY is configured")
	}
	if hasRuntimeMessage(snapshot.Messages, msgGoogleParallelKeyMissing) {
		t.Fatalf("missing-key message must not be present with configured key: %+v", snapshot.Messages)
	}
}

// TestLocalResolverCheckDeniesGoogleScoutWhenParallelKeyIsUnset verifies that
// Check returns a 402 denial for CapabilityScoutRunGoogle when the key is absent.
func TestLocalResolverCheckDeniesGoogleScoutWhenParallelKeyIsUnset(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "sentinel_to_ensure_setenv_registered")
	if err := os.Unsetenv("PARALLEL_API_KEY"); err != nil {
		t.Fatalf("failed to unset PARALLEL_API_KEY: %v", err)
	}
	resolver := NewLocalResolver(RuntimeModeSelfHosted, nil)
	decision, err := resolver.Check(context.Background(), CheckRequest{Capability: CapabilityScoutRunGoogle})
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if decision.Allowed {
		t.Fatal("google scout check must be denied without PARALLEL_API_KEY")
	}
	if decision.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("StatusCode = %d, want %d", decision.StatusCode, http.StatusPaymentRequired)
	}
}

func hasRuntimeMessage(messages []RuntimeMessage, code string) bool {
	for _, message := range messages {
		if message.Code == code {
			return true
		}
	}
	return false
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
