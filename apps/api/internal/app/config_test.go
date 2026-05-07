package app

import (
	"strings"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("SCOUT_DB_PATH", "")
	t.Setenv("THREADLENS_RUNTIME_MODE", "")
	cfg := LoadConfig()
	if cfg.Port != "4749" {
		t.Fatalf("Port = %q, want 4749", cfg.Port)
	}
	if cfg.DBPath == "" {
		t.Fatal("DBPath must not be empty")
	}
	if !strings.HasSuffix(cfg.DBPath, "scout.db") {
		t.Fatalf("DBPath = %q, want path ending in scout.db", cfg.DBPath)
	}
	if cfg.RuntimeMode != entitlements.RuntimeModeSelfHosted {
		t.Fatalf("RuntimeMode = %q, want %q", cfg.RuntimeMode, entitlements.RuntimeModeSelfHosted)
	}
}

func TestLoadConfigRuntimeModeHosted(t *testing.T) {
	t.Setenv("THREADLENS_RUNTIME_MODE", "hosted")
	cfg := LoadConfig()
	if cfg.RuntimeMode != entitlements.RuntimeModeHosted {
		t.Fatalf("RuntimeMode = %q, want %q", cfg.RuntimeMode, entitlements.RuntimeModeHosted)
	}
}

func TestLoadConfigRuntimeModeInvalidFallback(t *testing.T) {
	t.Setenv("THREADLENS_RUNTIME_MODE", "invalid_value")
	cfg := LoadConfig()
	if cfg.RuntimeMode != entitlements.RuntimeModeSelfHosted {
		t.Fatalf("RuntimeMode = %q, want %q (fallback for invalid)", cfg.RuntimeMode, entitlements.RuntimeModeSelfHosted)
	}
}

func TestLoadConfigEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("SCOUT_DB_PATH", "/tmp/scout-test.db")
	cfg := LoadConfig()
	if cfg.Port != "9999" {
		t.Fatalf("Port = %q, want 9999", cfg.Port)
	}
	if cfg.DBPath != "/tmp/scout-test.db" {
		t.Fatalf("DBPath = %q", cfg.DBPath)
	}
}
