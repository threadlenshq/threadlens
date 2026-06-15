package services

import (
	"context"
	"encoding/json"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/tenant"
)

// ResolvedModel holds the result of resolving which model to use for a task.
type ResolvedModel struct {
	ModelID string `json:"modelId"`
	Source  string `json:"source"` // "user" or "default"
}

// ModelService handles business logic for model configuration.
type ModelService struct {
	repo     *repository.Repository
	mode     entitlements.RuntimeMode
	resolver entitlements.Resolver
}

// NewModelService creates a new ModelService.
func NewModelService(repo *repository.Repository, mode entitlements.RuntimeMode, resolver entitlements.Resolver) *ModelService {
	return &ModelService{repo: repo, mode: mode, resolver: resolver}
}

// Catalog returns the model catalog with entitlement-aware managed provider metadata,
// read-only host bridge status, the user's current provider, and per-task defaults
// overridden to match the current provider.
func (s *ModelService) Catalog(ctx context.Context) (map[string]any, error) {
	subject := tenant.SubjectFromContext(ctx, s.mode)
	decision, err := s.resolver.Check(ctx, entitlements.CheckRequest{Subject: subject, Capability: entitlements.CapabilityManagedAIUse, Action: "models.catalog.managed_ai"})
	if err != nil {
		return nil, err
	}

	bridgeState := ai.LoadBridgeStatus()

	// Read the user's current provider from app_settings; default to "sdk".
	provider := "sdk"
	if raw, ok, getErr := s.repo.GetSetting(ctx, "ai_provider"); getErr == nil && ok && raw != "" {
		provider = raw
	}

	// Build a shallow copy of ai.Tasks with per-provider default overrides.
	// The override only applies when DefaultByProvider has the key AND the
	// referenced model exists in the catalog.
	tasksCopy := make([]ai.TaskEntry, len(ai.Tasks))
	copy(tasksCopy, ai.Tasks)
	for i := range tasksCopy {
		if override, found := tasksCopy[i].DefaultByProvider[provider]; found && override != "" {
			if ai.GetModel(override) != nil {
				tasksCopy[i].Default = override
			}
		}
	}

	return map[string]any{
		"models": ai.ModelCatalog,
		"tasks":  tasksCopy,
		"managedProvider": map[string]any{
			"available":  decision.Allowed,
			"capability": entitlements.CapabilityManagedAIUse,
		},
		"externalRuntime": safeBridgeStatus(bridgeState),
		"currentProvider": provider,
	}, nil
}

// safeBridgeStatus maps a BridgeState into a read-only catalog payload.
// It explicitly excludes token values, token file paths, and the bridge URL
// so that no secrets or host paths are leaked to the client.
func safeBridgeStatus(s ai.BridgeState) map[string]any {
	runtimes := s.Runtimes
	if runtimes == nil {
		runtimes = []string{}
	}
	return map[string]any{
		"type":                "host-cli-bridge",
		"detected":            s.Detected,
		"availableRuntimes":   runtimes,
		"source":              s.Source,
		"autoLaunchAttempted": s.AutoLaunchAttempted,
		"message":             s.Message,
		"scope":               "optional-local",
		"description":         "Optional local bridge transport for Copilot and Claude CLI catalog models; production can use direct CLI or API-key providers without it.",
	}
}

// ResolveTaskModel resolves which model to use for the given task, checking user overrides first.
func ResolveTaskModel(ctx context.Context, repo *repository.Repository, taskID string) (ResolvedModel, error) {
	task := ai.GetTask(taskID)
	if task == nil {
		return ResolvedModel{}, &ModelError{Kind: "unknownTask", TaskID: taskID}
	}

	key := "model." + taskID
	raw, ok, err := repo.GetSetting(ctx, key)
	if err != nil {
		return ResolvedModel{}, err
	}

	if ok && raw != "" {
		var obj map[string]string
		if jsonErr := json.Unmarshal([]byte(raw), &obj); jsonErr == nil {
			if modelID, exists := obj["modelId"]; exists && ai.GetModel(modelID) != nil {
				return ResolvedModel{ModelID: modelID, Source: "user"}, nil
			}
		}
	}

	return ResolvedModel{ModelID: task.Default, Source: "default"}, nil
}

// GetConfig returns the current model configuration for all tasks.
func (s *ModelService) GetConfig(ctx context.Context) (map[string]ResolvedModel, error) {
	config := make(map[string]ResolvedModel, len(ai.Tasks))
	for _, task := range ai.Tasks {
		resolved, err := ResolveTaskModel(ctx, s.repo, task.ID)
		if err != nil {
			return nil, err
		}
		config[task.ID] = resolved
	}
	return config, nil
}

// SetConfig sets the model for a task. Returns structured ModelError on validation failures.
func (s *ModelService) SetConfig(ctx context.Context, taskID string, modelID string) (ResolvedModel, error) {
	if ai.GetTask(taskID) == nil {
		return ResolvedModel{}, &ModelError{Kind: "unknownTask", TaskID: taskID}
	}
	if ai.GetModel(modelID) == nil {
		return ResolvedModel{}, &ModelError{Kind: "unknownModel", ModelID: modelID}
	}
	if err := s.repo.SetModelSetting(ctx, taskID, modelID); err != nil {
		return ResolvedModel{}, err
	}
	return ResolvedModel{ModelID: modelID, Source: "user"}, nil
}

// DeleteConfig resets the model for a task back to its default. Returns error if task unknown.
func (s *ModelService) DeleteConfig(ctx context.Context, taskID string) error {
	if ai.GetTask(taskID) == nil {
		return &ModelError{Kind: "unknownTask", TaskID: taskID}
	}
	return s.repo.DeleteModelSetting(ctx, taskID)
}

// StoreProvider persists the chosen AI provider to app_settings under the key
// "ai_provider". The provider value is stored as a plain string (not JSON).
func (s *ModelService) StoreProvider(ctx context.Context, provider string) error {
	if provider == "" {
		return &ModelError{Kind: "invalidProvider"}
	}
	return s.repo.SetSetting(ctx, "ai_provider", provider)
}

// ModelError carries structured error info for model operations.
type ModelError struct {
	Kind    string
	TaskID  string
	ModelID string
}

func (e *ModelError) Error() string {
	switch e.Kind {
	case "unknownTask":
		return "unknownTask:" + e.TaskID
	case "unknownModel":
		return "unknownModel:" + e.ModelID
	case "invalidProvider":
		return "invalidProvider: provider must not be empty"
	}
	return "model error"
}
