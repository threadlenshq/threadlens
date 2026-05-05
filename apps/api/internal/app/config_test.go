package app

import (
	"strings"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("SCOUT_DB_PATH", "")
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
