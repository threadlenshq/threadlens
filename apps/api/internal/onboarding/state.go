package onboarding

import "time"

// Phase represents the current onboarding phase.
type Phase string

const (
	PhaseDisabled      Phase = "disabled"
	PhaseRequiredSetup Phase = "required_setup"
	PhaseExploration   Phase = "exploration"
	PhaseComplete      Phase = "complete"
)

// RequiredStep represents a step in the required setup flow.
type RequiredStep string

const (
	RequiredStepWelcome     RequiredStep = "welcome"
	RequiredStepAIProvider  RequiredStep = "ai_provider"
	RequiredStepAppDatabase RequiredStep = "app_database"
	RequiredStepReview      RequiredStep = "review"
)

// RequiredStatus represents the status of required setup.
type RequiredStatus string

const (
	RequiredStatusNotStarted RequiredStatus = "not_started"
	RequiredStatusActive     RequiredStatus = "active"
	RequiredStatusComplete   RequiredStatus = "complete"
)

// ExplorationStatus represents the status of the exploration phase.
type ExplorationStatus string

const (
	ExplorationStatusNotStarted ExplorationStatus = "not_started"
	ExplorationStatusActive     ExplorationStatus = "active"
	ExplorationStatusComplete   ExplorationStatus = "complete"
)

// ExplorationItem represents an item in the exploration checklist.
type ExplorationItem string

const (
	ExplorationItemStarterProject ExplorationItem = "starter_project"
	ExplorationItemStarterQuery   ExplorationItem = "starter_query"
	ExplorationItemFirstScout     ExplorationItem = "first_scout"
	ExplorationItemReviewResults  ExplorationItem = "review_results"
	ExplorationItemReportsIntro   ExplorationItem = "reports_intro"
	ExplorationItemSettingsIntro  ExplorationItem = "settings_intro"
)

// ItemState represents the state of an exploration item.
type ItemState string

const (
	ItemStatePending   ItemState = "pending"
	ItemStateCompleted ItemState = "completed"
	ItemStateSkipped   ItemState = "skipped"
	ItemStateBlocked   ItemState = "blocked"
)

// RequiredSteps is the ordered list of required setup steps.
var RequiredSteps = []RequiredStep{
	RequiredStepWelcome,
	RequiredStepAIProvider,
	RequiredStepAppDatabase,
	RequiredStepReview,
}

// ExplorationItems is the ordered list of exploration items.
var ExplorationItems = []ExplorationItem{
	ExplorationItemStarterProject,
	ExplorationItemStarterQuery,
	ExplorationItemFirstScout,
	ExplorationItemReviewResults,
	ExplorationItemReportsIntro,
	ExplorationItemSettingsIntro,
}

// Progress is the full onboarding progress state stored in the KV store.
type Progress struct {
	Version       int                `json:"version"`
	RequiredSetup RequiredSetupState `json:"requiredSetup"`
	Exploration   ExplorationState   `json:"exploration"`
	Context       OnboardingContext  `json:"context"`
	CreatedAt     string             `json:"createdAt"`
	UpdatedAt     string             `json:"updatedAt"`
}

// RequiredSetupState holds progress through the required setup steps.
type RequiredSetupState struct {
	Status          RequiredStatus `json:"status"`
	CurrentStep     RequiredStep   `json:"currentStep"`
	CompletedSteps  []RequiredStep `json:"completedSteps"`
	CompletedAt     string         `json:"completedAt"`
	RestartRequired bool           `json:"restartRequired"`
}

// ExplorationState holds progress through the exploration checklist.
type ExplorationState struct {
	Status      ExplorationStatus          `json:"status"`
	CurrentItem ExplorationItem            `json:"currentItem"`
	Items       map[ExplorationItem]ItemState `json:"items"`
	Dismissed   bool                       `json:"dismissed"`
	CompletedAt string                     `json:"completedAt"`
}

// OnboardingContext holds contextual IDs set during onboarding.
type OnboardingContext struct {
	SelectedProjectID string `json:"selectedProjectId"`
	StarterProjectID  string `json:"starterProjectId"`
	StarterQueryID    int64  `json:"starterQueryId"`
	AIProviderPath    string `json:"aiProviderPath"`
}

// StepView is a display-ready view of a required setup step.
type StepView struct {
	ID        RequiredStep `json:"id"`
	Label     string       `json:"label"`
	Completed bool         `json:"completed"`
	Current   bool         `json:"current"`
}

