package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempConfig writes a bridge config JSON to a temp file and returns the path.
func writeTempConfig(t *testing.T, v any) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "ai-bridge-*.json")
	if err != nil {
		t.Fatalf("create temp config: %v", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(v); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return f.Name()
}

// writeTokenFile creates a token file with the given content and returns the path.
func writeTokenFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "token-*")
	if err != nil {
		t.Fatalf("create temp token file: %v", err)
	}
	defer f.Close()
	f.WriteString(content)
	return f.Name()
}

// clearBridgeEnv clears all SCOUT_AI_BRIDGE_* env vars and restores them after the test.
func clearBridgeEnv(t *testing.T) {
	t.Helper()
	keys := []string{
		"SCOUT_AI_BRIDGE_CONFIG",
		"SCOUT_AI_BRIDGE_URL",
		"SCOUT_AI_BRIDGE_TOKEN_FILE",
		"SCOUT_AI_BRIDGE_DISABLE",
		"SCOUT_AI_BRIDGE_HELPER",
		"SCOUT_AI_BRIDGE_MODE",
		"XDG_CONFIG_HOME",
	}
	for _, k := range keys {
		orig, exists := os.LookupEnv(k)
		if exists {
			t.Cleanup(func() { os.Setenv(k, orig) })
		} else {
			t.Cleanup(func() { os.Unsetenv(k) })
		}
		os.Unsetenv(k)
	}
}

