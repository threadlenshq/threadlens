package ai

// ModelEntry represents a single AI model available in the catalog.
type ModelEntry struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Label    string `json:"label"`
	Tier     string `json:"tier"`
	Cost     string `json:"cost"`
}

// TaskEntry represents a task that can be configured with a model.
type TaskEntry struct {
	ID                string            `json:"id"`
	Label             string            `json:"label"`
	Complexity        string            `json:"complexity"`
	Volume            string            `json:"volume"`
	TimeoutMs         int               `json:"timeoutMs"`
	Description       string            `json:"description"`
	Default           string            `json:"default"`
	DefaultByProvider map[string]string `json:"defaultByProvider"`
}

// ModelCatalog is the full list of available AI models, mirroring catalog.js MODEL_CATALOG.
var ModelCatalog = []ModelEntry{
	{ID: "copilot:gpt-4.1", Provider: "copilot", Model: "gpt-4.1", Label: "GPT-4.1 (Copilot)", Tier: "medium", Cost: "usage-based"},
	{ID: "copilot:gpt-5-mini", Provider: "copilot", Model: "gpt-5-mini", Label: "GPT-5 mini (Copilot)", Tier: "low", Cost: "usage-based"},
	{ID: "copilot:gpt-5.4-mini", Provider: "copilot", Model: "gpt-5.4-mini", Label: "GPT-5.4 mini (Copilot)", Tier: "medium", Cost: "usage-based"},
	{ID: "copilot:gpt-5.4", Provider: "copilot", Model: "gpt-5.4", Label: "GPT-5.4 (Copilot)", Tier: "high", Cost: "usage-based"},
	{ID: "copilot:gpt-5.5", Provider: "copilot", Model: "gpt-5.5", Label: "GPT-5.5 (Copilot)", Tier: "reasoning", Cost: "usage-based"},
	{ID: "copilot:claude-haiku-4.5", Provider: "copilot", Model: "claude-haiku-4.5", Label: "Claude Haiku 4.5 (Copilot)", Tier: "low", Cost: "usage-based"},
	{ID: "copilot:claude-sonnet-4.6", Provider: "copilot", Model: "claude-sonnet-4.6", Label: "Claude Sonnet 4.6 (Copilot)", Tier: "medium", Cost: "usage-based"},
	{ID: "claude-cli:haiku", Provider: "claude-cli", Model: "haiku", Label: "Claude Haiku (CLI)", Tier: "low", Cost: "token-billed"},
	{ID: "claude-cli:sonnet", Provider: "claude-cli", Model: "sonnet", Label: "Claude Sonnet (CLI)", Tier: "medium", Cost: "token-billed"},
	{ID: "claude-cli:opus", Provider: "claude-cli", Model: "opus", Label: "Claude Opus (CLI)", Tier: "reasoning", Cost: "token-billed"},
	{ID: "sdk:haiku", Provider: "sdk", Model: "claude-haiku-4-5-20251001", Label: "Claude Haiku (SDK)", Tier: "low", Cost: "token-billed"},
	{ID: "gemini:2.5-flash", Provider: "gemini", Model: "gemini-2.5-flash", Label: "Gemini 2.5 Flash", Tier: "low", Cost: "token-billed"},
	{ID: "gemini:3.5-flash", Provider: "gemini", Model: "gemini-3.5-flash", Label: "Gemini 3.5 Flash", Tier: "reasoning", Cost: "token-billed"},
	{ID: "gemini:2.5-pro", Provider: "gemini", Model: "gemini-2.5-pro", Label: "Gemini 2.5 Pro", Tier: "medium", Cost: "token-billed"},
	{ID: "gemini:3.1-pro", Provider: "gemini", Model: "gemini-3.1-pro", Label: "Gemini 3.1 Pro", Tier: "high", Cost: "token-billed"},
	{ID: "opencode:big-pickle", Provider: "opencode", Model: "big-pickle", Label: "Big Pickle (opencode)", Tier: "medium", Cost: "free (0x)"},
	{ID: "opencode:deepseek-v4-flash-free", Provider: "opencode", Model: "deepseek-v4-flash-free", Label: "DeepSeek V4 Flash Free (opencode)", Tier: "medium", Cost: "free (0x)"},
	{ID: "opencode:mimo-v2.5-free", Provider: "opencode", Model: "mimo-v2.5-free", Label: "Mimo v2.5 Free (opencode)", Tier: "medium", Cost: "free (0x)"},
	{ID: "opencode:nemotron-3-ultra-free", Provider: "opencode", Model: "nemotron-3-ultra-free", Label: "Nemotron 3 Ultra Free (opencode)", Tier: "high", Cost: "free (0x)"},
	{ID: "opencode:north-mini-code-free", Provider: "opencode", Model: "north-mini-code-free", Label: "North Mini Code Free (opencode)", Tier: "medium", Cost: "free (0x)"},
	{ID: "opencode-go:deepseek-v4-flash", Provider: "opencode-go", Model: "deepseek-v4-flash", Label: "DeepSeek V4 Flash (opencode Go)", Tier: "medium", Cost: "subscription"},
	{ID: "opencode-go:mimo-v2.5", Provider: "opencode-go", Model: "mimo-v2.5", Label: "Mimo v2.5 (opencode Go)", Tier: "medium", Cost: "subscription"},
	{ID: "opencode-go:glm-5.1", Provider: "opencode-go", Model: "glm-5.1", Label: "GLM-5.1 (opencode Go)", Tier: "high", Cost: "subscription"},
	{ID: "opencode-go:glm-5.2", Provider: "opencode-go", Model: "glm-5.2", Label: "GLM-5.2 (opencode Go)", Tier: "reasoning", Cost: "subscription"},
	{ID: "opencode-go:mimo-v2.5-pro", Provider: "opencode-go", Model: "mimo-v2.5-pro", Label: "Mimo v2.5 Pro (opencode Go)", Tier: "high", Cost: "subscription"},
	{ID: "opencode-go:minimax-m2.7", Provider: "opencode-go", Model: "minimax-m2.7", Label: "MiniMax M2.7 (opencode Go)", Tier: "high", Cost: "subscription"},
	{ID: "opencode-go:qwen3.6-plus", Provider: "opencode-go", Model: "qwen3.6-plus", Label: "Qwen 3.6 Plus (opencode Go)", Tier: "medium", Cost: "subscription"},
	{ID: "opencode-go:deepseek-v4-pro", Provider: "opencode-go", Model: "deepseek-v4-pro", Label: "DeepSeek V4 Pro (opencode Go)", Tier: "high", Cost: "subscription"},
	{ID: "opencode-go:minimax-m3", Provider: "opencode-go", Model: "minimax-m3", Label: "MiniMax M3 (opencode Go)", Tier: "high", Cost: "subscription"},
	{ID: "opencode-go:qwen3.7-plus", Provider: "opencode-go", Model: "qwen3.7-plus", Label: "Qwen 3.7 Plus (opencode Go)", Tier: "high", Cost: "subscription"},
	{ID: "opencode-go:qwen3.7-max", Provider: "opencode-go", Model: "qwen3.7-max", Label: "Qwen 3.7 Max (opencode Go)", Tier: "high", Cost: "subscription"},
	{ID: "opencode-go:kimi-k2.6", Provider: "opencode-go", Model: "kimi-k2.6", Label: "Kimi K2.6 (opencode Go)", Tier: "high", Cost: "subscription"},
	{ID: "opencode-go:kimi-k2.7-code", Provider: "opencode-go", Model: "kimi-k2.7-code", Label: "Kimi K2.7 Code (opencode Go)", Tier: "high", Cost: "subscription"},
}