// ItemView is a display-ready view of an exploration item.
type ItemView struct {
	ID    ExplorationItem `json:"id"`
	Label string          `json:"label"`
	State ItemState       `json:"state"`
}

// Capabilities describes what providers and sources are available.
type Capabilities struct {
	Providers []ProviderCapability `json:"providers"`
	Sources   SourceCapabilities   `json:"sources"`
}

// ProviderCapability describes a single AI provider.
type ProviderCapability struct {
	ID                   string `json:"id"`
	Label                string `json:"label"`
	Configured           bool   `json:"configured"`
	RequiresSecret       bool   `json:"requiresSecret"`
	WritableInOnboarding bool   `json:"writableInOnboarding"`
	MaskedValue          string `json:"maskedValue"`
}

// SourceCapabilities describes which data sources are ready to use.
type SourceCapabilities struct {
	RedditReady  bool `json:"redditReady"`
	GoogleReady  bool `json:"googleReady"`
	BlueskyReady bool `json:"blueskyReady"`
}

// AppDatabaseStatus describes the current app database configuration state.
type AppDatabaseStatus struct {
	DatabasePathLabel string `json:"databasePathLabel"`
	RuntimeMode       string `json:"runtimeMode"`
	EnvFileLabel      string `json:"envFileLabel"`
	EnvWritable       bool   `json:"envWritable"`
	RestartRequired   bool   `json:"restartRequired"`
}

// Status is the full onboarding status returned to the frontend.
type Status struct {
	Enabled                bool              `json:"enabled"`
	Complete               bool              `json:"complete"`
	RequiredSetupComplete  bool              `json:"requiredSetupComplete"`
	ExplorationComplete    bool              `json:"explorationComplete"`
	Phase                  Phase             `json:"phase"`
	CurrentRequiredStep    RequiredStep      `json:"currentRequiredStep"`
	CurrentExplorationItem ExplorationItem   `json:"currentExplorationItem"`
	Steps                  []StepView        `json:"steps"`
	Items                  []ItemView        `json:"items"`
	Capabilities           Capabilities      `json:"capabilities"`
	AppDatabase            AppDatabaseStatus `json:"appDatabase"`
	Context                OnboardingContext `json:"context"`
	EnvFilePath            string            `json:"-"`
}

// NewProgress returns a fresh Progress with sensible defaults.
func NewProgress() Progress {
	now := time.Now().UTC().Format(time.RFC3339)
	items := make(map[ExplorationItem]ItemState, len(ExplorationItems))
	for _, item := range ExplorationItems {
		items[item] = ItemStatePending
	}
	return Progress{
		Version: 1,
		RequiredSetup: RequiredSetupState{
			Status:         RequiredStatusNotStarted,
			CurrentStep:    RequiredStepWelcome,
			CompletedSteps: []RequiredStep{},
		},
		Exploration: ExplorationState{
			Status:      ExplorationStatusNotStarted,
			CurrentItem: ExplorationItemStarterProject,
			Items:       items,
		},
		Context: OnboardingContext{
			AIProviderPath: "anthropic",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// PhaseForProgress derives the current phase from enabled flag and progress.
func PhaseForProgress(enabled bool, p Progress) Phase {
	if !enabled {
		return PhaseDisabled
	}
	if p.RequiredSetup.Status != RequiredStatusComplete {
		return PhaseRequiredSetup
	}
	if ExplorationComplete(p.Exploration.Items) || p.Exploration.Dismissed || p.Exploration.Status == ExplorationStatusComplete {
		return PhaseComplete
	}
	return PhaseExploration
}

// ExplorationComplete returns true when every canonical exploration item is
// present in items and each is either completed or skipped. A sparse/partial
// map (missing items) is treated as incomplete.
func ExplorationComplete(items map[ExplorationItem]ItemState) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range ExplorationItems {
		state, ok := items[item]
		if !ok {
			return false
		}
		if state != ItemStateCompleted && state != ItemStateSkipped {
			return false
		}
	}
	return true
}

// MarkUpdated sets UpdatedAt to the current UTC time.
func MarkUpdated(p *Progress) {
	p.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
}

// stringInRequiredSteps returns true if the step is a known required step.
func stringInRequiredSteps(step RequiredStep) bool {
	for _, s := range RequiredSteps {
		if s == step {
			return true
		}
	}
	return false
}
