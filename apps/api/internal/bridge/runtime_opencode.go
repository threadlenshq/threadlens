package bridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// opencodeGoModels is the set of model suffixes that use the "opencode-go/" CLI prefix.
// Populated from the curated catalog at construction time.
var opencodeGoModels = map[string]struct{}{
	"deepseek-v4-flash": {},
	"mimo-v2.5":         {},
	"minimax-m2.5":      {},
	"glm-5":             {},
	"glm-5.1":           {},
	"glm-5.2":           {},
	"mimo-v2.5-pro":     {},
	"minimax-m2.7":      {},
	"qwen3.6-plus":      {},
	"deepseek-v4-pro":   {},
	"minimax-m3":        {},
	"qwen3.7-plus":      {},
	"qwen3.7-max":       {},
	"kimi-k2.5":         {},
	"kimi-k2.6":         {},
	"kimi-k2.7-code":    {},
}

// opencodeFreeModels is the set of model suffixes that use the "opencode/" CLI prefix.
var opencodeFreeModels = map[string]struct{}{
	"big-pickle":             {},
	"deepseek-v4-flash-free": {},
	"mimo-v2.5-free":         {},
	"nemotron-3-ultra-free":  {},
	"north-mini-code-free":   {},
}

// OpencodeRuntime implements Runtime by invoking the opencode CLI with positional
// prompt arguments and parsing NDJSON event-stream responses.
type OpencodeRuntime struct {
	id           string // "opencode"
	binaryName   string // "opencode"
	resolvedPath string

	lookPath func(string) (string, error)
	run      func(ctx context.Context, path string, args []string, env []string) ([]byte, error)
}

// NewOpencodeRuntime returns a Runtime that invokes the opencode CLI.
func NewOpencodeRuntime() Runtime {
	return &OpencodeRuntime{
		id:         "opencode",
		binaryName: "opencode",
	}
}

func (r *OpencodeRuntime) ID() string { return r.id }

// Detect probes the host to see if the opencode binary is present and authenticated.
func (r *OpencodeRuntime) Detect(ctx context.Context) RuntimeStatus {
	lookPath := r.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	path, err := lookPath(r.binaryName)
	if err != nil {
		return RuntimeStatus{ID: r.id, Available: false, Message: "opencode not found"}
	}

	run := r.run
	if run == nil {
		run = execRun
	}

	// Auth probe: use a free-tier model with a tiny prompt and --format json.
	// A successful exit with valid NDJSON means authenticated.
	out, err := run(ctx, path, []string{"run", "--model", "opencode/big-pickle", "ping", "--format", "json"}, nil)
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = "runtime probe failed"
		}
		lowerMsg := strings.ToLower(msg)
		if strings.Contains(lowerMsg, "not authenticated") ||
			strings.Contains(lowerMsg, "login required") ||
			strings.Contains(lowerMsg, "sign in") ||
			strings.Contains(lowerMsg, "authenticate") {
			msg = "not authenticated"
		}
		return RuntimeStatus{ID: r.id, Available: false, Message: msg}
	}

	r.resolvedPath = path
	return RuntimeStatus{ID: r.id, Available: true, Message: "ok"}
}

// Generate calls the opencode CLI with the given request, parses the NDJSON output,
// and returns the concatenated text from type:"text" events.
func (r *OpencodeRuntime) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	if req.Provider != r.id {
		return "", fmt.Errorf("bridge: runtime %q cannot serve provider %q", r.id, req.Provider)
	}

	timeout := time.Duration(req.TimeoutMs) * time.Millisecond
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	combined := req.SystemPrompt + "\n\n" + req.UserMessage

	// Translate catalog model name to CLI prefix.
	cliModel, err := opencodeCLIPrefix(req.Model)
	if err != nil {
		return "", err
	}

	run := r.run
	if run == nil {
		run = execRun
	}

	env := sanitizedEnv()
	out, err := run(ctx, r.resolvedPath, []string{"run", "--model", cliModel, combined, "--format", "json"}, env)
	if err != nil {
		return "", err
	}

	text, parseErr := parseOpencodeNDJSON(out)
	if parseErr != nil {
		return "", parseErr
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("bridge: runtime %q returned empty response", r.id)
	}
	return strings.TrimSpace(text), nil
}

// opencodeCLIPrefix translates a catalog model suffix into the full CLI model
// argument with the correct opencode/ or opencode-go/ prefix.
func opencodeCLIPrefix(model string) (string, error) {
	if _, ok := opencodeGoModels[model]; ok {
		return "opencode-go/" + model, nil
	}
	if _, ok := opencodeFreeModels[model]; ok {
		return "opencode/" + model, nil
	}
	return "", fmt.Errorf("bridge: unknown opencode model %q", model)
}

// opencodeEvent is the minimal shape of an opencode NDJSON event.
type opencodeEvent struct {
	Type string `json:"type"`
	Part struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"part"`
}

// parseOpencodeNDJSON scans NDJSON output line-by-line, extracts .part.text from
// type:"text" events, and returns the concatenated result. Non-text events are
// silently skipped. Malformed lines are silently skipped.
func parseOpencodeNDJSON(data []byte) (string, error) {
	var builder strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Allow up to 1 MiB per line to handle long text events.
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var evt opencodeEvent
		if err := json.Unmarshal(line, &evt); err != nil {
			// Skip malformed lines.
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
