package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/configfile"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
)

// ErrDisabled is returned by Save (and any future mutating operations) when the
// onboarding flow has been administratively disabled. Handlers map this
// sentinel to HTTP 403 so callers can distinguish "you're not allowed" from a
// generic server error.
var ErrDisabled = errors.New("onboarding: disabled")

// ResetMode controls how much state is cleared on a Reset call.
type ResetMode string

const (
	// ResetModeProgress clears both the v1 progress state and the legacy
	// completion key, returning the service to a brand-new-install state.
	ResetModeProgress ResetMode = "progress"
)

// ExplorationUpdate carries the parameters for an UpdateExploration call.
type ExplorationUpdate struct {
	// Dismiss, when true, marks all pending exploration items as skipped.
	Dismiss bool `json:"dismiss"`
	// ItemID, when non-empty, marks a specific item as completed.
	ItemID ExplorationItem `json:"itemId,omitempty"`
}

// StarterProjectRequest carries input for CreateStarterProject.
type StarterProjectRequest struct {
	Name string `json:"name"`
}

// StarterProjectResult carries output from CreateStarterProject.
type StarterProjectResult struct {
	ProjectID string `json:"projectId"`
}

// ServiceIface is the narrow interface that HTTP handlers depend on. It is
// satisfied by *Service and by any test stub that needs to drive handler
// behaviour without touching real I/O.
type ServiceIface interface {
	GetStatus(ctx context.Context) (Status, error)
	SaveRequiredStep(ctx context.Context, step RequiredStep, values map[string]string) error
	Save(ctx context.Context, values map[string]string) error
	UpdateExploration(ctx context.Context, req ExplorationUpdate) error
	CreateStarterProject(ctx context.Context, req StarterProjectRequest) (StarterProjectResult, error)
	Reset(ctx context.Context, mode ResetMode) error
}

// Service encapsulates all business logic for the onboarding flow.
type Service struct {
	cfg         Config
	repo        *settings.Repository
	projectRepo *repository.Repository
}

// NewService constructs a Service. It returns an error if the Config is
// inconsistent or the repository is nil. projectRepo may be nil for
// environments that do not exercise project-count behaviour.
func NewService(cfg Config, repo *settings.Repository, projectRepo *repository.Repository) (*Service, error) {
	if repo == nil {
		return nil, errors.New("onboarding: settings repository must not be nil")
	}
	if cfg.CompletionKey == "" {
		return nil, errors.New("onboarding: Config.CompletionKey must not be empty")
	}
	if cfg.StateKey == "" {
		return nil, errors.New("onboarding: Config.StateKey must not be empty")
	}
	if cfg.DockerMode && cfg.EnvFilePath == "" {
		return nil, errors.New("onboarding: Config.EnvFilePath must not be empty in Docker mode")
	}
	return &Service{cfg: cfg, repo: repo, projectRepo: projectRepo}, nil
}

// ── persistence helpers ────────────────────────────────────────────────────────

// loadProgress reads the v1 Progress from StateKey, falling back to the legacy
// CompletionKey if no v1 state is found. It always returns a valid Progress.
func (s *Service) loadProgress(ctx context.Context) (Progress, error) {
	// Try the v1 state key first.
	raw, found, err := s.repo.Get(ctx, s.cfg.StateKey)
	if err != nil {
		return Progress{}, fmt.Errorf("onboarding: loadProgress (state key): %w", err)
	}
	if found && raw != "" {
		var p Progress
		if jsonErr := json.Unmarshal([]byte(raw), &p); jsonErr == nil {
			return s.normalizeProgress(p), nil
		}
		// Corrupt JSON — fall through to defaults.
	}

	// Fall back: check the legacy completion key for migration.
	legacyVal, legacyFound, legacyErr := s.repo.Get(ctx, s.cfg.CompletionKey)
	if legacyErr != nil {
		return Progress{}, fmt.Errorf("onboarding: loadProgress (legacy key): %w", legacyErr)
	}
	if legacyFound && legacyVal == "true" {
		// Migrate legacy "complete" into a v1 progress that is already in
		// exploration phase with required setup done.
		p := NewProgress()
		p.RequiredSetup.Status = RequiredStatusComplete
		p.RequiredSetup.CurrentStep = RequiredStepReview
		p.RequiredSetup.CompletedSteps = []RequiredStep{
			RequiredStepWelcome,
			RequiredStepAIProvider,
			RequiredStepAppDatabase,
			RequiredStepReview,
		}
		p.Exploration.Status = ExplorationStatusActive
		MarkUpdated(&p)
		return s.normalizeProgress(p), nil
	}

	// Brand-new install.
	return s.normalizeProgress(NewProgress()), nil
}

