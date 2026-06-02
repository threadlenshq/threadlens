package pipeline

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

type fakeTrustLookup struct{ records []domain.TrustRecord }

func (f fakeTrustLookup) ListTrustRecords(ctx context.Context, projectID string) ([]domain.TrustRecord, error) {
	return f.records, nil
}

func TestClassifyFilterInputTrustedSourceWinsBeforeRules(t *testing.T) {
	c := NewFilterClassifier(fakeTrustLookup{records: []domain.TrustRecord{{ProjectID: "p1", Platform: "reddit", TrustType: domain.TrustTypeSource, SourceKind: "reddit_author", SourceKey: "trusted_user"}}}, nil)
	d, err := c.Classify(context.Background(), "p1", FilterInput{Platform: "reddit", Title: "I built a launch tool", SourceIdentity: domain.SourceIdentity{"reddit_author": "trusted_user"}})
	if err != nil {
		t.Fatal(err)
	}
	if d.State != domain.FilterStateVisible || d.Source != domain.FilterSourceTrustedOverride {
		t.Fatalf("decision = %#v", d)
	}
}

func TestClassifyFilterInputFlagsPromotionalLaunch(t *testing.T) {
	c := NewFilterClassifier(fakeTrustLookup{}, nil)
	d, err := c.Classify(context.Background(), "p1", FilterInput{Platform: "reddit", Title: "I built an AI tool", Body: "feedback welcome", SourceIdentity: domain.SourceIdentity{"reddit_author": "maker"}})
	if err != nil {
		t.Fatal(err)
	}
	if d.State != domain.FilterStateFiltered {
		t.Fatalf("state = %s", d.State)
	}
	if d.Reason != domain.FilterReasonSpam {
		t.Fatalf("reason = %s", d.Reason)
	}
	if d.Signature == "" {
		t.Fatal("expected signature")
	}
}

func TestClassifyFilterInputAllowsGenuinePainPost(t *testing.T) {
	c := NewFilterClassifier(fakeTrustLookup{}, nil)
	d, err := c.Classify(context.Background(), "p1", FilterInput{Platform: "bluesky", Title: "Why is billing still confusing?", Body: "I keep getting surprised by invoices.", SourceIdentity: domain.SourceIdentity{"bluesky_handle": "person.example"}})
	if err != nil {
		t.Fatal(err)
	}
	if d.State != domain.FilterStateVisible {
		t.Fatalf("decision = %#v", d)
	}
	if d.Source != domain.FilterSourceNone {
		t.Fatalf("source = %s", d.Source)
	}
}

func TestClassifyFilterInputFlagsAIBoilerplateWithConfidence(t *testing.T) {
	c := NewFilterClassifier(fakeTrustLookup{}, nil)
	d, err := c.Classify(context.Background(), "p1", FilterInput{Platform: "google", Title: "Unlock productivity today", Body: "In today's fast-paced digital landscape, leverage cutting-edge solutions to streamline workflows and unlock your potential.", SourceIdentity: domain.SourceIdentity{"domain": "spam.example"}})
	if err != nil {
		t.Fatal(err)
	}
	if d.State != domain.FilterStateFiltered {
		t.Fatalf("decision = %#v", d)
	}
	if d.Reason != domain.FilterReasonAIGenerated {
		t.Fatalf("reason = %s", d.Reason)
	}
}
