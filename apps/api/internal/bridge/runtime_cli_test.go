package bridge

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCLIRuntimeDetectsMissingBinary(t *testing.T) {
	rt := &CLIRuntime{id: "copilot", binaryName: "copilot", defaultModel: "gpt-5-mini", lookPath: func(string) (string, error) { return "", errors.New("missing") }}
	status := rt.Detect(context.Background())
	if status.Available || !strings.Contains(status.Message, "not found") {
		t.Fatalf("expected missing status, got %#v", status)
	}
}

func TestCLIRuntimeDetectsAuthenticatedBinary(t *testing.T) {
	calls := 0
	rt := &CLIRuntime{
		id: "copilot", binaryName: "copilot", defaultModel: "gpt-5-mini",
		lookPath: func(string) (string, error) { return "/fake/copilot", nil },
		run: func(ctx context.Context, path string, args []string, env []string) ([]byte, error) {
			calls++
			if path != "/fake/copilot" || len(args) != 4 || args[0] != "--model" || args[2] != "-p" || args[3] != "ping" {
				t.Fatalf("unexpected auth probe command path=%q args=%#v", path, args)
			}
			return []byte("ok"), nil
		},
	}
	status := rt.Detect(context.Background())
	if !status.Available {
		t.Fatalf("expected authenticated runtime, got %#v", status)
	}
	if calls != 1 {
		t.Fatalf("expected one auth probe, got %d", calls)
	}
}

func TestCLIRuntimeDetectsUnauthenticatedBinary(t *testing.T) {
	rt := &CLIRuntime{
		id: "claude-cli", binaryName: "claude", defaultModel: "haiku",
		lookPath: func(string) (string, error) { return "/fake/claude", nil },
		run:      func(context.Context, string, []string, []string) ([]byte, error) { return []byte("login required"), errors.New("exit 1") },
	}
	status := rt.Detect(context.Background())
	if status.Available || !strings.Contains(status.Message, "not authenticated") {
		t.Fatalf("expected unauthenticated status, got %#v", status)
	}
}

func TestCLIRuntimeGenerateUsesProviderModelAndSanitizedEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "secret-key")
	rt := &CLIRuntime{
		id: "claude-cli", binaryName: "claude", defaultModel: "haiku", resolvedPath: "/fake/claude",
		run: func(ctx context.Context, path string, args []string, env []string) ([]byte, error) {
			if path != "/fake/claude" {
				t.Fatalf("unexpected path %q", path)
			}
			if strings.Join(args, "|") != "--model|opus|-p|sys\n\nuser" {
				t.Fatalf("unexpected args %#v", args)
			}
			for _, entry := range env {
				if strings.HasPrefix(entry, "ANTHROPIC_API_KEY=") {
					t.Fatal("ANTHROPIC_API_KEY leaked into CLI environment")
				}
			}
			return []byte(" generated text \n"), nil
		},
	}
	text, err := rt.Generate(context.Background(), GenerateRequest{Provider: "claude-cli", Model: "opus", SystemPrompt: "sys", UserMessage: "user", TimeoutMs: 12345})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "generated text" {
		t.Fatalf("expected trimmed text, got %q", text)
	}
}

func TestCLIRuntimeGenerateErrorsOnTimeoutAndWrongProvider(t *testing.T) {
	rt := &CLIRuntime{id: "copilot", binaryName: "copilot", defaultModel: "gpt-5-mini", resolvedPath: "/fake/copilot"}
	if _, err := rt.Generate(context.Background(), GenerateRequest{Provider: "claude-cli", Model: "haiku"}); err == nil {
		t.Fatal("expected wrong provider error")
	}
	rt.run = func(ctx context.Context, path string, args []string, env []string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	_, err := rt.Generate(context.Background(), GenerateRequest{Provider: "copilot", Model: "gpt-5-mini", TimeoutMs: 1})
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected timeout error, got %v", err)
	}
	_ = os.Unsetenv("ANTHROPIC_API_KEY")
	_ = time.Second
}
