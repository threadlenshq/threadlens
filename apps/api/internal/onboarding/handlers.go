package onboarding

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/telemetry"
)

// MountRoutes registers the onboarding HTTP endpoints on r under the prefix
// /api/onboarding. It does not modify any global state and does not wire into
// the main application router — that is left to the server bootstrap.
func MountRoutes(r chi.Router, svc ServiceIface, rec *telemetry.Recorder) {
	r.Get("/api/onboarding/status", handleStatus(svc))
	r.Post("/api/onboarding/save", handleSave(svc, rec))
	r.Post("/api/onboarding/reset", handleReset(svc))
	r.Post("/api/onboarding/steps/{step}", handleSaveRequiredStep(svc))
	r.Post("/api/onboarding/required-step", handleRequiredStep(svc))
	r.Post("/api/onboarding/exploration", handleUpdateExploration(svc))
	r.Post("/api/onboarding/starter-project", handleCreateStarterProject(svc))
	r.Post("/api/onboarding/test-ai", handleTestAI())
}

// ── GET /api/onboarding/status ────────────────────────────────────────────────

func handleStatus(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := svc.GetStatus(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to retrieve onboarding status")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, status)
	}
}

// ── POST /api/onboarding/save ─────────────────────────────────────────────────

type saveRequest struct {
	Values map[string]string `json:"values"`
}

func handleSave(svc ServiceIface, rec *telemetry.Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req saveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if len(req.Values) == 0 {
			httpx.WriteError(w, http.StatusBadRequest, "values must not be empty")
			return
		}
		for _, v := range req.Values {
			if strings.TrimSpace(v) == "" {
				httpx.WriteError(w, http.StatusBadRequest, "all values must be non-blank")
				return
			}
		}

		if err := svc.Save(r.Context(), req.Values); err != nil {
			if errors.Is(err, ErrDisabled) {
				httpx.WriteError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			if rec != nil {
				rec.Record(telemetry.EventErrorOnboardingSave)
			}
			httpx.WriteError(w, http.StatusInternalServerError, "failed to save configuration")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
			httpx.WriteError(w, http.StatusInternalServerError, "failed to reset onboarding state")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if strings.TrimSpace(string(req.Step)) == "" {
			httpx.WriteError(w, http.StatusBadRequest, "step is required")
			return
		}
		if !stringInRequiredSteps(req.Step) {
			httpx.WriteError(w, http.StatusBadRequest, "unknown step")
			return
		}
		status, err := svc.SaveRequiredStep(r.Context(), req.Step, req.Values)
		if err != nil {
			if errors.Is(err, ErrDisabled) {
				httpx.WriteError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "failed to save required step")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, status)
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
			httpx.WriteError(w, http.StatusBadRequest, "unknown step")
			return
		}

		var req saveRequiredStepRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		status, err := svc.SaveRequiredStep(r.Context(), step, req.Values)
		if err != nil {
			if errors.Is(err, ErrDisabled) {
				httpx.WriteError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "failed to save required step")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, status)
	}
}

// ── POST /api/onboarding/exploration ──────────────────────────────────────────

func handleUpdateExploration(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ExplorationUpdate
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		status, err := svc.UpdateExploration(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrDisabled) {
				httpx.WriteError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "failed to update exploration")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, status)
	}
}

// ── POST /api/onboarding/starter-project ──────────────────────────────────────

func handleCreateStarterProject(svc ServiceIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req StarterProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.ProjectName) == "" || strings.TrimSpace(req.Query) == "" {
			httpx.WriteError(w, http.StatusBadRequest, "projectId, projectName and query are required")
			return
		}

		result, err := svc.CreateStarterProject(r.Context(), req)
		if err != nil {
			if errors.Is(err, ErrDisabled) {
				httpx.WriteError(w, http.StatusForbidden, "onboarding is disabled")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "failed to create starter project")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"project":        result.Project,
			"query":          result.Query,
			"createdProject": result.CreatedProject,
			"createdQuery":   result.CreatedQuery,
		})
	}
}
