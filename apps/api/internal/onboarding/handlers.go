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
// the main application router — that is left to the server bootstrap.
func MountRoutes(r chi.Router, svc ServiceIface) {
	r.Get("/api/onboarding/status", handleStatus(svc))
	r.Post("/api/onboarding/save", handleSave(svc))
	r.Post("/api/onboarding/reset", handleReset(svc))
	r.Post("/api/onboarding/steps/{step}", handleSaveRequiredStep(svc))
	r.Post("/api/onboarding/required-step", handleRequiredStep(svc))
	r.Post("/api/onboarding/exploration", handleUpdateExploration(svc))
	r.Post("/api/onboarding/starter-project", handleCreateStarterProject(svc))
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
			"enabled":                status.Enabled,
			"complete":               status.Complete,
			"requiredSetupComplete":  status.RequiredSetupComplete,
			"explorationComplete":    status.ExplorationComplete,
			"phase":                  status.Phase,
			"currentRequiredStep":    status.CurrentRequiredStep,
			"currentExplorationItem": status.CurrentExplorationItem,
			"steps":                  status.Steps,
			"items":                  status.Items,
			"capabilities":           status.Capabilities,
			"appDatabase":            status.AppDatabase,
			"context":                status.Context,
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

type resetRequest struct {
	Mode ResetMode `json:"mode"`
}

func handleReset(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req resetRequest
		if r.Body != nil && r.ContentLength != 0 {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		mode := req.Mode
		if mode == "" {
			mode = ResetModeProgress
		}
		if err := svc.Reset(r.Context(), mode); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to reset onboarding state")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// ── POST /api/onboarding/required-step ───────────────────────────────────────

type requiredStepRequest struct {
	Step   RequiredStep      `json:"step"`
	Values map[string]string `json:"values"`
}

func handleRequiredStep(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req requiredStepRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if strings.TrimSpace(string(req.Step)) == "" {
			writeError(w, http.StatusBadRequest, "step is required")
			return
		}
		if !stringInRequiredSteps(req.Step) {
			writeError(w, http.StatusBadRequest, "unknown step")
			return
		}
		status, err := svc.SaveRequiredStep(r.Context(), req.Step, req.Values)
		if err != nil {
			if errors.Is(err, ErrDisabled) {
				writeError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to save required step")
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

// ── POST /api/onboarding/steps/{step} ─────────────────────────────────────────

type saveRequiredStepRequest struct {
	Values map[string]string `json:"values"`
}

func handleSaveRequiredStep(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stepStr := chi.URLParam(r, "step")
		step := RequiredStep(stepStr)
		if !stringInRequiredSteps(step) {
			writeError(w, http.StatusBadRequest, "unknown step")
			return
		}

		var req saveRequiredStepRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		status, err := svc.SaveRequiredStep(r.Context(), step, req.Values)
		if err != nil {
			if errors.Is(err, ErrDisabled) {
				writeError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to save required step")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":               status.Enabled,
			"complete":              status.Complete,
			"requiredSetupComplete": status.RequiredSetupComplete,
			"explorationComplete":   status.ExplorationComplete,
			"phase":                 status.Phase,
			"currentRequiredStep":   status.CurrentRequiredStep,
			"steps":                 status.Steps,
			"items":                 status.Items,
		})
	}
}

// ── POST /api/onboarding/exploration ──────────────────────────────────────────

func handleUpdateExploration(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ExplorationUpdate
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		status, err := svc.UpdateExploration(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrDisabled) {
				writeError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update exploration")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":             status.Enabled,
			"complete":            status.Complete,
			"explorationComplete": status.ExplorationComplete,
			"phase":               status.Phase,
			"items":               status.Items,
		})
	}
}

// ── POST /api/onboarding/starter-project ──────────────────────────────────────

func handleCreateStarterProject(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req StarterProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.ProjectName) == "" || strings.TrimSpace(req.Query) == "" {
			writeError(w, http.StatusBadRequest, "projectId, projectName and query are required")
			return
		}

		result, err := svc.CreateStarterProject(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrDisabled) {
				writeError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to create starter project")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"project":        result.Project,
			"query":          result.Query,
			"createdProject": result.CreatedProject,
			"createdQuery":   result.CreatedQuery,
		})
	}
}
