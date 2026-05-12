package onboarding

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// MountRoutes registers the onboarding HTTP endpoints on r under the prefix
// /api/onboarding. It does not modify any global state and does not wire into
// the main application router — that is left to Task 10.
func MountRoutes(r chi.Router, svc ServiceIface) {
	r.Get("/api/onboarding/status", handleStatus(svc))
	r.Post("/api/onboarding/save", handleSave(svc))
	r.Post("/api/onboarding/reset", handleReset(svc))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ── GET /api/onboarding/status ────────────────────────────────────────────────

func handleStatus(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := svc.GetStatus(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to retrieve onboarding status")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":  status.Enabled,
			"complete": status.Complete,
		})
	}
}

// ── POST /api/onboarding/save ─────────────────────────────────────────────────

type saveRequest struct {
	Values map[string]string `json:"values"`
}

func handleSave(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req saveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if len(req.Values) == 0 {
			writeError(w, http.StatusBadRequest, "values must not be empty")
			return
		}
		for _, v := range req.Values {
			if strings.TrimSpace(v) == "" {
				writeError(w, http.StatusBadRequest, "all values must be non-blank")
				return
			}
		}

		if err := svc.Save(r.Context(), req.Values); err != nil {
			if errors.Is(err, ErrDisabled) {
				writeError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to save configuration")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// ── POST /api/onboarding/reset ────────────────────────────────────────────────

func handleReset(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Reset(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to reset onboarding state")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