// TestDefaultConfigPath_XDG verifies XDG_CONFIG_HOME is used when set.
func TestDefaultConfigPath_XDG(t *testing.T) {
	clearBridgeEnv(t)

	xdgDir := t.TempDir()
	scoutDir := filepath.Join(xdgDir, "scout")
	if err := os.MkdirAll(scoutDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configPath := filepath.Join(scoutDir, "ai-bridge.json")

	tok := writeTokenFile(t, "tok-xdg")
	cfg := map[string]any{
		"type":      "http-localhost",
		"url":       "http://127.0.0.1:9900",
		"tokenFile": tok,
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	os.Setenv("XDG_CONFIG_HOME", xdgDir)
	os.Setenv("SCOUT_AI_BRIDGE_MODE", "local")
	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.Detected {
		t.Errorf("expected Detected=true, got false (message: %s)", state.Message)
	}
	if state.Source != "config" {
		t.Errorf("expected source=config, got %q", state.Source)
	}
	if state.Token != "tok-xdg" {
		t.Errorf("expected token=tok-xdg, got %q", state.Token)
	}
}

// TestDefaultConfigPath_Fallback verifies ~/.config is used when XDG_CONFIG_HOME is unset.
func TestDefaultConfigPath_Fallback(t *testing.T) {
	clearBridgeEnv(t)

	// Redirect HOME so we control where ~/.config points.
	homeDir := t.TempDir()
	origHome, hasHome := os.LookupEnv("HOME")
	os.Setenv("HOME", homeDir)
	if hasHome {
		t.Cleanup(func() { os.Setenv("HOME", origHome) })
	} else {
		t.Cleanup(func() { os.Unsetenv("HOME") })
	}

	scoutDir := filepath.Join(homeDir, ".config", "scout")
	if err := os.MkdirAll(scoutDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configPath := filepath.Join(scoutDir, "ai-bridge.json")

	tok := writeTokenFile(t, "tok-home")
	cfg := map[string]any{
		"type":      "http-localhost",
		"url":       "http://127.0.0.1:9901",
		"tokenFile": tok,
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	os.Setenv("SCOUT_AI_BRIDGE_MODE", "local")
	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.Detected {
		t.Errorf("expected Detected=true (message: %s)", state.Message)
	}
	if state.Source != "config" {
		t.Errorf("expected source=config, got %q", state.Source)
	}
}

// TestEnvOverridePath verifies SCOUT_AI_BRIDGE_CONFIG overrides the default path.
func TestEnvOverridePath(t *testing.T) {
	clearBridgeEnv(t)

	tok := writeTokenFile(t, "env-token")
	cfg := map[string]any{
		"type":      "http-localhost",
		"url":       "http://127.0.0.1:9902",
		"tokenFile": tok,
	}
	cfgPath := writeTempConfig(t, cfg)
	os.Setenv("SCOUT_AI_BRIDGE_CONFIG", cfgPath)

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.Detected {
		t.Errorf("expected Detected=true (message: %s)", state.Message)
	}
	if state.Token != "env-token" {
		t.Errorf("expected token=env-token, got %q", state.Token)
	}
}

// TestDisabledMode verifies SCOUT_AI_BRIDGE_DISABLE=1 returns not-detected.
func TestDisabledMode(t *testing.T) {
	clearBridgeEnv(t)
	os.Setenv("SCOUT_AI_BRIDGE_DISABLE", "1")

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Detected {
		t.Error("expected Detected=false in disabled mode")
	}
	if state.Enabled {
		t.Error("expected Enabled=false in disabled mode")
	}
	if state.Source != "disabled" {
		t.Errorf("expected source=disabled, got %q", state.Source)
	}
}

// TestInvalidJSON verifies invalid JSON returns a safe not-detected state.
func TestInvalidJSON(t *testing.T) {
	clearBridgeEnv(t)

	f, _ := os.CreateTemp(t.TempDir(), "bridge-*.json")
	f.WriteString("not valid json{{{")
	f.Close()
	os.Setenv("SCOUT_AI_BRIDGE_CONFIG", f.Name())

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("should not return error for invalid JSON, got: %v", err)
	}
	if state.Detected {
		t.Error("expected Detected=false for invalid JSON")
	}
}

// TestUnsupportedBridgeType verifies unknown types return not-detected safely.
func TestUnsupportedBridgeType(t *testing.T) {
	clearBridgeEnv(t)

	cfg := map[string]any{
		"type": "some-future-type",
		"url":  "http://127.0.0.1:9903",
	}
	cfgPath := writeTempConfig(t, cfg)
	os.Setenv("SCOUT_AI_BRIDGE_CONFIG", cfgPath)

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Detected {
		t.Error("expected Detected=false for unsupported bridge type")
	}
}

// TestMissingTokenFile verifies a missing token file returns not-detected.
func TestMissingTokenFile(t *testing.T) {
	clearBridgeEnv(t)

	cfg := map[string]any{
		"type":      "http-localhost",
		"url":       "http://127.0.0.1:9904",
		"tokenFile": "/this/path/does/not/exist/token.txt",
	}
	cfgPath := writeTempConfig(t, cfg)
	os.Setenv("SCOUT_AI_BRIDGE_CONFIG", cfgPath)

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Detected {
		t.Error("expected Detected=false when token file is missing")
	}
	// Token must not be leaked in message
	if strings.Contains(state.Message, "/this/path/does/not/exist") {
		t.Error("token path should not appear in message")
	}
}

// TestMissingTokenReference verifies that a config with no tokenFile returns not-detected.
func TestMissingTokenReference(t *testing.T) {
	clearBridgeEnv(t)

	cfg := map[string]any{
		"type": "http-localhost",
		"url":  "http://127.0.0.1:9905",
		// no tokenFile key
	}
	cfgPath := writeTempConfig(t, cfg)
	os.Setenv("SCOUT_AI_BRIDGE_CONFIG", cfgPath)

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Detected {
		t.Error("expected Detected=false when tokenFile is absent")
	}
}

// TestEmptyTokenFile verifies an empty token file returns not-detected.
func TestEmptyTokenFile(t *testing.T) {
	clearBridgeEnv(t)

	tok := writeTokenFile(t, "   \n  ") // whitespace only
	cfg := map[string]any{
		"type":      "http-localhost",
		"url":       "http://127.0.0.1:9906",
		"tokenFile": tok,
	}
	cfgPath := writeTempConfig(t, cfg)
	os.Setenv("SCOUT_AI_BRIDGE_CONFIG", cfgPath)

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Detected {
		t.Error("expected Detected=false for empty/whitespace-only token file")
	}
}

// TestPublicURLRejection verifies non-loopback URLs are rejected when not via env override.
func TestPublicURLRejection(t *testing.T) {
	clearBridgeEnv(t)

	tok := writeTokenFile(t, "some-token")
	cfg := map[string]any{
		"type":      "http-localhost",
		"url":       "https://api.example.com/bridge",
		"tokenFile": tok,
	}
	cfgPath := writeTempConfig(t, cfg)
	os.Setenv("SCOUT_AI_BRIDGE_CONFIG", cfgPath)

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Detected {
		t.Error("expected Detected=false for public/non-loopback URL")
	}
}

// TestExplicitURLAndTokenOverride verifies SCOUT_AI_BRIDGE_URL + SCOUT_AI_BRIDGE_TOKEN_FILE
// allow a non-loopback URL (explicitly supplied).
func TestExplicitURLAndTokenOverride(t *testing.T) {
	clearBridgeEnv(t)

	tok := writeTokenFile(t, "override-token")
	os.Setenv("SCOUT_AI_BRIDGE_URL", "https://my-private-bridge.internal/ai")
	os.Setenv("SCOUT_AI_BRIDGE_TOKEN_FILE", tok)

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.Detected {
		t.Errorf("expected Detected=true for explicit env URL override (message: %s)", state.Message)
	}
	if state.Token != "override-token" {
		t.Errorf("expected token=override-token, got %q", state.Token)
	}
	if state.Source != "env" {
		t.Errorf("expected source=env, got %q", state.Source)
	}
}

// TestTokenNotLeakedInMessages verifies the token value doesn't appear in state.Message.
func TestTokenNotLeakedInMessages(t *testing.T) {
	clearBridgeEnv(t)

	tok := writeTokenFile(t, "super-secret-token-value")
	cfg := map[string]any{
		"type":      "http-localhost",
		"url":       "http://127.0.0.1:9907",
		"tokenFile": tok,
	}
	cfgPath := writeTempConfig(t, cfg)
	os.Setenv("SCOUT_AI_BRIDGE_CONFIG", cfgPath)

	state, _ := LoadBridgeConfig()
	if strings.Contains(state.Message, "super-secret-token-value") {
		t.Error("token value must not appear in state.Message")
	}
}

// TestLoopbackVariants verifies common loopback addresses are allowed.
func TestLoopbackVariants(t *testing.T) {
	clearBridgeEnv(t)

	cases := []string{
		"http://127.0.0.1:8080",
		"http://localhost:8080",
		"http://[::1]:8080",
	}
	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			clearBridgeEnv(t)
			tok := writeTokenFile(t, "tok")
			cfg := map[string]any{
				"type":      "http-localhost",
				"url":       u,
				"tokenFile": tok,
			}
			cfgPath := writeTempConfig(t, cfg)
			os.Setenv("SCOUT_AI_BRIDGE_CONFIG", cfgPath)

			state, err := LoadBridgeConfig()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !state.Detected {
				t.Errorf("expected Detected=true for loopback URL %q (message: %s)", u, state.Message)
			}
		})
	}
}

