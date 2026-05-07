package usage

import (
	"context"
	"testing"
)

func TestNoopMeterRecordDoesNotFail(t *testing.T) {
	if err := (NoopMeter{}).Record(context.Background(), Event{TaskID: "post_scoring"}); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
}

func TestMemoryMeterRecordsEvents(t *testing.T) {
	meter := NewMemoryMeter()
	if err := meter.Record(context.Background(), Event{TenantID: "tenant-1", TaskID: "report_clustering", Success: true}); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	events := meter.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].TenantID != "tenant-1" || events[0].TaskID != "report_clustering" || !events[0].Success {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}
