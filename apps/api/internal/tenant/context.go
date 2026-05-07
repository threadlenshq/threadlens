package tenant

import (
	"context"

	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
)

type contextKey struct{}

var subjectKey contextKey

func WithSubject(ctx context.Context, subject entitlements.Subject) context.Context {
	return context.WithValue(ctx, subjectKey, subject)
}

func SubjectFromContext(ctx context.Context, runtimeMode entitlements.RuntimeMode) entitlements.Subject {
	if subject, ok := ctx.Value(subjectKey).(entitlements.Subject); ok {
		if subject.RuntimeMode == "" {
			subject.RuntimeMode = runtimeMode
		}
		if subject.ActorID == "" {
			subject.ActorID = "local-user"
		}
		if subject.TenantID == "" {
			subject.TenantID = "local-instance"
		}
		return subject
	}
	return entitlements.Subject{ActorID: "local-user", TenantID: "local-instance", RuntimeMode: runtimeMode}
}