// saveProgress serialises p to JSON and writes it to the StateKey. Secrets are
// never embedded in Progress, so serialisation is safe.
func (s *Service) saveProgress(ctx context.Context, p Progress) error {
	MarkUpdated(&p)
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("onboarding: saveProgress marshal: %w", err)
	}
	if err := s.repo.Set(ctx, s.cfg.StateKey, string(data)); err != nil {
		return fmt.Errorf("onboarding: saveProgress set: %w", err)
	}
	return nil
}

// normalizeProgress fills in any missing fields with sensible defaults so that
// old/partial progress values do not cause nil-pointer panics downstream.
func (s *Service) normalizeProgress(p Progress) Progress {
	if p.Version == 0 {
		p.Version = 1
	}
	if p.RequiredSetup.CompletedSteps == nil {
		p.RequiredSetup.CompletedSteps = []RequiredStep{}
	}
	if p.RequiredSetup.CurrentStep == "" {
		p.RequiredSetup.CurrentStep = RequiredStepWelcome
	}
	if p.RequiredSetup.Status == "" {
		p.RequiredSetup.Status = RequiredStatusNotStarted
	}
	if p.Exploration.Items == nil {
		p.Exploration.Items = make(map[ExplorationItem]ItemState, len(ExplorationItems))
		for _, item := range ExplorationItems {
			p.Exploration.Items[item] = ItemStatePending
		}
	} else {
		// Ensure all canonical items are present.
		for _, item := range ExplorationItems {
			if _, ok := p.Exploration.Items[item]; !ok {
				p.Exploration.Items[item] = ItemStatePending
			}
		}
	}
	if p.Exploration.CurrentItem == "" {
		p.Exploration.CurrentItem = ExplorationItemStarterProject
	}
	if p.Exploration.Status == "" {
		p.Exploration.Status = ExplorationStatusNotStarted
	}
	return p
}

// ── status assembly ────────────────────────────────────────────────────────────

// GetStatus returns a point-in-time snapshot of the full v1 onboarding state.
func (s *Service) GetStatus(ctx context.Context) (Status, error) {
	if s.cfg.Disabled {
		return Status{
			Enabled:     false,
			Phase:       PhaseDisabled,
			EnvFilePath: s.cfg.EnvFilePath,
			Steps:       buildStepViews(NewProgress()),
			Items:       buildItemViews(NewProgress()),
		}, nil
	}

	p, err := s.loadProgress(ctx)
	if err != nil {
		return Status{}, err
	}

	phase := PhaseForProgress(true, p)
	requiredSetupComplete := p.RequiredSetup.Status == RequiredStatusComplete
	explorationComplete := ExplorationComplete(p.Exploration.Items) ||
		p.Exploration.Dismissed ||
		p.Exploration.Status == ExplorationStatusComplete

	return Status{
		Enabled:                true,
		Complete:               phase == PhaseComplete || phase == PhaseExploration,
		RequiredSetupComplete:  requiredSetupComplete,
		ExplorationComplete:    explorationComplete,
		Phase:                  phase,
		CurrentRequiredStep:    p.RequiredSetup.CurrentStep,
		CurrentExplorationItem: p.Exploration.CurrentItem,
		Steps:                  buildStepViews(p),
		Items:                  buildItemViews(p),
		Capabilities:           Capabilities{},
		AppDatabase:            AppDatabaseStatus{RuntimeMode: s.cfg.EnvFilePath},
		Context:                p.Context,
		EnvFilePath:            s.cfg.EnvFilePath,
	}, nil
}

