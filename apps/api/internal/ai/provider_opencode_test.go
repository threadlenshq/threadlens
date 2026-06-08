package ai

import (
	"strings"
	"testing"
)

func TestOpencodeProvider_Name(t *testing.T) {
	p := &OpencodeProvider{}
	if p.Name() != "opencode" {
		t.Fatalf("expected name 'opencode', got %q", p.Name())
	}
}

func TestOpencodeProvider_AvailableFalseWhenNoPath(t *testing.T) {
	p := &OpencodeProvider{resolvedPath: ""}
	if p.Available() {
		t.Fatal("expected unavailable when resolvedPath is empty")
	}
}

func TestOpencodeProvider_AvailableTrueWhenPathSet(t *testing.T) {
	p := &OpencodeProvider{resolvedPath: "/fake/opencode"}
	if !p.Available() {
		t.Fatal("expected available when resolvedPath is set")
	}
}

func TestParseOpencodeProviderNDJSON_TextEvents(t *testing.T) {
	input := `{"type":"step_start","part":{"type":"step-start"}}
{"type":"text","part":{"type":"text","text":"Hello "}}
{"type":"text","part":{"type":"text","text":"World"}}
{"type":"step_finish","part":{"type":"step-finish"}}`

	text, err := parseOpencodeProviderNDJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Hello World" {
		t.Fatalf("expected 'Hello World', got %q", text)
	}
}

func TestParseOpencodeProviderNDJSON_EmptyInput(t *testing.T) {
	text, err := parseOpencodeProviderNDJSON([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Fatalf("expected empty string, got %q", text)
	}
}

func TestParseOpencodeProviderNDJSON_NoTextEvents(t *testing.T) {
	input := `{"type":"step_start","part":{"type":"step-start"}}
{"type":"step_finish","part":{"type":"step-finish"}}`

	text, err := parseOpencodeProviderNDJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Fatalf("expected empty string when no text events, got %q", text)
	}
}

func TestParseOpencodeProviderNDJSON_MalformedLines(t *testing.T) {
	input := `not-valid-json
{"type":"text","part":{"type":"text","text":"ok"}}
{also broken`

	text, err := parseOpencodeProviderNDJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "ok" {
		t.Fatalf("expected 'ok', got %q", text)
	}
}

func TestOpencodeProviderCLIPrefix_GoModel(t *testing.T) {
	result, err := opencodeProviderCLIPrefix("qwen3.7-max")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "opencode-go/qwen3.7-max" {
		t.Fatalf("expected 'opencode-go/qwen3.7-max', got %q", result)
	}
}

func TestOpencodeProviderCLIPrefix_FreeModel(t *testing.T) {
	result, err := opencodeProviderCLIPrefix("big-pickle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "opencode/big-pickle" {
		t.Fatalf("expected 'opencode/big-pickle', got %q", result)
	}
}

func TestOpencodeProviderCLIPrefix_UnknownModel(t *testing.T) {
	_, err := opencodeProviderCLIPrefix("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
	if !strings.Contains(err.Error(), "unknown opencode model") {
		t.Fatalf("expected unknown-model error, got: %v", err)
	}
}
