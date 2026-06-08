package bridge

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestOpencodeRuntime_DetectMissingBinary(t *testing.T) {
	rt := &OpencodeRuntime{
		id: "opencode", binaryName: "opencode",
		lookPath: func(string) (string, error) { return "", errors.New("missing") },
	}
	status := rt.Detect(context.Background())
	if status.Available || !strings.Contains(status.Message, "not found") {
		t.Fatalf("expected not-found status, got %#v", status)
	}
}

func TestOpencodeRuntime_DetectUnauthenticated(t *testing.T) {
	rt := &OpencodeRuntime{
		id: "opencode", binaryName: "opencode",
		lookPath: func(string) (string, error) { return "/fake/opencode", nil },
		run: func(context.Context, string, []string, []string) ([]byte, error) {
			return []byte("Error: not authenticated with opencode providers"), errors.New("exit 1")
		},
	}
	status := rt.Detect(context.Background())
	if status.Available || !strings.Contains(status.Message, "not authenticated") {
		t.Fatalf("expected not-authenticated status, got %#v", status)
	}
}

func TestOpencodeRuntime_DetectAuthenticated(t *testing.T) {
	calls := 0
	rt := &OpencodeRuntime{
		id: "opencode", binaryName: "opencode",
		lookPath: func(string) (string, error) { return "/fake/opencode", nil },
		run: func(ctx context.Context, path string, args []string, env []string) ([]byte, error) {
			calls++
			if path != "/fake/opencode" {
				t.Fatalf("unexpected path %q", path)
			}
			// Verify auth probe uses free-tier model and positional prompt.
			if len(args) != 6 || args[0] != "run" || args[1] != "--model" ||
				args[2] != "opencode/big-pickle" || args[3] != "ping" ||
				args[4] != "--format" || args[5] != "json" {
				t.Fatalf("unexpected auth probe args: %#v", args)
			}
			return []byte(`{"type":"text","part":{"type":"text","text":"pong"}}`), nil
		},
	}
	status := rt.Detect(context.Background())
	if !status.Available || status.Message != "ok" {
		t.Fatalf("expected available status, got %#v", status)
	}
	if calls != 1 {
		t.Fatalf("expected 1 probe call, got %d", calls)
	}
	// Verify resolved path is cached.
	if rt.resolvedPath != "/fake/opencode" {
		t.Fatalf("expected resolvedPath=/fake/opencode, got %q", rt.resolvedPath)
	}
}

