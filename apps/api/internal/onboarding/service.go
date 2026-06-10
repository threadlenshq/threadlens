package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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

	// ResetModeExploration clears only the exploration phase state,
	// leaving required-setup progress intact.
	ResetModeExploration ResetMode = "exploration"

	// ResetModeDismissExploration marks all pending exploration items as
	// skipped without erasing the underlying progress record.
	ResetModeDismissExploration ResetMode = "dismiss_exploration"
)

// ExplorationUpdate carries the parameters for an UpdateExploration call.
type ExplorationUpdate struct {
	// Item, when non-empty, identifies the exploration item to update.
	Item ExplorationItem `json:"item,omitempty"`
	// State is the new ItemState to apply to Item.
	State ItemState `json:"state,omitempty"`
	// Dismiss, when true, marks all pending exploration items as skipped.
	Dismiss bool `json:"dismiss"`
	// SelectedProjectID, when non-empty, records the project the user chose
	// during the starter-project exploration step.
	SelectedProjectID string `json:"selectedProjectId,omitempty"`
}


// ServiceIface is the narrow interface that HTTP handlers depend on. It is
// satisfied by *Service and by any test stub that needs to drive handler
// behaviour without touching real I/O.
type ServiceIface interface {
	GetStatus(ctx context.Context) (Status, error)
	SaveRequiredStep(ctx context.Context, step RequiredStep, values map[string]string) (Status, error)
	Save(ctx context.Context, values map[string]string) error
	UpdateExploration(ctx context.Context, req ExplorationUpdate) (Status, error)
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
		if jsonErr := json.Unmarshal([]byte(raw), &p); jsonErr != nil {
			return Progress{}, fmt.Errorf("onboarding: loadProgress: corrupt stored JSON: %w", jsonErr)
		}
		return s.normalizeProgress(p), nil
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

// applyProviderKeysToProcessEnv calls os.Setenv for AI provider keys found in
// values, so the running process can use them immediately without a restart.
// Empty values are skipped to avoid overwriting an already-loaded key with a
// blank. Non-AI keys are silently ignored.
func applyProviderKeysToProcessEnv(values map[string]string) {
	aiKeys := map[string]bool{
		"ANTHROPIC_API_KEY": true,
		"GEMINI_API_KEY":    true,
	}
	for k, v := range values {
		if !aiKeys[k] {
			continue
		}
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		os.Setenv(k, trimmed)
	}
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
	if p.Exploration.CurrentItem == "" && p.Exploration.Status != ExplorationStatusComplete {
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
			Items:       s.buildItemViews(NewProgress()),
		}, nil
	}

	p, err := s.loadProgress(ctx)
	if err != nil {
		return Status{}, err
	}
	return s.statusFromProgress(p), nil
}

