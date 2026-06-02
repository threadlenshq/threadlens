package repository_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

func TestGoogleResultFilterMetadataFieldsExist(t *testing.T) {
	conf := 0.85
	jobID := int64(7)
	filtered := "2026-06-02T00:00:00Z"
	reason := "spam"
	res := repository.GoogleResult{
		FilterState:       domain.FilterStateFiltered,
		FilterReason:      &reason,
		FilterReasons:     []string{domain.FilterReasonSpam},
		FilterExplanation: "test explanation",
		FilterConfidence:  &conf,
		FilterSource:      domain.FilterSourceRules,
		FilterSignature:   "filter:def456",
		FilterJobID:       &jobID,
		FilteredAt:        &filtered,
		RecoveredAt:       nil,
		RecoveryNote:      nil,
		SourceIdentity:    domain.SourceIdentity{"domain": "spam.example"},
	}
	body, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, key := range []string{
		"\"filter_state\"", "\"filter_reason\"", "\"filter_reasons\"",
		"\"filter_explanation\"", "\"filter_confidence\"", "\"filter_source\"",
		"\"filter_signature\"", "\"filter_job_id\"", "\"filtered_at\"",
		"\"recovered_at\"", "\"recovery_note\"", "\"source_identity\"",
	} {
		if !strings.Contains(text, key) {
			t.Fatalf("GoogleResult json missing key %s: %s", key, text)
		}
	}
}
