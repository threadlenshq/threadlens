package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProjectJSONUsesSnakeCase(t *testing.T) {
	project := Project{ID: "p1", Name: "Scout", Mode: "research", SelectedReportID: IntPtr(7), SelectedClusterIndex: IntPtr(2)}
	body, err := json.Marshal(project)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, key := range []string{"\"selected_report_id\":7", "\"selected_cluster_index\":2"} {
		if !strings.Contains(text, key) {
			t.Fatalf("json %s missing %s", text, key)
		}
	}
}

// --- filtering.go contract tests ---

func TestFilterStateConstants(t *testing.T) {
	if FilterStateVisible != "visible" {
		t.Fatalf("unexpected FilterStateVisible: %s", FilterStateVisible)
	}
	if FilterStateFiltered != "filtered" {
		t.Fatalf("unexpected FilterStateFiltered: %s", FilterStateFiltered)
	}
}

func TestFilterSourceConstants(t *testing.T) {
	for _, pair := range [][2]string{
		{FilterSourceNone, "none"},
		{FilterSourceRules, "rules"},
		{FilterSourceAI, "ai"},
		{FilterSourceTrustedOverride, "trusted_override"},
	} {
		if pair[0] != pair[1] {
			t.Fatalf("constant mismatch: got %s want %s", pair[0], pair[1])
		}
	}
}

func TestFilterReasonConstants(t *testing.T) {
	for _, pair := range [][2]string{
		{FilterReasonSpam, "spam"},
		{FilterReasonBotLike, "bot_like"},
		{FilterReasonLowQualityAccount, "low_quality_account"},
		{FilterReasonAIGenerated, "ai_generated"},
		{FilterReasonTrustedOverride, "trusted_override"},
	} {
		if pair[0] != pair[1] {
			t.Fatalf("constant mismatch: got %s want %s", pair[0], pair[1])
		}
	}
}

func TestFindingTypeConstants(t *testing.T) {
	if FindingTypePost != "post" {
		t.Fatalf("unexpected FindingTypePost: %s", FindingTypePost)
	}
	if FindingTypeGoogleResult != "google_result" {
		t.Fatalf("unexpected FindingTypeGoogleResult: %s", FindingTypeGoogleResult)
	}
}

func TestFilterJobStatusConstants(t *testing.T) {
	for _, pair := range [][2]string{
		{FilterJobStatusRunning, "running"},
		{FilterJobStatusCompleted, "completed"},
		{FilterJobStatusFailed, "failed"},
	} {
		if pair[0] != pair[1] {
			t.Fatalf("constant mismatch: got %s want %s", pair[0], pair[1])
		}
	}
}

func TestSourceIdentityJSON(t *testing.T) {
	si := SourceIdentity{"author": "alice"}
	got := si.JSON()
	if !strings.Contains(got, "alice") {
		t.Fatalf("SourceIdentity.JSON() = %s, want author key", got)
	}
	var nilSI SourceIdentity
	if nilSI.JSON() != "{}" {
		t.Fatalf("nil SourceIdentity.JSON() = %s, want {}", nilSI.JSON())
	}
}

func TestFilterMetadataJSONFields(t *testing.T) {
	fm := FilterMetadata{
		FilterState:  FilterStateVisible,
		FilterSource: FilterSourceNone,
	}
	body, err := json.Marshal(fm)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, key := range []string{"filter_state", "filter_source"} {
		if !strings.Contains(text, key) {
			t.Fatalf("FilterMetadata json missing key %s: %s", key, text)
		}
	}
}

func TestFilteredFindingJSONFields(t *testing.T) {
	ff := FilteredFinding{
		ID:          "p_abc",
		FindingType: FindingTypePost,
		FilterState: FilterStateFiltered,
	}
	body, err := json.Marshal(ff)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, key := range []string{"\"finding_type\"", "\"filter_state\""} {
		if !strings.Contains(text, key) {
			t.Fatalf("FilteredFinding json missing key %s: %s", key, text)
		}
	}
}

func TestFilterJobStatusField(t *testing.T) {
	job := FilterJob{
		ID:             1,
		ProjectID:      "proj1",
		Status:         FilterJobStatusRunning,
		RequestedScope: FilterJobScopeSelectedVisiblePosts,
		StartedAt:      "2026-06-02T00:00:00Z",
	}
	body, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "running") {
		t.Fatalf("FilterJob json missing status: %s", text)
	}
}

func TestPaginationJSONNames(t *testing.T) {
	p := Pagination{Page: 1, Limit: 20, Total: 21, TotalPages: 2, HasPreviousPage: false, HasNextPage: true}
	body, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "\"hasNextPage\":true") {
		t.Fatalf("pagination json = %s", body)
	}
}
