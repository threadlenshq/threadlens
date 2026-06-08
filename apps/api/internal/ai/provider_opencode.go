package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// OpencodeProvider implements Provider by invoking the opencode CLI directly
// in the host process. It is used as a fallback when no bridge is available
// (e.g. host-mode Go API). In Dockerized Scout the binary is not on PATH
// inside the container, so Available() returns false and only the bridge path works.
type OpencodeProvider struct {
	resolvedPath string
}

// NewOpencodeProvider creates an OpencodeProvider. The binary path is resolved
// once at construction so it works regardless of PATH changes.
func NewOpencodeProvider() *OpencodeProvider {
	path, _ := exec.LookPath("opencode")
	return &OpencodeProvider{resolvedPath: path}
}

func (p *OpencodeProvider) Name() string { return "opencode" }

func (p *OpencodeProvider) Available() bool { return p.resolvedPath != "" }

func (p *OpencodeProvider) Generate(ctx context.Context, model string, systemPrompt string, userMessage string, timeout time.Duration) (string, error) {
	if !p.Available() {
		return "", fmt.Errorf("opencode not found in PATH")
	}

	combinedPrompt := systemPrompt + "\n\n" + userMessage

	// Determine CLI prefix for the model.
	cliModel, err := opencodeProviderCLIPrefix(model)
	if err != nil {
		return "", err
	}

	// Strip ANTHROPIC_API_KEY so the CLI uses its own auth.
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "ANTHROPIC_API_KEY=") {
			filtered = append(filtered, e)
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, p.resolvedPath, "run", "--model", cliModel, combinedPrompt, "--format", "json")
	cmd.Env = filtered
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("opencode run --model %s failed: %w", cliModel, err)
	}

	text, parseErr := parseOpencodeProviderNDJSON(out)
	if parseErr != nil {
		return "", parseErr
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("opencode returned empty response")
	}
	return strings.TrimSpace(text), nil
}

// opencodeProviderCLIPrefix translates a catalog model suffix into the full
// CLI model argument. Uses the same prefix logic as the bridge runtime.
func opencodeProviderCLIPrefix(model string) (string, error) {
	goModels := map[string]struct{}{
		"deepseek-v4-flash": {}, "mimo-v2.5": {}, "minimax-m2.5": {},
		"glm-5": {}, "glm-5.1": {}, "mimo-v2.5-pro": {},
		"minimax-m2.7": {}, "qwen3.6-plus": {}, "deepseek-v4-pro": {},
		"minimax-m3": {}, "qwen3.7-plus": {}, "qwen3.7-max": {},
		"kimi-k2.5": {}, "kimi-k2.6": {},
	}
	freeModels := map[string]struct{}{
		"big-pickle": {}, "deepseek-v4-flash-free": {},
	}
	if _, ok := goModels[model]; ok {
		return "opencode-go/" + model, nil
	}
	if _, ok := freeModels[model]; ok {
		return "opencode/" + model, nil
	}
	return "", fmt.Errorf("unknown opencode model %q", model)
}

// opencodeProviderEvent is the minimal shape of an opencode NDJSON event.
type opencodeProviderEvent struct {
	Type string `json:"type"`
	Part struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"part"`
}

// parseOpencodeProviderNDJSON scans NDJSON output line-by-line, extracts .part.text
// from type:"text" events, and returns the concatenated result.
func parseOpencodeProviderNDJSON(data []byte) (string, error) {
	var builder strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var evt opencodeProviderEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			continue
		}
		if evt.Type == "text" && evt.Part.Text != "" {
			builder.WriteString(evt.Part.Text)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("opencode: error reading NDJSON: %w", err)
	}
	return builder.String(), nil
}
