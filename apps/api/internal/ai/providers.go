package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Provider is the interface all AI providers must implement.
type Provider interface {
	// Name returns a unique identifier for this provider instance (e.g. "copilot:gpt-5-mini").
	Name() string
	// Available reports whether this provider can be used right now (binary exists, key set, etc.).
	Available() bool
	// Generate sends the combined prompt and returns the trimmed text response.
	Generate(ctx context.Context, model string, systemPrompt string, userMessage string, timeout time.Duration) (string, error)
}

// ---------------------------------------------------------------------------
// CLIProvider – wraps the `copilot` or `claude` binary.
// ---------------------------------------------------------------------------

// CLIProvider runs a local CLI tool (copilot or claude) to fulfil requests.
type CLIProvider struct {
	// binaryName is the name used with exec.LookPath, e.g. "copilot" or "claude".
	binaryName string
	// resolvedPath caches the absolute path discovered at construction time.
	resolvedPath string
}

// NewCLIProvider creates a CLIProvider.  The binary path is resolved once at
// construction so it continues to work regardless of PATH changes.
func NewCLIProvider(binaryName string) *CLIProvider {
	path, _ := exec.LookPath(binaryName)
	return &CLIProvider{binaryName: binaryName, resolvedPath: path}
}

func (p *CLIProvider) Name() string { return p.binaryName }

func (p *CLIProvider) Available() bool { return p.resolvedPath != "" }

func (p *CLIProvider) Generate(ctx context.Context, model string, systemPrompt string, userMessage string, timeout time.Duration) (string, error) {
	if !p.Available() {
		return "", fmt.Errorf("%s not found in PATH", p.binaryName)
	}
	combinedPrompt := systemPrompt + "\n\n" + userMessage

	// Strip ANTHROPIC_API_KEY so the CLI uses its own auth, matching Express behaviour.
	env := os.Environ()
	filtered := env[:0]
	for _, e := range env {
		if !strings.HasPrefix(e, "ANTHROPIC_API_KEY=") {
			filtered = append(filtered, e)
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, p.resolvedPath, "--model", model, "-p", combinedPrompt)
	cmd.Env = filtered
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s --model %s failed: %w", p.binaryName, model, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ---------------------------------------------------------------------------
// AnthropicProvider – calls the Anthropic Messages REST API.
// ---------------------------------------------------------------------------

// AnthropicProvider calls the Anthropic Messages API using ANTHROPIC_API_KEY.
type AnthropicProvider struct{}

func (p *AnthropicProvider) Name() string    { return "sdk" }
func (p *AnthropicProvider) Available() bool { return os.Getenv("ANTHROPIC_API_KEY") != "" }

func (p *AnthropicProvider) Generate(ctx context.Context, model string, systemPrompt string, userMessage string, timeout time.Duration) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	reqBody, err := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages":   []map[string]string{{"role": "user", "content": userMessage}},
	})
	if err != nil {
		return "", err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("Anthropic returned empty content")
	}
	return strings.TrimSpace(result.Content[0].Text), nil
}

// ---------------------------------------------------------------------------
// GeminiProvider – calls the Gemini generateContent REST API.
// ---------------------------------------------------------------------------

// GeminiProvider calls the Google Gemini generateContent API using GEMINI_API_KEY.
type GeminiProvider struct{}

func (p *GeminiProvider) Name() string    { return "gemini" }
func (p *GeminiProvider) Available() bool { return os.Getenv("GEMINI_API_KEY") != "" }

func (p *GeminiProvider) Generate(ctx context.Context, model string, systemPrompt string, userMessage string, timeout time.Duration) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not set")
	}

	reqBody, err := json.Marshal(map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": userMessage}}},
		},
	})
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini returned empty candidates")
	}
	return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
}
