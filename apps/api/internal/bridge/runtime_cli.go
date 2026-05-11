package bridge

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CLIRuntime implements Runtime by invoking an external CLI binary.
type CLIRuntime struct {
	id           string
	binaryName   string
	defaultModel string
	resolvedPath string

	// Hooks for testing; if nil, real implementations are used.
	lookPath func(string) (string, error)
	run      func(ctx context.Context, path string, args []string, env []string) ([]byte, error)
}

func (r *CLIRuntime) ID() string { return r.id }

// Detect probes the host to see if the binary is present and authenticated.
func (r *CLIRuntime) Detect(ctx context.Context) RuntimeStatus {
	lookPath := r.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	path, err := lookPath(r.binaryName)
	if err != nil {
		return RuntimeStatus{ID: r.id, Available: false, Message: r.binaryName + " not found"}
	}

	run := r.run
	if run == nil {
		run = execRun
	}

	out, err := run(ctx, path, []string{"--model", r.defaultModel, "-p", ""}, nil)
	if err != nil {
		msg := string(out)
		if !strings.Contains(strings.ToLower(msg), "not authenticated") {
			msg = "not authenticated"
		}
		return RuntimeStatus{ID: r.id, Available: false, Message: msg}
	}

	r.resolvedPath = path
	return RuntimeStatus{ID: r.id, Available: true, Message: "ok"}
}

// Generate calls the CLI binary with the given request.
func (r *CLIRuntime) Generate(ctx context.Context, req GenerateRequest) (string, error) {
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

	run := r.run
	if run == nil {
		run = execRun
	}

	env := sanitizedEnv()
	out, err := run(ctx, r.resolvedPath, []string{"--model", req.Model, "-p", combined}, env)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// sanitizedEnv returns os.Environ() with ANTHROPIC_API_KEY removed.
func sanitizedEnv() []string {
	all := os.Environ()
	result := make([]string, 0, len(all))
	for _, entry := range all {
		if !strings.HasPrefix(entry, "ANTHROPIC_API_KEY=") {
			result = append(result, entry)
		}
	}
	return result
}

// execRun is the real subprocess runner used in production.
func execRun(ctx context.Context, path string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	if env != nil {
		cmd.Env = env
	}
	return cmd.Output()
}
