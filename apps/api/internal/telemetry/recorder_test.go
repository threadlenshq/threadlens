package telemetry

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRecorder_NoopWhenEnvOptInFalse(t *testing.T) {
	r := NewRecorder(RecorderConfig{EnvOptIn: false})
	r.Record(EventInstanceStarted) // should not panic
	r.Shutdown()                   // should return immediately
}

func TestRecorder_RejectsInvalidEventName(t *testing.T) {
	var mu sync.Mutex
	var received []Batch
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b Batch
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &b)
		mu.Lock()
		received = append(received, b)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	r := NewRecorder(RecorderConfig{
		EnvOptIn:       true,
		WorkerURL:      srv.URL,
		ScoutVersion:   "0.7.2",
		DeploymentType: "local",
		InstanceID:     "test-uuid",
		ConsentChecker: func() string { return "granted" },
	})

	r.Record(EventName("not_a_real_event")) // should be dropped
	r.Record(EventInstanceStarted)           // should be queued
	r.Shutdown()

	mu.Lock()
	defer mu.Unlock()
	if len(received) == 0 {
		t.Fatal("expected at least one batch to be sent")
	}
	for _, b := range received {
		for _, e := range b.Events {
			if e.EventName == "not_a_real_event" {
				t.Error("invalid event name should not appear in batch")
			}
		}
	}
}

func TestRecorder_BatchesUpTo50(t *testing.T) {
	var mu sync.Mutex
	var batches []Batch
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b Batch
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &b)
		mu.Lock()
		batches = append(batches, b)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	r := NewRecorder(RecorderConfig{
		EnvOptIn:       true,
		WorkerURL:      srv.URL,
		ScoutVersion:   "0.7.2",
		DeploymentType: "docker",
		InstanceID:     "test-uuid",
		ConsentChecker: func() string { return "granted" },
	})

	// Record exactly 50 events.
	for i := 0; i < 50; i++ {
		r.Record(EventFeatureScoutRun)
	}
	r.Shutdown()

	mu.Lock()
	defer mu.Unlock()
	totalEvents := 0
	for _, b := range batches {
		totalEvents += len(b.Events)
	}
	if totalEvents != 50 {
		t.Errorf("expected 50 events total, got %d", totalEvents)
	}
}

func TestRecorder_SkipsFlushWhenConsentNotGranted(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	r := NewRecorder(RecorderConfig{
		EnvOptIn:       true,
		WorkerURL:      srv.URL,
		ScoutVersion:   "0.7.2",
		DeploymentType: "local",
		InstanceID:     "test-uuid",
		ConsentChecker: func() string { return "declined" },
	})

	r.Record(EventInstanceStarted)
	r.Shutdown()

	if called {
		t.Error("flush should not have been called when consent is declined")
	}
}

func TestRecorder_SkipsFlushWhenInstanceIDEmpty(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	r := NewRecorder(RecorderConfig{
		EnvOptIn:       true,
		WorkerURL:      srv.URL,
		ScoutVersion:   "0.7.2",
		DeploymentType: "local",
		InstanceID:     "",
		ConsentChecker: func() string { return "granted" },
	})

	r.Record(EventInstanceStarted)
	r.Shutdown()

	if called {
		t.Error("flush should not have been called when instance_id is empty")
	}
}

func TestRecorder_WireFormat(t *testing.T) {
	var mu sync.Mutex
	var received Batch
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		json.Unmarshal(body, &received)
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	r := NewRecorder(RecorderConfig{
		EnvOptIn:       true,
		WorkerURL:      srv.URL,
		ScoutVersion:   "0.7.2",
		DeploymentType: "docker",
		InstanceID:     "550e8400-e29b-41d4-a716-446655440000",
		ConsentChecker: func() string { return "granted" },
	})

	r.Record(EventInstanceStarted)
	r.Shutdown()

	mu.Lock()
	defer mu.Unlock()
	if received.InstanceID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("unexpected instance_id: %s", received.InstanceID)
	}
	if len(received.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received.Events))
	}
	evt := received.Events[0]
	if evt.EventName != EventInstanceStarted {
		t.Errorf("unexpected event_name: %s", evt.EventName)
	}
	if evt.Source != "server" {
		t.Errorf("unexpected source: %s", evt.Source)
	}
	if evt.ScoutVersion != "0.7.2" {
		t.Errorf("unexpected scout_version: %s", evt.ScoutVersion)
	}
	if evt.EventTimeUnixMs == 0 {
		t.Error("event_time_unix_ms should be non-zero")
	}
}

func TestDetectDeploymentType(t *testing.T) {
	// Default should be "local"
	t.Setenv("SCOUT_ONBOARDING_MODE", "")
	if got := DetectDeploymentType(); got != "local" {
		t.Errorf("expected 'local', got %q", got)
	}
	t.Setenv("SCOUT_ONBOARDING_MODE", "docker")
	if got := DetectDeploymentType(); got != "docker" {
		t.Errorf("expected 'docker', got %q", got)
	}
}

func TestIsValidEventName(t *testing.T) {
	if !IsValidEventName(EventInstanceStarted) {
		t.Error("instance_started should be valid")
	}
	if IsValidEventName(EventName("bogus")) {
		t.Error("bogus should be invalid")
	}
}

// Ensure the ping ticker type compiles and the constant is reasonable.
func TestPingInterval(t *testing.T) {
	if pingInterval != 24*time.Hour {
		t.Errorf("expected ping interval of 24h, got %v", pingInterval)
	}
}
