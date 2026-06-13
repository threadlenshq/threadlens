package telemetry

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE app_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at TEXT DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestHandleStatus_Defaults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := settings.NewRepository(db)

	r := chi.NewRouter()
	MountRoutes(r, repo, nil, TelemetryStatusConfig{
		OptInMode:      "disabled",
		ScoutVersion:   "0.7.2",
		DeploymentType: "local",
		OSPlatform:     "darwin",
	})

	req := httptest.NewRequest("GET", "/api/telemetry/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["env_opt_in"] != "disabled" {
		t.Error("expected env_opt_in to be 'disabled'")
	}
	if resp["ui_consent"] != "unset" {
		t.Errorf("expected ui_consent 'unset', got %v", resp["ui_consent"])
	}
	if resp["instance_id"] != "" {
		t.Errorf("expected empty instance_id, got %v", resp["instance_id"])
	}
}

func TestHandleConsent_Granted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := settings.NewRepository(db)

	r := chi.NewRouter()
	MountRoutes(r, repo, nil, TelemetryStatusConfig{OptInMode: "consent"})

	body := `{"choice":"granted"}`
	req := httptest.NewRequest("POST", "/api/telemetry/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	val, found, _ := repo.Get(context.Background(), SettingsKeyUIChoice)
	if !found || val != "granted" {
		t.Errorf("expected 'granted', got %q (found=%v)", val, found)
	}
}

func TestHandleConsent_Declined(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := settings.NewRepository(db)

	r := chi.NewRouter()
	MountRoutes(r, repo, nil, TelemetryStatusConfig{OptInMode: "consent"})

	body := `{"choice":"declined"}`
	req := httptest.NewRequest("POST", "/api/telemetry/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	val, found, _ := repo.Get(context.Background(), SettingsKeyUIChoice)
	if !found || val != "declined" {
		t.Errorf("expected 'declined', got %q (found=%v)", val, found)
	}
}

func TestHandleConsent_InvalidChoice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := settings.NewRepository(db)

	r := chi.NewRouter()
	MountRoutes(r, repo, nil, TelemetryStatusConfig{OptInMode: "consent"})

	body := `{"choice":"maybe"}`
	req := httptest.NewRequest("POST", "/api/telemetry/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlePopupDismissed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := settings.NewRepository(db)

	r := chi.NewRouter()
	MountRoutes(r, repo, nil, TelemetryStatusConfig{OptInMode: "consent"})

	req := httptest.NewRequest("POST", "/api/telemetry/popup-dismissed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	val, found, _ := repo.Get(context.Background(), SettingsKeyPopupDismissedAt)
	if !found || val == "" {
		t.Error("expected popup_dismissed_at to be set")
	}
}

func TestHandleStatus_WithConsent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := settings.NewRepository(db)
	_ = repo.Set(context.Background(), SettingsKeyInstanceID, "test-uuid-123")
	_ = repo.Set(context.Background(), SettingsKeyUIChoice, "granted")
	_ = repo.Set(context.Background(), SettingsKeyPopupDismissedAt, "2026-06-12T08:31:00Z")

	r := chi.NewRouter()
	MountRoutes(r, repo, nil, TelemetryStatusConfig{
		OptInMode:      "consent",
		ScoutVersion:   "0.7.2",
		DeploymentType: "docker",
		OSPlatform:     "linux",
	})

	req := httptest.NewRequest("GET", "/api/telemetry/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["instance_id"] != "test-uuid-123" {
		t.Errorf("expected instance_id 'test-uuid-123', got %v", resp["instance_id"])
	}
	if resp["ui_consent"] != "granted" {
		t.Errorf("expected ui_consent 'granted', got %v", resp["ui_consent"])
	}
	if resp["popup_dismissed_at"] != "2026-06-12T08:31:00Z" {
		t.Errorf("unexpected popup_dismissed_at: %v", resp["popup_dismissed_at"])
	}
}

func TestHandleResetConsent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := settings.NewRepository(db)
	_ = repo.Set(context.Background(), SettingsKeyUIChoice, "granted")
	_ = repo.Set(context.Background(), SettingsKeyPopupDismissedAt, "2026-06-12T08:31:00Z")

	r := chi.NewRouter()
	MountRoutes(r, repo, nil, TelemetryStatusConfig{OptInMode: "consent"})

	req := httptest.NewRequest("POST", "/api/telemetry/reset-consent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	_, found, _ := repo.Get(context.Background(), SettingsKeyUIChoice)
	if found {
		t.Error("expected ui_choice to be deleted")
	}
	_, found, _ = repo.Get(context.Background(), SettingsKeyPopupDismissedAt)
	if found {
		t.Error("expected popup_dismissed_at to be deleted")
	}
}
