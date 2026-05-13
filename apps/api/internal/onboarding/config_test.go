package onboarding_test

// config_test.go defines the expected behaviour of the onboarding Config
// loader before the implementation exists.  All tests in this file are
// intentionally failing until onboarding.LoadConfig (and the Config type) are
// implemented.
//
// Design constraints kept here:
//   - Docker mode is active only when SCOUT_ONBOARDING_MODE=docker.
//   - Onboarding can be fully disabled via SCOUT_ONBOARDING_DISABLE=1.
//   - When Docker mode is active and SCOUT_ONBOARDING_ENV_FILE is unset or
//     empty, LoadConfig falls back to the default writable container path
//     /data/.env.
//   - The config exposes CompletionKey so callers can persist the "done" flag
//     via the settings repository without hard-coding the key elsewhere.

import (
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
)

// completionKey is the settings-repository key agreed in Task 3.
const completionKey = "onboarding.complete"

// stateKey is the versioned settings-repository key for ThreadLens v1.
const stateKey = "onboarding.threadlens.v1"

// defaultEnvFilePath is the writable path inside the open-core container.
const defaultEnvFilePath = "/data/.env"

// ── 1. Docker mode ────────────────────────────────────────────────────────────

func TestDockerModeEnabledWhenEnvSet(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_MODE", "docker")
	t.Setenv("SCOUT_ONBOARDING_ENV_FILE", "/data/.env")

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.DockerMode {
		t.Error("DockerMode should be true when SCOUT_ONBOARDING_MODE=docker")
	}
}

func TestDockerModeDisabledWhenEnvUnset(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_MODE", "")
	t.Setenv("SCOUT_ONBOARDING_ENV_FILE", "")

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DockerMode {
		t.Error("DockerMode should be false when SCOUT_ONBOARDING_MODE is not set")
	}
}

func TestDockerModeDisabledForArbitraryValue(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_MODE", "native")

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DockerMode {
		t.Error("DockerMode should be false for SCOUT_ONBOARDING_MODE=native")
	}
}

// ── 2. Disabled flag ──────────────────────────────────────────────────────────

func TestDisabledWhenEnvSet(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_DISABLE", "1")

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Disabled {
		t.Error("Disabled should be true when SCOUT_ONBOARDING_DISABLE=1")
	}
}

func TestEnabledWhenDisableEnvUnset(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_DISABLE", "")

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Disabled {
		t.Error("Disabled should be false when SCOUT_ONBOARDING_DISABLE is not set")
	}
}

func TestEnabledWhenDisableEnvZero(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_DISABLE", "0")

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Disabled {
		t.Error("Disabled should be false when SCOUT_ONBOARDING_DISABLE=0")
	}
}

// ── 3. Env-file path ──────────────────────────────────────────────────────────

func TestEnvFilePathExposedFromEnv(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_MODE", "docker")
	t.Setenv("SCOUT_ONBOARDING_ENV_FILE", "/data/.env")

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.EnvFilePath != "/data/.env" {
		t.Errorf("EnvFilePath = %q; want /data/.env", cfg.EnvFilePath)
	}
}

func TestEnvFilePathDefaultInDockerMode(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_MODE", "docker")
	t.Setenv("SCOUT_ONBOARDING_ENV_FILE", "") // explicitly unset

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error when env file path defaults: %v", err)
	}
	if cfg.EnvFilePath != defaultEnvFilePath {
		t.Errorf("EnvFilePath = %q; want default %q", cfg.EnvFilePath, defaultEnvFilePath)
	}
}

// ── 4. Missing env-file path is an error only in Docker mode ─────────────────

func TestMissingEnvFilePathIsNotErrorInNativeMode(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_MODE", "")
	t.Setenv("SCOUT_ONBOARDING_ENV_FILE", "")

	_, err := onboarding.LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig should not error in native mode without env file: %v", err)
	}
}

// An explicitly empty string in Docker mode should still resolve to the default.
func TestExplicitlyEmptyEnvFilePathUsesDefaultInDockerMode(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_MODE", "docker")
	t.Setenv("SCOUT_ONBOARDING_ENV_FILE", "")

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.EnvFilePath != defaultEnvFilePath {
		t.Errorf("EnvFilePath = %q; want default %q", cfg.EnvFilePath, defaultEnvFilePath)
	}
}

// ── 5. Config fields ──────────────────────────────────────────────────────────

func TestCompletionKeyIsFixed(t *testing.T) {
	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CompletionKey != completionKey {
		t.Errorf("CompletionKey = %q; want %q", cfg.CompletionKey, completionKey)
	}
}

func TestStateKeyIsFixed(t *testing.T) {
	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.StateKey != stateKey {
		t.Errorf("StateKey = %q; want %q", cfg.StateKey, stateKey)
	}
}

func TestConfigFieldsPresent(t *testing.T) {
	t.Setenv("SCOUT_ONBOARDING_MODE", "docker")
	t.Setenv("SCOUT_ONBOARDING_ENV_FILE", "/custom/.env")

	cfg, err := onboarding.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All expected exported fields must be accessible.
	_ = cfg.DockerMode    // bool
	_ = cfg.Disabled      // bool
	_ = cfg.EnvFilePath   // string
	_ = cfg.CompletionKey // string
}