// buildStepViews produces the ordered display list of required-setup steps.
func buildStepViews(p Progress) []StepView {
	completed := make(map[RequiredStep]bool, len(p.RequiredSetup.CompletedSteps))
	for _, s := range p.RequiredSetup.CompletedSteps {
		completed[s] = true
	}
	views := make([]StepView, 0, len(RequiredSteps))
	for _, id := range RequiredSteps {
		views = append(views, StepView{
			ID:        id,
			Label:     stepLabel(id),
			Completed: completed[id],
			Current:   id == p.RequiredSetup.CurrentStep,
		})
	}
	return views
}

// buildItemViews produces the ordered display list of exploration items.
func buildItemViews(p Progress) []ItemView {
	views := make([]ItemView, 0, len(ExplorationItems))
	for _, id := range ExplorationItems {
		state := p.Exploration.Items[id]
		views = append(views, ItemView{
			ID:    id,
			Label: itemLabel(id),
			State: state,
		})
	}
	return views
}

// stepLabel returns a human-readable label for a required setup step.
func stepLabel(s RequiredStep) string {
	switch s {
	case RequiredStepWelcome:
		return "Welcome"
	case RequiredStepAIProvider:
		return "AI Provider"
	case RequiredStepAppDatabase:
		return "App Database"
	case RequiredStepReview:
		return "Review"
	default:
		return string(s)
	}
}

// itemLabel returns a human-readable label for an exploration item.
func itemLabel(i ExplorationItem) string {
	switch i {
	case ExplorationItemStarterProject:
		return "Create a starter project"
	case ExplorationItemStarterQuery:
		return "Add a starter query"
	case ExplorationItemFirstScout:
		return "Run your first Scout"
	case ExplorationItemReviewResults:
		return "Review the results"
	case ExplorationItemReportsIntro:
		return "Explore Reports"
	case ExplorationItemSettingsIntro:
		return "Visit Settings"
	default:
		return string(i)
	}
}

// ── mutating operations ────────────────────────────────────────────────────────

// IsComplete reports whether the onboarding completion flag has been stored in
// the settings repository (legacy key check for backward compatibility).
func (s *Service) IsComplete(ctx context.Context) (bool, error) {
	val, found, err := s.repo.Get(ctx, s.cfg.CompletionKey)
	if err != nil {
		return false, fmt.Errorf("onboarding: IsComplete: %w", err)
	}
	return found && val == "true", nil
}

// SaveRequiredStep marks the given step as complete, advances the resume point
// to the next step, and persists non-secret values in the progress context.
// It returns the updated Status.
func (s *Service) SaveRequiredStep(ctx context.Context, step RequiredStep, values map[string]string) error {
	if s.cfg.Disabled {
		return ErrDisabled
	}

	p, err := s.loadProgress(ctx)
	if err != nil {
		return err
	}

	// Record non-secret context values derived from the step.
	if provider, ok := values["AI_PROVIDER"]; ok {
		p.Context.AIProviderPath = provider
	}

	// Mark this step complete and advance to the next.
	p.RequiredSetup.Status = RequiredStatusActive
	alreadyRecorded := false
	for _, cs := range p.RequiredSetup.CompletedSteps {
		if cs == step {
			alreadyRecorded = true
			break
		}
	}
	if !alreadyRecorded {
		p.RequiredSetup.CompletedSteps = append(p.RequiredSetup.CompletedSteps, step)
	}

	// Advance the current-step pointer.
	for i, s := range RequiredSteps {
		if s == step && i+1 < len(RequiredSteps) {
			p.RequiredSetup.CurrentStep = RequiredSteps[i+1]
			break
		}
	}

	return s.saveProgress(ctx, p)
}

