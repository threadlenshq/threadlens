package app

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
)

type Config struct {
	Port         string
	DBPath       string
	FrontendDist string
	RuntimeMode  entitlements.RuntimeMode
	Location     *time.Location
	Telemetry    TelemetryConfig
}

// TelemetryConfig holds the runtime telemetry gating state.
type TelemetryConfig struct {
	// EnvOptIn is true only when SCOUT_TELEMETRY_OPT_IN=1.
	EnvOptIn bool
}

func LoadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4749"
	}

	dbPath := os.Getenv("SCOUT_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Clean(filepath.Join("..", "..", "scout.db"))
	}

	frontendDist := os.Getenv("SCOUT_FRONTEND_DIST")
	if frontendDist == "" {
		frontendDist = filepath.Clean(filepath.Join("..", "web", "dist"))
	}

	runtimeMode := entitlements.RuntimeMode(os.Getenv("THREADLENS_RUNTIME_MODE"))
	if runtimeMode != entitlements.RuntimeModeSelfHosted && runtimeMode != entitlements.RuntimeModeHosted {
		runtimeMode = entitlements.RuntimeModeSelfHosted
	}

	loc := resolveLocation()

	telemetry := TelemetryConfig{
		EnvOptIn: os.Getenv("SCOUT_TELEMETRY_OPT_IN") == "1",
	}

	return Config{Port: port, DBPath: dbPath, FrontendDist: frontendDist, RuntimeMode: runtimeMode, Location: loc, Telemetry: telemetry}
}

// resolveLocation returns the timezone to use for the cron scheduler.
//
// Priority:
//  1. SCOUT_TIMEZONE env var (explicit override, any IANA name e.g. "America/New_York")
//  2. Auto-detect from the host's /etc/localtime symlink (works on Linux & macOS)
//  3. Fall back to UTC
func resolveLocation() *time.Location {
	if tz := os.Getenv("SCOUT_TIMEZONE"); tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			log.Printf("config: invalid SCOUT_TIMEZONE %q, falling back to UTC: %v", tz, err)
			return time.UTC
		}
		log.Printf("config: scheduler timezone set to %s (from SCOUT_TIMEZONE)", loc)
		return loc
	}

	// Auto-detect: /etc/localtime is a symlink like
	// /usr/share/zoneinfo/Australia/Brisbane on Linux/macOS.
	if target, err := os.Readlink("/etc/localtime"); err == nil {
		// Strip everything up to and including "zoneinfo/"
		const marker = "zoneinfo/"
		if idx := findLast(target, marker); idx >= 0 {
			name := target[idx+len(marker):]
			if loc, err := time.LoadLocation(name); err == nil {
				log.Printf("config: scheduler timezone auto-detected as %s (from /etc/localtime)", loc)
				return loc
			}
		}
	}

	log.Printf("config: scheduler timezone defaulting to UTC (set SCOUT_TIMEZONE to override)")
	return time.UTC
}

// findLast returns the index of the last occurrence of substr in s, or -1.
func findLast(s, substr string) int {
	idx := -1
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			idx = i
		}
	}
	return idx
}