func TestOpencodeRuntime_GenerateWrongProvider(t *testing.T) {
	rt := &OpencodeRuntime{id: "opencode", resolvedPath: "/fake/opencode"}
	_, err := rt.Generate(context.Background(), GenerateRequest{Provider: "copilot", Model: "gpt-5-mini"})
	if err == nil {
		t.Fatal("expected wrong-provider error")
	}
	if !strings.Contains(err.Error(), "cannot serve provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpencodeRuntime_GenerateParsesNDJSON(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "secret-key")
	ndjson := `{"type":"step_start","part":{"type":"step-start"}}
{"type":"text","part":{"type":"text","text":"HELLO_"}}
{"type":"text","part":{"type":"text","text":"WORLD"}}
{"type":"step_finish","part":{"type":"step-finish"}}`

	rt := &OpencodeRuntime{
		id: "opencode", resolvedPath: "/fake/opencode",
		run: func(ctx context.Context, path string, args []string, env []string) ([]byte, error) {
			// Verify argument order: run --model <prefix/model> <prompt> --format json
			if args[0] != "run" || args[1] != "--model" {
				t.Fatalf("expected 'run --model' prefix, got %#v", args[:2])
			}
			if args[2] != "opencode-go/deepseek-v4-flash" {
				t.Fatalf("expected opencode-go prefix, got model arg %q", args[2])
			}
			// args[3] is the combined prompt
			if args[4] != "--format" || args[5] != "json" {
				t.Fatalf("expected '--format json' suffix, got %#v", args[4:])
			}
			// Verify ANTHROPIC_API_KEY is stripped.
			for _, entry := range env {
				if strings.HasPrefix(entry, "ANTHROPIC_API_KEY=") {
					t.Fatal("ANTHROPIC_API_KEY leaked into CLI environment")
				}
			}
			return []byte(ndjson), nil
		},
	}
	text, err := rt.Generate(context.Background(), GenerateRequest{
		Provider: "opencode", Model: "deepseek-v4-flash",
		SystemPrompt: "sys", UserMessage: "user", TimeoutMs: 30000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "HELLO_WORLD" {
		t.Fatalf("expected 'HELLO_WORLD', got %q", text)
	}
}

func TestOpencodeRuntime_GenerateSkipsNonTextEvents(t *testing.T) {
	ndjson := `{"type":"step_start","part":{"type":"step-start"}}
{"type":"step_finish","part":{"type":"step-finish"}}`

	rt := &OpencodeRuntime{
		id: "opencode", resolvedPath: "/fake/opencode",
		run: func(context.Context, string, []string, []string) ([]byte, error) {
			return []byte(ndjson), nil
		},
	}
	_, err := rt.Generate(context.Background(), GenerateRequest{
		Provider: "opencode", Model: "big-pickle",
		SystemPrompt: "sys", UserMessage: "user",
	})
	if err == nil {
		t.Fatal("expected empty-response error when no text events present")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("expected empty-response error, got: %v", err)
	}
}

func TestOpencodeRuntime_GenerateEmptyStdout(t *testing.T) {
	rt := &OpencodeRuntime{
		id: "opencode", resolvedPath: "/fake/opencode",
		run: func(context.Context, string, []string, []string) ([]byte, error) {
			return []byte(""), nil
		},
	}
	_, err := rt.Generate(context.Background(), GenerateRequest{
		Provider: "opencode", Model: "big-pickle",
		SystemPrompt: "sys", UserMessage: "user",
	})
	if err == nil {
		t.Fatal("expected empty-response error for empty stdout")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("expected empty-response error, got: %v", err)
	}
}

func TestOpencodeRuntime_GenerateTimeout(t *testing.T) {
	rt := &OpencodeRuntime{
		id: "opencode", resolvedPath: "/fake/opencode",
		run: func(ctx context.Context, path string, args []string, env []string) ([]byte, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	_, err := rt.Generate(context.Background(), GenerateRequest{
		Provider: "opencode", Model: "big-pickle",
		SystemPrompt: "sys", UserMessage: "user", TimeoutMs: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestOpencodeRuntime_GenerateFreePrefix(t *testing.T) {
	var capturedModel string
	rt := &OpencodeRuntime{
		id: "opencode", resolvedPath: "/fake/opencode",
		run: func(ctx context.Context, path string, args []string, env []string) ([]byte, error) {
			capturedModel = args[2]
			return []byte(`{"type":"text","part":{"type":"text","text":"ok"}}`), nil
		},
	}
	_, err := rt.Generate(context.Background(), GenerateRequest{
		Provider: "opencode", Model: "big-pickle",
		SystemPrompt: "sys", UserMessage: "user",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedModel != "opencode/big-pickle" {
		t.Fatalf("expected opencode/big-pickle, got %q", capturedModel)
	}
}

func TestOpencodeRuntime_GenerateGoPrefix(t *testing.T) {
	var capturedModel string
	rt := &OpencodeRuntime{
		id: "opencode", resolvedPath: "/fake/opencode",
		run: func(ctx context.Context, path string, args []string, env []string) ([]byte, error) {
			capturedModel = args[2]
			return []byte(`{"type":"text","part":{"type":"text","text":"ok"}}`), nil
		},
	}
	_, err := rt.Generate(context.Background(), GenerateRequest{
		Provider: "opencode", Model: "qwen3.7-max",
		SystemPrompt: "sys", UserMessage: "user",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedModel != "opencode-go/qwen3.7-max" {
		t.Fatalf("expected opencode-go/qwen3.7-max, got %q", capturedModel)
	}
}

func TestOpencodeRuntime_GenerateUnknownModel(t *testing.T) {
	rt := &OpencodeRuntime{
		id: "opencode", resolvedPath: "/fake/opencode",
		run: func(context.Context, string, []string, []string) ([]byte, error) {
			return []byte(`{"type":"text","part":{"type":"text","text":"ok"}}`), nil
		},
	}
	_, err := rt.Generate(context.Background(), GenerateRequest{
		Provider: "opencode", Model: "nonexistent-model",
		SystemPrompt: "sys", UserMessage: "user",
	})
	if err == nil {
		t.Fatal("expected unknown-model error")
	}
	if !strings.Contains(err.Error(), "unknown opencode model") {
		t.Fatalf("expected unknown-model error, got: %v", err)
	}
}

func TestParseOpencodeNDJSON_MalformedLines(t *testing.T) {
	input := `not-json
{"type":"text","part":{"type":"text","text":"good"}}
{broken
`
	text, err := parseOpencodeNDJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "good" {
		t.Fatalf("expected 'good', got %q", text)
	}
}
