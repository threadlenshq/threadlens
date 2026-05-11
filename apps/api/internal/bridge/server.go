package bridge

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// ServerConfig configures the bridge HTTP daemon.
type ServerConfig struct {
	// BearerToken is the shared secret required in every request's Authorization header.
	BearerToken string
	// Registry is the runtime registry used for health and generate. Defaults to
	// DefaultRegistry() when nil.
	Registry *Registry
	// MaxBodyBytes is the maximum request body size in bytes. Defaults to 1 MiB
	// when <= 0.
	MaxBodyBytes int64
	// RequestTimeout is the per-request context deadline. Defaults to 5 minutes
	// when <= 0.
	RequestTimeout time.Duration
}

// NewHandler returns an http.Handler implementing the bridge daemon API.
func NewHandler(cfg ServerConfig) http.Handler {
	if cfg.Registry == nil {
		cfg.Registry = DefaultRegistry()
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 1 << 20
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 5 * time.Minute
	}

	mux := http.NewServeMux()
	s := &bridgeServer{cfg: cfg}
	mux.HandleFunc("/v1/health", s.health)
	mux.HandleFunc("/v1/generate", s.generate)
	return mux
}

type bridgeServer struct {
	cfg ServerConfig
}

func (s *bridgeServer) auth(r *http.Request) bool {
	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(hdr, "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.BearerToken)) == 1
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *bridgeServer) health(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()

	ids := s.cfg.Registry.AvailableRuntimeIDs(ctx)
	runtimes := make([]RuntimeStatus, 0, len(ids))
	for _, id := range ids {
		rt := s.cfg.Registry.runtimes[id]
		runtimes = append(runtimes, s.cfg.Registry.detect(ctx, id, rt))
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		OK:       true,
		Runtimes: runtimes,
	})
}

func (s *bridgeServer) generate(w http.ResponseWriter, r *http.Request) {
	if !s.auth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()

	// Enforce body size limit before JSON parsing so malformed large bodies
	// return 413 rather than 400.
	lr := &io.LimitedReader{R: r.Body, N: s.cfg.MaxBodyBytes + 1}
	body, err := io.ReadAll(lr)
	if lr.N == 0 {
		// More bytes remain — body exceeded limit.
		http.Error(w, "request entity too large", http.StatusRequestEntityTooLarge)
		return
	}
	if err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	var req GenerateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	text, err := s.cfg.Registry.Generate(ctx, req)
	if err != nil {
		http.Error(w, "bad gateway: "+err.Error(), http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, GenerateResponse{Text: strings.TrimSpace(text)})
}
