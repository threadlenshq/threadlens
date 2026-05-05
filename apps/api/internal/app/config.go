package app

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port         string
	DBPath       string
	FrontendDist string
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

	return Config{Port: port, DBPath: dbPath, FrontendDist: frontendDist}
}
