package ai

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BridgeState describes the result of bridge discovery.
type BridgeState struct {
	// Enabled reports whether the bridge subsystem is not explicitly disabled.
	Enabled bool
	// Detected reports whether a usable bridge was found.
	Detected bool
	// URL is the base URL of the discovered bridge (empty if not detected).
	URL string
	// Token is the bearer token to use (empty if not detected).
	Token string
	// Runtimes is an optional list of runtime identifiers advertised by the bridge config.
	Runtimes []string
	// Message is a human-readable status string. Token values and file paths are never included.
	Message string
	// Source indicates how the bridge was found: "config", "env", "disabled", or "auto-launch".
	Source string
	// AutoLaunchAttempted reports whether a helper binary launch was attempted.
	AutoLaunchAttempted bool
}

// bridgeConfigFile is the JSON structure of ai-bridge.json.
type bridgeConfigFile struct {
	Type       string   `json:"type"`
	URL        string   `json:"url"`
	TokenFile  string   `json:"tokenFile"`
	Runtimes   []string `json:"runtimes"`
	AutoLaunch struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	} `json:"autoLaunch"`
}

// notDetected returns a BridgeState that is enabled but not detected.
func notDetected(msg, source string) BridgeState {
	return BridgeState{Enabled: true, Detected: false, Message: msg, Source: source}
}

// LoadBridgeConfig discovers the host CLI bridge by reading config and/or env vars.
// It never returns an error for "expected" failures (missing file, bad JSON, etc.);
// errors are only returned for unexpected internal problems. Callers should always
// check the returned BridgeState regardless of the error value.
func LoadBridgeConfig() (BridgeState, error) {
	// 1. Explicit disable wins over everything.
	if os.Getenv("SCOUT_AI_BRIDGE_DISABLE") == "1" {
		return BridgeState{
			Enabled:  false,
			Detected: false,
			Message:  "bridge disabled via SCOUT_AI_BRIDGE_DISABLE",
			Source:   "disabled",
		}, nil
	}

	// 2. Full env override: URL + token file supplied directly.
	if envURL := os.Getenv("SCOUT_AI_BRIDGE_URL"); envURL != "" {
		return loadFromEnvOverride(envURL)
	}

	// 3. Locate config file.
	cfgPath := resolveConfigPath()

	state, ok := loadFromConfigFile(cfgPath)
	if ok {
		return state, nil
	}

	// 4. Best-effort auto-launch if SCOUT_AI_BRIDGE_HELPER is set.
	if helper := os.Getenv("SCOUT_AI_BRIDGE_HELPER"); helper != "" {
		return attemptAutoLaunch(helper, cfgPath, state)
	}

	return state, nil
}

// LoadBridgeStatus is a read-only variant of LoadBridgeConfig that never spawns
// processes or waits. It is safe to call from hot paths such as the model catalog
// endpoint. Auto-launch is intentionally skipped; the returned AutoLaunchAttempted
// will always be false.
func LoadBridgeStatus() BridgeState {
	// 1. Explicit disable wins over everything.
	if os.Getenv("SCOUT_AI_BRIDGE_DISABLE") == "1" {
		return BridgeState{
			Enabled:  false,
			Detected: false,
			Message:  "bridge disabled via SCOUT_AI_BRIDGE_DISABLE",
			Source:   "disabled",
		}
	}

	// 2. Full env override: URL + token file supplied directly.
	if envURL := os.Getenv("SCOUT_AI_BRIDGE_URL"); envURL != "" {
		state, _ := loadFromEnvOverride(envURL)
		return state
	}

	// 3. Locate config file (read-only; auto-launch is intentionally skipped).
	cfgPath := resolveConfigPath()
	state, _ := loadFromConfigFile(cfgPath)
	return state
}

// resolveConfigPath returns the path to the bridge config JSON.
// Priority: SCOUT_AI_BRIDGE_CONFIG > $XDG_CONFIG_HOME/scout/ai-bridge.json > ~/.config/scout/ai-bridge.json
func resolveConfigPath() string {
	if p := os.Getenv("SCOUT_AI_BRIDGE_CONFIG"); p != "" {
		return p
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "scout", "ai-bridge.json")
}

