package app

import (
	"os"
	"path/filepath"

	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
)

type Config struct {
	Port         string
	DBPath       string
	FrontendDist string
	RuntimeMode  entitlements.RuntimeMode
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

	return Config{Port: port, DBPath: dbPath, FrontendDist: frontendDist, RuntimeMode: runtimeMode}
}
