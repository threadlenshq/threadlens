package usage

import (
	"context"
	"sync"
	"time"
)

type Event struct {
	TenantID   string    `json:"tenantId"`
	ActorID    string    `json:"actorId"`
	ProjectID  string    `json:"projectId,omitempty"`
	TaskID     string    `json:"taskId"`
	ModelID    string    `json:"modelId,omitempty"`
	Operation  string    `json:"operation"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	Units      int64     `json:"units"`
	RecordedAt time.Time `json:"recordedAt"`
}

type Meter interface {
	Record(ctx context.Context, event Event) error
}

type NoopMeter struct{}

func (NoopMeter) Record(context.Context, Event) error {
	return nil
}

type MemoryMeter struct {
	mu     sync.Mutex
	events []Event
}

func NewMemoryMeter() *MemoryMeter {
	return &MemoryMeter{}
}

func (m *MemoryMeter) Record(_ context.Context, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if event.RecordedAt.IsZero() {
		event.RecordedAt = time.Now().UTC()
	}
	m.events = append(m.events, event)
	return nil
}

func (m *MemoryMeter) Events() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event(nil), m.events...)
}