// loadFromConfigFile attempts to read and parse the config file at path.
// Returns (state, true) on success, (partial state, false) on failure.
func loadFromConfigFile(cfgPath string) (BridgeState, bool) {
	if cfgPath == "" {
		return notDetected("no config path resolved", "config"), false
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		// Missing file is not an error; just not detected.
		return notDetected("config file not found", "config"), false
	}

	var cfg bridgeConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return notDetected("invalid bridge config JSON", "config"), false
	}

	return buildStateFromConfig(cfg, false)
}

// loadFromEnvOverride builds a BridgeState from SCOUT_AI_BRIDGE_URL and
// SCOUT_AI_BRIDGE_TOKEN_FILE env vars. Public URLs are allowed because the
// user supplied the URL explicitly.
func loadFromEnvOverride(rawURL string) (BridgeState, error) {
	tokenFile := os.Getenv("SCOUT_AI_BRIDGE_TOKEN_FILE")
	if tokenFile == "" {
		return notDetected("SCOUT_AI_BRIDGE_URL set but no SCOUT_AI_BRIDGE_TOKEN_FILE", "env"), nil
	}

	token, err := readToken(tokenFile)
	if err != nil {
		return notDetected("token file unreadable", "env"), nil
	}
	if token == "" {
		return notDetected("token file is empty", "env"), nil
	}

	// Validate the URL is parseable.
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return notDetected("SCOUT_AI_BRIDGE_URL is not a valid URL", "env"), nil
	}

	return BridgeState{
		Enabled:  true,
		Detected: true,
		URL:      rawURL,
		Token:    token,
		Message:  "bridge configured via environment",
		Source:   "env",
	}, nil
}

// buildStateFromConfig validates a parsed config and returns a BridgeState.
// explicitURL indicates the URL was supplied externally (env), so loopback check is skipped.
func buildStateFromConfig(cfg bridgeConfigFile, explicitURL bool) (BridgeState, bool) {
	// Only http-localhost type is supported.
	if cfg.Type != "http-localhost" {
		return notDetected(fmt.Sprintf("unsupported bridge type %q", cfg.Type), "config"), false
	}

	if cfg.URL == "" {
		return notDetected("bridge config missing url", "config"), false
	}

	// Reject public (non-loopback) URLs unless explicitly supplied via env.
	if !explicitURL && !isLoopback(cfg.URL) {
		return notDetected("bridge URL must be loopback (127.x, localhost, ::1)", "config"), false
	}

	if cfg.TokenFile == "" {
		return notDetected("bridge config missing tokenFile", "config"), false
	}

	token, err := readToken(cfg.TokenFile)
	if err != nil {
		return notDetected("token file unreadable", "config"), false
	}
	if token == "" {
		return notDetected("token file is empty", "config"), false
	}

	return BridgeState{
		Enabled:  true,
		Detected: true,
		URL:      cfg.URL,
		Token:    token,
		Runtimes: cfg.Runtimes,
		Message:  "bridge detected via config",
		Source:   "config",
	}, true
}

// readToken reads a token from a file, trims whitespace, and returns it.
func readToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// isLoopback reports whether rawURL resolves to a loopback address.
func isLoopback(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

// attemptAutoLaunch tries to run the helper binary, waits briefly, then retries
// detection from the config file. It is best-effort: any failure is swallowed.
func attemptAutoLaunch(helper, cfgPath string, prevState BridgeState) (BridgeState, error) {
	result := prevState
	result.AutoLaunchAttempted = true

	cmd := exec.Command(helper) //nolint:gosec // helper path comes from trusted env var
	_ = cmd.Start()             // best-effort; ignore error

	// Wait briefly for the helper to start.
	time.Sleep(500 * time.Millisecond)

	// Retry detection once.
	if state, ok := loadFromConfigFile(cfgPath); ok {
		state.AutoLaunchAttempted = true
		state.Source = "auto-launch"
		return state, nil
	}

	result.Message = "auto-launch attempted but bridge not detected"
	result.Source = "auto-launch"
	return result, nil
}