// TestAutoLaunchFieldPresent verifies AutoLaunchAttempted is set correctly.
func TestAutoLaunchFieldPresent(t *testing.T) {
	clearBridgeEnv(t)
	// With no config and no helper, AutoLaunchAttempted must be false.
	state, _ := LoadBridgeConfig()
	if state.AutoLaunchAttempted {
		t.Error("expected AutoLaunchAttempted=false when no helper set")
	}
}

// TestAutoLaunchWithHelper verifies AutoLaunchAttempted=true when SCOUT_AI_BRIDGE_HELPER is set
// but the binary doesn't exist (best-effort, should not panic or error fatally).
func TestAutoLaunchWithHelper(t *testing.T) {
	clearBridgeEnv(t)
	os.Setenv("SCOUT_AI_BRIDGE_MODE", "local")
	os.Setenv("SCOUT_AI_BRIDGE_HELPER", "/this/binary/does/not/exist")

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.AutoLaunchAttempted {
		t.Error("expected AutoLaunchAttempted=true when helper binary is set")
	}
	// Still not detected (binary doesn't exist).
	if state.Detected {
		t.Error("expected Detected=false when helper doesn't exist")
	}
}

// TestBridgeStateFields verifies all expected fields exist on BridgeState.
func TestBridgeStateFields(t *testing.T) {
	var s BridgeState
	// Just ensure the struct has the expected shape by assigning fields.
	s.Enabled = true
	s.Detected = true
	s.URL = "http://127.0.0.1:9000"
	s.Token = "tok"
	s.Runtimes = []string{"go"}
	s.Message = "ok"
	s.Source = "config"
	s.AutoLaunchAttempted = false
	_ = s
}