// Tasks is the full list of configurable tasks, mirroring catalog.js TASKS.
var Tasks = []TaskEntry{
	{
		ID: "post_scoring", Label: "Post Scoring", Complexity: "low", Volume: "high", TimeoutMs: 60000,
		Description: "Per-post pain-signal scoring during Reddit/Bluesky scout runs. High volume - runs 10 batches of 15 posts in parallel.",
		Default:     "copilot:gpt-5-mini",
		DefaultByProvider: map[string]string{
			"copilot":    "copilot:gpt-5-mini",
			"claude-cli": "claude-cli:haiku",
			"opencode":   "opencode-go:deepseek-v4-flash",
			"sdk":        "sdk:haiku",
			"gemini":     "gemini:2.5-flash",
		},
	},
	{
		ID: "query_suggestion", Label: "Query Suggestion", Complexity: "medium", Volume: "one-time", TimeoutMs: 60000,
		Description: "Generates 10 search URLs and angles when adding queries.",
		Default:     "copilot:gpt-5.4-mini",
		DefaultByProvider: map[string]string{
			"copilot":    "copilot:gpt-5.4-mini",
			"claude-cli": "claude-cli:sonnet",
			"opencode":   "opencode-go:mimo-v2.5-pro",
			"sdk":        "sdk:haiku",
			"gemini":     "gemini:3.1-pro",
		},
	},
	{
		ID: "query_refinement", Label: "Query Refinement", Complexity: "reasoning", Volume: "one-time", TimeoutMs: 300000,
		Description: "Reviews current queries against report findings and recommends which to disable or add next. Uses a 5-minute timeout because report-backed refinement can run longer than basic suggestions.",
		Default:     "claude-cli:opus",
		DefaultByProvider: map[string]string{
			"copilot":    "copilot:gpt-5.5",
			"claude-cli": "claude-cli:opus",
			"opencode":   "opencode-go:glm-5.2",
			"sdk":        "sdk:haiku",
			"gemini":     "gemini:3.5-flash",
		},
	},
	{
		ID: "report_clustering", Label: "Research Report", Complexity: "reasoning", Volume: "one-time", TimeoutMs: 300000,
		Description: "Clusters posts into pain themes and proposes product angles. One-time per report; 5-minute timeout.",
		Default:     "claude-cli:opus",
		DefaultByProvider: map[string]string{
			"copilot":    "copilot:gpt-5.5",
			"claude-cli": "claude-cli:opus",
			"opencode":   "opencode-go:glm-5.2",
			"sdk":        "sdk:haiku",
			"gemini":     "gemini:3.5-flash",
		},
	},
	{
		ID: "draft_generation", Label: "Draft Comment / DM", Complexity: "medium", Volume: "per-post", TimeoutMs: 60000,
		Description: "Drafts contextual replies and direct messages for outreach.",
		Default:     "claude-cli:sonnet",
		DefaultByProvider: map[string]string{
			"copilot":    "copilot:claude-sonnet-4.6",
			"claude-cli": "claude-cli:sonnet",
			"opencode":   "opencode-go:mimo-v2.5-pro",
			"sdk":        "sdk:haiku",
			"gemini":     "gemini:3.1-pro",
		},
	},
	{
		ID: "google_analysis", Label: "Google Search Analysis", Complexity: "low", Volume: "high", TimeoutMs: 60000,
		Description: "Per-result product extraction plus relevance and intent scoring in Google scout.",
		Default:     "copilot:claude-haiku-4.5",
		DefaultByProvider: map[string]string{
			"copilot":    "copilot:gpt-5-mini",
			"claude-cli": "claude-cli:haiku",
			"opencode":   "opencode-go:deepseek-v4-flash",
			"sdk":        "sdk:haiku",
			"gemini":     "gemini:2.5-flash",
		},
	},
	{
		ID: "advisor_council", Label: "Advisor Council", Complexity: "reasoning", Volume: "one-time", TimeoutMs: 300000,
		Description: "Five-advisor council critique of a completed research report.",
		Default:     "claude-cli:opus",
		DefaultByProvider: map[string]string{
			"copilot":    "copilot:gpt-5.5",
			"claude-cli": "claude-cli:opus",
			"opencode":   "opencode-go:glm-5.2",
			"sdk":        "sdk:haiku",
			"gemini":     "gemini:3.5-flash",
		},
	},
	{
		ID: "prompt_suggestion", Label: "Prompt Suggestion", Complexity: "medium", Volume: "one-time", TimeoutMs: 60000,
		Description: "Generates three complete prompt drafts for a given platform+type slot, seeded with project context and existing queries.",
		Default:     "copilot:gpt-5.4-mini",
		DefaultByProvider: map[string]string{
			"copilot":    "copilot:gpt-5.4-mini",
			"claude-cli": "claude-cli:sonnet",
			"opencode":   "opencode-go:mimo-v2.5-pro",
			"sdk":        "sdk:haiku",
			"gemini":     "gemini:3.1-pro",
		},
	},
}

// GetModel returns the model entry with the given ID, or nil if not found.
func GetModel(id string) *ModelEntry {
	for i := range ModelCatalog {
		if ModelCatalog[i].ID == id {
			return &ModelCatalog[i]
		}
	}
	return nil
}

// GetTask returns the task entry with the given ID, or nil if not found.
func GetTask(id string) *TaskEntry {
	for i := range Tasks {
		if Tasks[i].ID == id {
			return &Tasks[i]
		}
	}
	return nil
}
