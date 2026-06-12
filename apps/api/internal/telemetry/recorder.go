package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	maxBatchSize    = 50
	flushInterval   = 30 * time.Second
	pingInterval    = 24 * time.Hour
	shutdownTimeout = 5 * time.Second

	// DefaultWorkerURL is the hardcoded telemetry ingest endpoint.
	// This is NOT configurable in production. The worker URL is fixed
	// to prevent accidental data loss; telemetry data belongs to the
	// Scout team for product improvement.
	DefaultWorkerURL = "https://telemetry.threadlens.dev"
)

// RecorderConfig holds the parameters needed to construct a Recorder.
type RecorderConfig struct {
	// EnvOptIn is the infrastructure-level gate. When false, the recorder
	// is a no-op and never makes network calls.
	EnvOptIn bool
	// WorkerURL overrides the default worker endpoint. Leave empty to use
	// DefaultWorkerURL. This field exists solely for testing: production
	// code always leaves it empty.
	WorkerURL string
	// ScoutVersion is the semver of the running API build.
	ScoutVersion string
	// DeploymentType is "docker" or "local".
	DeploymentType string
	// InstanceID is the per-install UUID. Must be non-empty for events to flow.
	InstanceID string
	// HTTPClient is the HTTP client used for flush requests. If nil,
	// http.DefaultClient is used.
	HTTPClient *http.Client
	// ConsentChecker returns the current UI consent value ("granted",
	// "declined", or "unset"). Called before each flush. When the return
	// value is not "granted", the flush is skipped.
	ConsentChecker func() string
}

// Recorder queues telemetry events in memory and flushes them in batches
// to the remote worker endpoint. It is safe for concurrent use.
type Recorder struct {
	cfg       RecorderConfig
	mu        sync.Mutex
	queue     []Event
	client    *http.Client
	workerURL string
	stop      chan struct{}
	done      chan struct{}
}

// NewRecorder creates and starts a Recorder. When cfg.EnvOptIn is false,
// the recorder is a no-op: Record is a no-op, no goroutines are started,
// and no network calls are made.
func NewRecorder(cfg RecorderConfig) *Recorder {
	workerURL := cfg.WorkerURL
	if workerURL == "" {
		workerURL = DefaultWorkerURL
	}
	r := &Recorder{
		cfg:       cfg,
		queue:     make([]Event, 0, maxBatchSize),
		client:    cfg.HTTPClient,
		workerURL: workerURL,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	if r.client == nil {
		r.client = http.DefaultClient
	}
	if !cfg.EnvOptIn {
		// No-op mode: close done so Shutdown returns immediately.
		close(r.done)
		return r
	}
	go r.loop()
	return r
}

// Record enqueues a single event. It is a no-op when EnvOptIn is false,
// when the event name is not in the allow-list, or when the queue is full.
func (r *Recorder) Record(name EventName) {
	if !r.cfg.EnvOptIn {
		return
	}
	if !IsValidEventName(name) {
		return
	}
	evt := NewEvent(name, r.cfg.ScoutVersion, r.cfg.DeploymentType, detectOSPlatform(), "server")
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.queue) >= maxBatchSize {
		// Drop oldest to prevent unbounded growth.
		r.queue = r.queue[1:]
	}
	r.queue = append(r.queue, evt)
}

// Shutdown stops the background loop, performs one best-effort flush with
// a 5-second deadline, and returns. Events still in the queue after the
// deadline are dropped.
func (r *Recorder) Shutdown() {
	if !r.cfg.EnvOptIn {
		return
	}
	close(r.stop)
	<-r.done
}

// loop runs the background flush and ping tickers.
func (r *Recorder) loop() {
	defer close(r.done)
	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-r.stop:
			r.flush()
			return
		case <-flushTicker.C:
			r.flush()
		case <-pingTicker.C:
			r.Record(EventInstancePing)
			r.flush()
		}
	}
}

// flush drains the queue and sends one batch to the worker. It is a no-op
// when the queue is empty, the instance ID is empty, or consent is not granted.
func (r *Recorder) flush() {
	r.mu.Lock()
	if len(r.queue) == 0 {
		r.mu.Unlock()
		return
	}
	batch := make([]Event, len(r.queue))
	copy(batch, r.queue)
	r.queue = r.queue[:0]
	r.mu.Unlock()

	if r.cfg.InstanceID == "" {
		return
	}
	if r.cfg.ConsentChecker != nil && r.cfg.ConsentChecker() != "granted" {
		return
	}

	wire := Batch{
		InstanceID: r.cfg.InstanceID,
		Events:     batch,
	}
	body, err := json.Marshal(wire)
	if err != nil {
		log.Printf("telemetry: marshal error: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.workerURL+"/v1/events", bytes.NewReader(body))
	if err != nil {
		log.Printf("telemetry: request build error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		log.Printf("telemetry: flush error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		// Transient: re-enqueue once. If the queue has grown, drop.
		r.mu.Lock()
		if len(r.queue) == 0 {
			r.queue = batch
		}
		r.mu.Unlock()
	}
	// 4xx and 204 are terminal for this batch.
}

// detectOSPlatform returns the runtime OS as a telemetry-safe string.
func detectOSPlatform() string {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		return runtime.GOOS
	default:
		return "unknown"
	}
}

// DetectDeploymentType returns "docker" when SCOUT_ONBOARDING_MODE=docker,
// otherwise "local".
func DetectDeploymentType() string {
	if os.Getenv("SCOUT_ONBOARDING_MODE") == "docker" {
		return "docker"
	}
	return "local"
}
