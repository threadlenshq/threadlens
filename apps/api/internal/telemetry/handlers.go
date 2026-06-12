package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
)

// Settings keys used by the telemetry consent system.
const (
	SettingsKeyInstanceID       = "telemetry.instance_id"
	SettingsKeyUIChoice         = "telemetry.consent.ui_choice"
	SettingsKeyPopupDismissedAt = "telemetry.consent.popup_dismissed_at"
)

// MountRoutes registers the telemetry consent API routes on r.
func MountRoutes(r chi.Router, repo *settings.Repository, recorder *Recorder, cfg TelemetryStatusConfig) {
	r.Get("/api/telemetry/status", handleStatus(repo, cfg))
	r.Post("/api/telemetry/consent", handleConsent(repo, recorder))
	r.Post("/api/telemetry/popup-dismissed", handlePopupDismissed(repo))
	r.Post("/api/telemetry/reset-consent", handleResetConsent(repo))
}

// TelemetryStatusConfig holds the static values returned by the status endpoint.
type TelemetryStatusConfig struct {
	EnvOptIn       bool
	ScoutVersion   string
	DeploymentType string
	OSPlatform     string
}

// handleStatus returns the current effective telemetry state.
func handleStatus(repo *settings.Repository, cfg TelemetryStatusConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		instanceID, _, _ := repo.Get(ctx, SettingsKeyInstanceID)
		uiChoice, _, _ := repo.Get(ctx, SettingsKeyUIChoice)
		popupDismissedAt, _, _ := repo.Get(ctx, SettingsKeyPopupDismissedAt)

		if uiChoice == "" {
			uiChoice = "unset"
		}

		resp := map[string]any{
			"instance_id":        instanceID,
			"env_opt_in":         cfg.EnvOptIn,
			"ui_consent":         uiChoice,
			"popup_dismissed_at": popupDismissedAt,
			"scout_version":      cfg.ScoutVersion,
			"deployment_type":    cfg.DeploymentType,
			"os_platform":        cfg.OSPlatform,
		}
		httpx.WriteJSON(w, http.StatusOK, resp)
	}
}

// handleConsent persists the user's UI consent choice and emits a heartbeat.
func handleConsent(repo *settings.Repository, recorder *Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Choice string `json:"choice"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if body.Choice != "granted" && body.Choice != "declined" {
			httpx.WriteError(w, http.StatusBadRequest, "choice must be 'granted' or 'declined'")
			return
		}

		if err := repo.Set(r.Context(), SettingsKeyUIChoice, body.Choice); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to save consent")
			return
		}

		// Emit consent-changed heartbeat.
		if recorder != nil {
			recorder.Record(EventInstanceConsentChanged)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// handlePopupDismissed records that the bottom-left popup has been answered.
func handlePopupDismissed(repo *settings.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ts := time.Now().UTC().Format(time.RFC3339)
		if err := repo.Set(r.Context(), SettingsKeyPopupDismissedAt, ts); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to record popup dismissal")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleResetConsent clears both ui_consent and popup_dismissed_at so the
// bottom-left toast reappears on next page load.
func handleResetConsent(repo *settings.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := repo.Delete(ctx, SettingsKeyUIChoice); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to clear consent")
			return
		}
		if err := repo.Delete(ctx, SettingsKeyPopupDismissedAt); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "failed to clear popup dismissal")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ReadConsentChoice reads the current UI consent value from app_settings.
// Returns "unset" when no choice has been recorded.
func ReadConsentChoice(repo *settings.Repository) string {
	val, found, err := repo.Get(context.Background(), SettingsKeyUIChoice)
	if err != nil || !found || val == "" {
		return "unset"
	}
	return val
}
