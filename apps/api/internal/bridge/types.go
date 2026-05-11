package bridge

import "context"

// RuntimeStatus reports the availability of a CLI runtime detected on the host.
type RuntimeStatus struct {
	ID        string `json:"id"`
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

// GenerateRequest is the payload sent to the bridge daemon's /generate endpoint.
type GenerateRequest struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	SystemPrompt string `json:"systemPrompt"`
	UserMessage  string `json:"userMessage"`
	TimeoutMs    int    `json:"timeoutMs"`
}

// GenerateResponse is the payload returned by the bridge daemon's /generate endpoint.
type GenerateResponse struct {
	Text string `json:"text"`
}

// HealthResponse is the payload returned by the bridge daemon's /health endpoint.
type HealthResponse struct {
	OK       bool            `json:"ok"`
	Runtimes []RuntimeStatus `json:"runtimes"`
}

// Runtime is the interface each CLI-backed AI runtime must implement.
type Runtime interface {
	// ID returns the unique identifier for this runtime (e.g. "copilot", "claude").
	ID() string
	// Detect probes the host to determine whether the runtime is available.
	Detect(ctx context.Context) RuntimeStatus
	// Generate runs a prompt through the runtime and returns the generated text.
	Generate(ctx context.Context, req GenerateRequest) (string, error)
}