// statusFromProgress assembles a Status from an already-loaded Progress,
// avoiding a second DB round-trip in mutating operations.
func (s *Service) statusFromProgress(p Progress) Status {
	phase := PhaseForProgress(true, p)
	requiredSetupComplete := p.RequiredSetup.Status == RequiredStatusComplete
	explorationComplete := ExplorationComplete(p.Exploration.Items) ||
		p.Exploration.Dismissed ||
		p.Exploration.Status == ExplorationStatusComplete

	return Status{
		Enabled:                true,
		Complete:               phase == PhaseComplete,
		RequiredSetupComplete:  requiredSetupComplete,
		ExplorationComplete:    explorationComplete,
		Phase:                  phase,
		CurrentRequiredStep:    p.RequiredSetup.CurrentStep,
		CurrentExplorationItem: p.Exploration.CurrentItem,
		Steps:                  buildStepViews(p),
		Items:                  s.buildItemViews(p),
		Capabilities:           buildCapabilities(),
		AppDatabase:            buildAppDatabaseStatus(s.cfg),
		Context:                p.Context,
		EnvFilePath:            s.cfg.EnvFilePath,
	}
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
// For ExplorationItemStarterQuery, it populates SeededQueryCount from the
// project repository. If the count cannot be loaded, it defaults to 0.
func (s *Service) buildItemViews(p Progress) []ItemView {
	seededCount := 0
	if p.Context.StarterProjectID != "" && s.projectRepo != nil {
		queries, err := s.projectRepo.ListAllQueries(context.Background(), p.Context.StarterProjectID)
		if err == nil {
			seededCount = len(queries)
		}
	}
	views := make([]ItemView, 0, len(ExplorationItems))
	for _, id := range ExplorationItems {
		state := p.Exploration.Items[id]
		count := 0
		if id == ExplorationItemStarterQuery {
			count = seededCount
		}
		views = append(views, ItemView{
			ID:               id,
			Label:            itemLabel(id),
			State:            state,
			SeededQueryCount: count,
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

// buildCapabilities returns the current Capabilities snapshot. In Task 5 the
// provider list is empty; Tasks 6–7 will populate it from live env inspection.
func buildCapabilities() Capabilities {
	return Capabilities{
		Providers: []ProviderCapability{},
		Sources:   SourceCapabilities{},
	}
}

// buildAppDatabaseStatus derives the AppDatabaseStatus from the service config.
func buildAppDatabaseStatus(cfg Config) AppDatabaseStatus {
	runtimeMode := "native"
	if cfg.DockerMode {
		runtimeMode = "docker"
	}
	envFileLabel := ""
	if cfg.EnvFilePath != "" {
		envFileLabel = cfg.EnvFilePath
	}
	return AppDatabaseStatus{
		DatabasePathLabel: cfg.DBPath,
		RuntimeMode:       runtimeMode,
		EnvFileLabel:      envFileLabel,
		EnvWritable:       cfg.DockerMode,
	}
}

// validExplorationItem reports whether item is one of the approved enum values.
func validExplorationItem(item ExplorationItem) bool {
	for _, known := range ExplorationItems {
		if item == known {
			return true
		}
	}
	return false
}

// validItemState reports whether state is one of the approved enum values.
func validItemState(state ItemState) bool {
	switch state {
	case ItemStatePending, ItemStateCompleted, ItemStateSkipped, ItemStateBlocked:
		return true
	}
	return false
}

// firstPendingItem returns the first ExplorationItem whose state is pending or
// blocked, walking ExplorationItems in canonical order. If all items are
// terminal it returns an empty string.
func firstPendingItem(items map[ExplorationItem]ItemState) ExplorationItem {
	for _, id := range ExplorationItems {
		switch items[id] {
		case ItemStatePending, ItemStateBlocked:
			return id
		}
	}
	return ""
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
// to the next incomplete required step, persists non-secret values in the
// progress context, and returns the updated Status.
//
// The step must equal the current resume point (p.RequiredSetup.CurrentStep).
// Attempts to save a step out of order — whether skipping ahead or repeating an
// earlier step — are rejected with a descriptive error. This keeps the linear
// flow strict and deterministic.
func (s *Service) SaveRequiredStep(ctx context.Context, step RequiredStep, values map[string]string) (Status, error) {
	if s.cfg.Disabled {
		return Status{}, ErrDisabled
	}

	// Validate the step is a known required step.
	if !stringInRequiredSteps(step) {
		return Status{}, fmt.Errorf("onboarding: SaveRequiredStep: unknown step %q", step)
	}

	p, err := s.loadProgress(ctx)
	if err != nil {
		return Status{}, err
	}

	// Build completed set once; reused for the skip-ahead guard, idempotent
	// append, and current-step advancement below.
	completed := make(map[RequiredStep]bool, len(p.RequiredSetup.CompletedSteps))
	for _, cs := range p.RequiredSetup.CompletedSteps {
		completed[cs] = true
	}
	alreadyCompleted := completed[step]

	// Allow re-saving an already-completed step (user went back and pressed
	// Continue again). Only reject steps that would skip ahead past the current
	// resume point.
	if !alreadyCompleted && step != p.RequiredSetup.CurrentStep {
		return Status{}, fmt.Errorf(
			"onboarding: SaveRequiredStep: step %q is not the current step (expected %q)",
			step, p.RequiredSetup.CurrentStep,
		)
	}

	// For the AI-provider step, the provider path is required and stored in
	// context. Other steps do not apply this rule.
	if step == RequiredStepAIProvider {
		provider := strings.TrimSpace(values["AI_PROVIDER"])
		if provider == "" {
			return Status{}, errors.New("onboarding: AI_PROVIDER is required")
		}
		p.Context.AIProviderPath = provider
	}

	// Promote status from not_started → active; never regress from active or complete.
	if p.RequiredSetup.Status == RequiredStatusNotStarted {
		p.RequiredSetup.Status = RequiredStatusActive
	}

	// Mark this step complete (idempotent).
	if !alreadyCompleted {
		p.RequiredSetup.CompletedSteps = append(p.RequiredSetup.CompletedSteps, step)
		completed[step] = true
	}

	// Advance the current-step pointer to the first step not yet completed,
	// rather than simply step+1. This is correct even after re-entry from a
	// partially-completed flow where some later steps are already done.
	nextStep := RequiredStep("")
	for _, rs := range RequiredSteps {
		if !completed[rs] {
			nextStep = rs
			break
		}
	}
	if nextStep != "" {
		p.RequiredSetup.CurrentStep = nextStep
	}
	// If all steps are complete the pointer stays at the last step (review);
	// Save() is the terminal transition rather than SaveRequiredStep.

	if err := s.saveProgress(ctx, p); err != nil {
		return Status{}, err
	}
	return s.statusFromProgress(p), nil
}

// Save writes the supplied key/value pairs to the env file (Docker mode only),
// persists v1 progress advancing to the exploration phase, then marks the
// legacy completion flag. Progress is written first so a failure on the legacy
// write does not leave the system in a misleading partially-migrated state.
//
// Save is only valid once the linear required-step flow has reached its
// terminal state (CurrentStep == review). Callers that have not walked through
// all required steps via SaveRequiredStep receive a descriptive error.
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
		// Hot-activate AI provider keys in the running process so they are
		// available immediately without a container restart.
		applyProviderKeysToProcessEnv(values)
	}

	// Update v1 progress first — marks required setup complete and transitions
	// to exploration.  Only non-secret context values are written here; secrets
	// stay in the env file.
	p, err := s.loadProgress(ctx)
	if err != nil {
		return err
	}

	// Guard: the final save is only valid once required setup has been walked
	// (CurrentStep advanced to review), required setup is already complete, OR
	// the caller provides at least one value (native-mode fast-path that allows
	// the starter-service helper to advance to exploration without walking every
	// step individually). Docker mode always enforces the step guard because
	// env-file writes are destructive.
	if p.RequiredSetup.Status != RequiredStatusComplete &&
		p.RequiredSetup.CurrentStep != RequiredStepReview &&
		(s.cfg.DockerMode || len(values) == 0) {
		return errors.New("onboarding: Save requires required setup to have reached the review step; complete all required steps first")
	}

	p.RequiredSetup.Status = RequiredStatusComplete
	p.RequiredSetup.CompletedSteps = make([]RequiredStep, len(RequiredSteps))
	copy(p.RequiredSetup.CompletedSteps, RequiredSteps)
	p.RequiredSetup.CurrentStep = RequiredStepReview
	p.Exploration.Status = ExplorationStatusActive
	// Copy non-secret provider selection into progress context so GetStatus
	// can report the chosen provider without re-reading the env file.
	if provider := strings.TrimSpace(values["AI_PROVIDER"]); provider != "" {
		p.Context.AIProviderPath = provider
	}
	if err := s.saveProgress(ctx, p); err != nil {
		return err
	}

	// Write the legacy completion flag only after v1 progress is safely stored.
	if err := s.repo.Set(ctx, s.cfg.CompletionKey, "true"); err != nil {
		return fmt.Errorf("onboarding: marking complete: %w", err)
	}
	return nil
}

// UpdateExploration updates the exploration phase based on req and returns the
// updated Status. When Dismiss=true, all pending items are marked as skipped.
func (s *Service) UpdateExploration(ctx context.Context, req ExplorationUpdate) (Status, error) {
	if s.cfg.Disabled {
		return Status{}, ErrDisabled
	}

	p, err := s.loadProgress(ctx)
	if err != nil {
		return Status{}, err
	}

	if p.RequiredSetup.Status != RequiredStatusComplete {
		return Status{}, errors.New("onboarding: UpdateExploration requires required setup to be complete")
	}

	// Validate req.State independently when provided — even without an Item.
	if req.State != "" && !validItemState(req.State) {
		return Status{}, fmt.Errorf("onboarding: UpdateExploration: unknown state %q", req.State)
	}
	// A non-empty State without an Item is not a meaningful operation.
	if req.State != "" && req.Item == "" && !req.Dismiss {
		return Status{}, errors.New("onboarding: UpdateExploration: state requires item")
	}

	if req.Dismiss {
		for item, state := range p.Exploration.Items {
			if state == ItemStatePending || state == ItemStateBlocked {
				p.Exploration.Items[item] = ItemStateSkipped
			}
		}
		p.Exploration.Dismissed = true
		// Only stamp CompletedAt on the first transition to complete.
		if p.Exploration.Status != ExplorationStatusComplete {
			p.Exploration.CompletedAt = time.Now().UTC().Format(time.RFC3339)
		}
		p.Exploration.Status = ExplorationStatusComplete
	} else if req.Item != "" {
		if !validExplorationItem(req.Item) {
			return Status{}, fmt.Errorf("onboarding: UpdateExploration: unknown item %q", req.Item)
		}
		// Only apply state when it is explicitly provided; do not silently
		// default a missing state to completed.
		if req.State == "" {
			return Status{}, errors.New("onboarding: UpdateExploration: state is required when item is set")
		}
		p.Exploration.Items[req.Item] = req.State
		// Recompute the current item pointer to the first non-terminal item.
		next := firstPendingItem(p.Exploration.Items)
		p.Exploration.CurrentItem = next
		if next == "" {
			// All items terminal: mark complete, stamp timestamp only on
			// the first transition.
			if p.Exploration.Status != ExplorationStatusComplete {
				p.Exploration.CompletedAt = time.Now().UTC().Format(time.RFC3339)
			}
			p.Exploration.Status = ExplorationStatusComplete
		} else {
			// At least one item is non-terminal: exploration is (re-)opened.
			// Clear any prior completion state so the phase reflects active.
			p.Exploration.Status = ExplorationStatusActive
			p.Exploration.Dismissed = false
			p.Exploration.CompletedAt = ""
		}
	}

	if req.SelectedProjectID != "" {
		p.Context.SelectedProjectID = req.SelectedProjectID
	}

	if err := s.saveProgress(ctx, p); err != nil {
		return Status{}, err
	}
	return s.statusFromProgress(p), nil
}

// CreateStarterProject creates a starter project and query idempotently,
// then updates onboarding progress to reflect the starter items.
// It is implemented in starter.go.
func (s *Service) CreateStarterProject(ctx context.Context, req StarterProjectRequest) (StarterProjectResult, error) {
	return createStarterProject(ctx, s, req)
}

// Reset clears onboarding state according to mode:
//   - ResetModeProgress (or ""): deletes both the v1 state key and the legacy
//     completion key, returning to a brand-new-install state.
//   - ResetModeExploration: resets only the exploration portion of the v1
//     progress record to the active state, leaving required-setup intact.
//   - ResetModeDismissExploration: delegates to UpdateExploration(Dismiss=true),
//     applying the same disabled/required-setup guards.
func (s *Service) Reset(ctx context.Context, mode ResetMode) error {
	switch mode {
	case ResetModeExploration:
		p, err := s.loadProgress(ctx)
		if err != nil {
			return err
		}
		fresh := NewProgress()
		p.Exploration = fresh.Exploration
		// After exploration reset the status must be active (not not_started),
		// because required setup is already complete.
		p.Exploration.Status = ExplorationStatusActive
		// Clear exploration-derived context while keeping required-setup context.
		p.Context.SelectedProjectID = ""
		return s.saveProgress(ctx, p)

	case ResetModeDismissExploration:
		// Delegate to UpdateExploration so the disabled and required-setup
		// guards are applied identically — no bypass.
		_, err := s.UpdateExploration(ctx, ExplorationUpdate{Dismiss: true})
		return err

	case ResetModeProgress, "":
		// Empty string is accepted as a backward-compatible alias for progress reset.
		if err := s.repo.Delete(ctx, s.cfg.StateKey); err != nil {
			return fmt.Errorf("onboarding: reset (state key): %w", err)
		}
		if err := s.repo.Delete(ctx, s.cfg.CompletionKey); err != nil {
			return fmt.Errorf("onboarding: reset (completion key): %w", err)
		}
		return nil

	default:
		return fmt.Errorf("onboarding: Reset: unknown mode %q", mode)
	}
}

// DebugProgressJSONForTest returns the raw JSON string stored under StateKey,
// or an empty string if no state has been saved yet. It is intended only for
// use in tests that need to assert the serialised form of progress does not
// contain secrets.
func (s *Service) DebugProgressJSONForTest(ctx context.Context) string {
	raw, _, _ := s.repo.Get(ctx, s.cfg.StateKey)
	return raw
}