// TestConfigFileMissing verifies a missing config file returns not-detected (not an error).
func TestConfigFileMissing(t *testing.T) {
	clearBridgeEnv(t)
	os.Setenv("SCOUT_AI_BRIDGE_CONFIG", "/no/such/file.json")

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("missing config must not return error, got: %v", err)
	}
	if state.Detected {
		t.Error("expected Detected=false for missing config file")
	}
}

// TestDefaultPolicyDoesNotReadImplicitUserConfig verifies that without SCOUT_AI_BRIDGE_MODE=local,
// a config file present at the implicit XDG path is NOT read and returns a policy not-detected result.
func TestDefaultPolicyDoesNotReadImplicitUserConfig(t *testing.T) {
	clearBridgeEnv(t)

	// Create a valid config at the implicit XDG path.
	xdgDir := t.TempDir()
	scoutDir := filepath.Join(xdgDir, "scout")
	if err := os.MkdirAll(scoutDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tok := writeTokenFile(t, "should-not-be-read")
	cfg := map[string]any{
		"type":      "http-localhost",
		"url":       "http://127.0.0.1:9910",
		"tokenFile": tok,
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(scoutDir, "ai-bridge.json"), data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	os.Setenv("XDG_CONFIG_HOME", xdgDir)
	// SCOUT_AI_BRIDGE_MODE is intentionally NOT set.

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Detected {
		t.Error("expected Detected=false: implicit config should not be read without SCOUT_AI_BRIDGE_MODE=local")
	}
	if state.Source != "policy" {
		t.Errorf("expected Source=policy, got %q", state.Source)
	}
	if !strings.Contains(state.Message, "SCOUT_AI_BRIDGE_MODE=local") {
		t.Errorf("expected policy message to mention SCOUT_AI_BRIDGE_MODE=local, got: %q", state.Message)
	}
}

// TestLocalModeReadsImplicitUserConfig verifies that with SCOUT_AI_BRIDGE_MODE=local,
// the implicit XDG config file IS discovered and returned.
func TestLocalModeReadsImplicitUserConfig(t *testing.T) {
	clearBridgeEnv(t)

	xdgDir := t.TempDir()
	scoutDir := filepath.Join(xdgDir, "scout")
	if err := os.MkdirAll(scoutDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tok := writeTokenFile(t, "local-mode-token")
	cfg := map[string]any{
		"type":      "http-localhost",
		"url":       "http://127.0.0.1:9911",
		"tokenFile": tok,
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(scoutDir, "ai-bridge.json"), data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	os.Setenv("XDG_CONFIG_HOME", xdgDir)
	os.Setenv("SCOUT_AI_BRIDGE_MODE", "local")

	state, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.Detected {
		t.Errorf("expected Detected=true with SCOUT_AI_BRIDGE_MODE=local (message: %s)", state.Message)
	}
	if state.Token != "local-mode-token" {
		t.Errorf("expected token=local-mode-token, got %q", state.Token)
	}
	if state.Source != "config" {
		t.Errorf("expected Source=config, got %q", state.Source)
	}
}
