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
