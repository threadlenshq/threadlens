package tenant

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
)

func TestSubjectFromContextDefaultsToLocal(t *testing.T) {
	subject := SubjectFromContext(context.Background(), entitlements.RuntimeModeSelfHosted)
	if subject.ActorID != "local-user" {
		t.Fatalf("ActorID = %q, want local-user", subject.ActorID)
	}
	if subject.TenantID != "local-instance" {
		t.Fatalf("TenantID = %q, want local-instance", subject.TenantID)
	}
}

func TestSubjectFromContextPreservesExplicitHostedSubject(t *testing.T) {
	ctx := WithSubject(context.Background(), entitlements.Subject{ActorID: "user-1", TenantID: "tenant-1", RuntimeMode: entitlements.RuntimeModeHosted})
	subject := SubjectFromContext(ctx, entitlements.RuntimeModeSelfHosted)
	if subject.ActorID != "user-1" || subject.TenantID != "tenant-1" || subject.RuntimeMode != entitlements.RuntimeModeHosted {
		t.Fatalf("unexpected subject: %+v", subject)
	}
}
