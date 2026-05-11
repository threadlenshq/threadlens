package bridge

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type cachedStatus struct {
	status    RuntimeStatus
	fetchedAt time.Time
}

// Registry holds a set of runtimes and caches detection results.
type Registry struct {
	runtimes map[string]Runtime
	cacheTTL time.Duration

	mu    sync.Mutex
	cache map[string]cachedStatus
}

// NewRegistry creates a Registry keyed by each Runtime's ID.
func NewRegistry(cacheTTL time.Duration, runtimes ...Runtime) *Registry {
	r := &Registry{
		runtimes: make(map[string]Runtime, len(runtimes)),
		cacheTTL: cacheTTL,
		cache:    make(map[string]cachedStatus, len(runtimes)),
	}
	for _, rt := range runtimes {
		r.runtimes[rt.ID()] = rt
	}
	return r
}

// DefaultRegistry returns a registry with no runtimes registered.
func DefaultRegistry() *Registry {
	return NewRegistry(5 * time.Minute)
}

func (r *Registry) detect(ctx context.Context, id string, rt Runtime) RuntimeStatus {
	r.mu.Lock()
	if r.cacheTTL > 0 {
		if c, ok := r.cache[id]; ok && time.Since(c.fetchedAt) < r.cacheTTL {
			r.mu.Unlock()
			return c.status
		}
	}
	r.mu.Unlock()

	status := rt.Detect(ctx)

	r.mu.Lock()
	r.cache[id] = cachedStatus{status: status, fetchedAt: time.Now()}
	r.mu.Unlock()

	return status
}

// AvailableRuntimeIDs returns sorted IDs of runtimes that are currently available.
func (r *Registry) AvailableRuntimeIDs(ctx context.Context) []string {
	var ids []string
	for id, rt := range r.runtimes {
		if s := r.detect(ctx, id, rt); s.Available {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// Generate dispatches to the runtime identified by req.Provider.
func (r *Registry) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	rt, ok := r.runtimes[req.Provider]
	if !ok {
		return "", fmt.Errorf("bridge: unknown runtime %q", req.Provider)
	}
	status := r.detect(ctx, req.Provider, rt)
	if !status.Available {
		msg := status.Message
		if msg == "" {
			msg = "not available"
		}
		return "", fmt.Errorf("bridge: runtime %q unavailable: %s", req.Provider, msg)
	}
	text, err := rt.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("bridge: runtime %q error: %w", req.Provider, err)
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("bridge: runtime %q returned empty response", req.Provider)
	}
	return text, nil
}