// Save writes the supplied key/value pairs to the env file (Docker mode only),
// marks the legacy completion flag, and advances the progress to the
// exploration phase. It returns an error when onboarding is disabled or, in
// Docker mode, when values are missing/empty.
func (s *Service) Save(ctx context.Context, values map[string]string) error {
	if s.cfg.Disabled {
		return ErrDisabled
	}

	if s.cfg.DockerMode {
		if len(values) == 0 {
			return errors.New("onboarding: save rejected — no values provided in Docker mode")
		}
		for k, v := range values {
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("onboarding: save rejected — empty value for key %q in Docker mode", k)
			}
		}
		if _, err := configfile.UpdateFile(s.cfg.EnvFilePath, values, nil); err != nil {
			return fmt.Errorf("onboarding: writing env file: %w", err)
		}
	}

	// Set the legacy completion flag.
	if err := s.repo.Set(ctx, s.cfg.CompletionKey, "true"); err != nil {
		return fmt.Errorf("onboarding: marking complete: %w", err)
	}

	// Update v1 progress to show required setup complete and move to exploration.
	p, err := s.loadProgress(ctx)
	if err != nil {
		return err
	}
	p.RequiredSetup.Status = RequiredStatusComplete
	if p.RequiredSetup.CurrentStep != RequiredStepReview {
		// Mark all steps as completed.
		p.RequiredSetup.CompletedSteps = make([]RequiredStep, len(RequiredSteps))
		copy(p.RequiredSetup.CompletedSteps, RequiredSteps)
		p.RequiredSetup.CurrentStep = RequiredStepReview
	}
	p.Exploration.Status = ExplorationStatusActive

	// Do not persist any secret values — only the non-secret provider path.
	// Secrets stay only in the env file.

	return s.saveProgress(ctx, p)
}

// UpdateExploration updates the exploration phase based on req. When
// Dismiss=true, all pending items are marked as skipped.
func (s *Service) UpdateExploration(ctx context.Context, req ExplorationUpdate) error {
	p, err := s.loadProgress(ctx)
	if err != nil {
		return err
	}

	if req.Dismiss {
		for item, state := range p.Exploration.Items {
			if state == ItemStatePending || state == ItemStateBlocked {
				p.Exploration.Items[item] = ItemStateSkipped
			}
		}
		p.Exploration.Dismissed = true
		p.Exploration.Status = ExplorationStatusComplete
	} else if req.ItemID != "" {
		p.Exploration.Items[req.ItemID] = ItemStateCompleted
	}

	return s.saveProgress(ctx, p)
}

// CreateStarterProject creates a starter project. This is a stub implementation
// for Task 5; full logic is added in Tasks 6–7.
func (s *Service) CreateStarterProject(ctx context.Context, req StarterProjectRequest) (StarterProjectResult, error) {
	return StarterProjectResult{}, errors.New("onboarding: CreateStarterProject not yet implemented")
}

// Reset clears onboarding state according to mode. ResetModeProgress removes
// both the v1 progress state and the legacy completion key.
func (s *Service) Reset(ctx context.Context, mode ResetMode) error {
	if err := s.repo.Delete(ctx, s.cfg.StateKey); err != nil {
		return fmt.Errorf("onboarding: reset (state key): %w", err)
	}
	if err := s.repo.Delete(ctx, s.cfg.CompletionKey); err != nil {
		return fmt.Errorf("onboarding: reset (completion key): %w", err)
	}
	return nil
}

// DebugProgressJSONForTest returns the raw JSON string stored under StateKey.
// It is intended only for use in tests that need to assert the serialised form
// of progress does not contain secrets.
func (s *Service) DebugProgressJSONForTest(ctx context.Context) (string, bool, error) {
	return s.repo.Get(ctx, s.cfg.StateKey)
}
