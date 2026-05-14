package onboarding

import (
	"os"
	"strings"
)

const (
	defaultEnvFilePath = "/data/.env"
	completionKey      = "onboarding.complete"
	stateKey           = "onboarding.threadlens.v1"
)

// Config holds the resolved onboarding configuration derived from environment
// variables. Values are resolved at load time and should not be mutated.
type Config struct {
	// DockerMode is true only when SCOUT_ONBOARDING_MODE=docker.
	DockerMode bool

	// Disabled is true when SCOUT_ONBOARDING_DISABLE=1.
	Disabled bool

	// EnvFilePath is the writable .env file path used in Docker mode.
	// Defaults to /data/.env when SCOUT_ONBOARDING_ENV_FILE is unset or empty.
	EnvFilePath string

	// DBPath is the resolved path to the SQLite database file. Injected by the
	// caller after app config is loaded; not read from environment variables.
	DBPath string

	// CompletionKey is the settings-repository key used to persist the
	// "onboarding complete" flag.  Fixed at "onboarding.complete".
	CompletionKey string

	// StateKey is the versioned settings-repository key for ThreadLens v1
	// onboarding progress. Fixed at "onboarding.threadlens.v1".
	StateKey string
}

// LoadConfig reads the onboarding configuration from environment variables and
// returns a fully-resolved Config.  It never returns an error under currently
// defined constraints (the default env-file path covers missing values in
// Docker mode), but the signature is kept error-returning so callers don't need
// updating if validation is added later.
func LoadConfig() (Config, error) {
	dockerMode := strings.TrimSpace(os.Getenv("SCOUT_ONBOARDING_MODE")) == "docker"

	envFilePath := strings.TrimSpace(os.Getenv("SCOUT_ONBOARDING_ENV_FILE"))
	if dockerMode && envFilePath == "" {
		envFilePath = defaultEnvFilePath
	}

	disabled := strings.TrimSpace(os.Getenv("SCOUT_ONBOARDING_DISABLE")) == "1"

	return Config{
		DockerMode:    dockerMode,
		Disabled:      disabled,
		EnvFilePath:   envFilePath,
		CompletionKey: completionKey,
		StateKey:      stateKey,
	}, nil
}
