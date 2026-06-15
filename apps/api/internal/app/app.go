package app

import (
	"context"
	crand "crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/modules"
	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/scheduler"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
	"github.com/kyle/scout/open-core/apps/api/internal/telemetry"
	"github.com/kyle/scout/open-core/apps/api/internal/templates"
	"github.com/kyle/scout/open-core/apps/api/internal/usage"
)

// scoutVersion is the build version of the API. Override via ldflags at build time:
//   go build -ldflags "-X github.com/kyle/scout/open-core/apps/api/internal/app.scoutVersion=0.8.0"
var scoutVersion = "0.7.2-dev"

type App struct {
	Config            Config
	DB                *sql.DB
	Router            *chi.Mux
	Repo              *repository.Repository
	Scheduler         *scheduler.Scheduler
	ModuleRegistry    *modules.Registry
	OnboardingService onboarding.ServiceIface
	RuntimeService    *services.RuntimeService
	InsightsService   *services.InsightsService
	ProjectService    *services.ProjectService
	QueryService      *services.QueryService
	PromptService     *services.PromptService
	PostService       *services.PostService
	ModelService      *services.ModelService
	ReportService     *services.ReportService
	GoogleService     *services.GoogleService
	ScoutService      *services.ScoutService
	ScheduleService   *services.ScheduleService
	FilterClassifier  *pipeline.FilterClassifier
	TelemetryRecorder *telemetry.Recorder
	SettingsRepo      *settings.Repository
}

func New(cfg Config, db *sql.DB) *App {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(10 * time.Minute))
	r.Use(httpx.RecoverJSON)

	repo := repository.New(db)
	usageMeter := usage.NoopMeter{}
	aiSvc := ai.NewServiceWithUsage(repo, usageMeter)
	runner := pipeline.NewRunner(repo, aiSvc)
	filterClassifier := pipeline.NewFilterClassifier(repo, nil)
	sched := scheduler.New(repo, runner, cfg.Location)

	moduleRegistry := modules.NewRegistry(modules.CoreResearchModule{})
	entitlementResolver := entitlements.NewLocalResolver(cfg.RuntimeMode, moduleRegistry.Statuses())
	templateCatalog := templates.NewLocalCatalog(entitlementResolver)

	onboardingCfg, err := onboarding.LoadConfig()
	if err != nil {
		panic("onboarding: failed to load config: " + err.Error())
	}
	onboardingCfg.DBPath = cfg.DBPath
	settingsRepo := settings.NewRepository(db)
	modelSvc := services.NewModelService(repo, cfg.RuntimeMode, entitlementResolver)
	onboardingSvc, err := onboarding.NewService(onboardingCfg, settingsRepo, repo, modelSvc)
	if err != nil {
		panic("onboarding: failed to construct service: " + err.Error())
	}

	// Ensure instance_id exists in app_settings on first boot.
	instanceID := ensureInstanceID(settingsRepo)

	telemetryRecorder := telemetry.NewRecorder(telemetry.RecorderConfig{
		OptInMode:      cfg.Telemetry.OptInMode,
		ScoutVersion:   scoutVersion,
		DeploymentType: telemetry.DetectDeploymentType(),
		InstanceID:     instanceID,
		ConsentChecker: telemetryConsentChecker(cfg.Telemetry.OptInMode, settingsRepo),
	})

	// Fire instance_started on boot (no-op when OptInMode is "disabled").
	telemetryRecorder.Record(telemetry.EventInstanceStarted)

	a := &App{
		Config:            cfg,
		DB:                db,
		Router:            r,
		Repo:              repo,
		Scheduler:         sched,
		ModuleRegistry:    moduleRegistry,
		OnboardingService: onboardingSvc,
		RuntimeService:    services.NewRuntimeService(cfg.RuntimeMode, entitlementResolver, templateCatalog),
		InsightsService:   services.NewInsightsService(repo),
		ProjectService:    services.NewProjectService(repo, cfg.RuntimeMode, entitlementResolver),
		QueryService:      services.NewQueryService(repo, aiSvc),
		PromptService:     services.NewPromptService(repo),
		PostService:       services.NewPostServiceFull(repo, aiSvc, redditContextFetcher{}, blueskyReplierAdapter{}),
		ModelService:      modelSvc,
		ReportService:     services.NewReportService(repo, db, aiSvc, cfg.RuntimeMode, entitlementResolver),
		GoogleService:     services.NewGoogleService(repo),
		ScoutService:      services.NewScoutService(repo, runner, cfg.RuntimeMode, entitlementResolver),
		ScheduleService:   services.NewScheduleService(repo, sched),
		FilterClassifier:  filterClassifier,
		TelemetryRecorder: telemetryRecorder,
		SettingsRepo:      settingsRepo,
	}
	a.mountRoutes()
	return a
}

func (a *App) Handler() http.Handler {
	return a.Router
}

type redditContextFetcher struct{}

func (redditContextFetcher) FetchRedditContext(rctx context.Context, postURL string) (services.RedditContext, error) {
	ctx, err := pipeline.FetchRedditContext(rctx, postURL)
	if err != nil {
		return services.RedditContext{}, err
	}
	comments := make([]services.RedditTopComment, len(ctx.TopComments))
	for i, comment := range ctx.TopComments {
		comments[i] = services.RedditTopComment{
			Author: comment.Author,
			Body:   comment.Body,
			Score:  comment.Score,
		}
	}
	return services.RedditContext{FullBody: ctx.FullBody, TopComments: comments}, nil
}

// blueskyReplierAdapter adapts pipeline.PostBlueskyReply to services.BlueskyReplier.
type blueskyReplierAdapter struct{}

func (blueskyReplierAdapter) PostBlueskyReply(ctx context.Context, handle, appPassword, text, parentURI, parentCID string) (json.RawMessage, error) {
	return pipeline.PostBlueskyReply(ctx, handle, appPassword, text, parentURI, parentCID)
}

// telemetryConsentChecker returns a consent checker that respects the
// tri-state OptInMode. When mode is "enabled", consent is always "granted";
// otherwise the stored user choice is used.
func telemetryConsentChecker(mode string, repo *settings.Repository) func() string {
	if mode == "enabled" {
		return func() string { return "granted" }
	}
	return func() string { return telemetry.ReadConsentChoice(repo) }
}

// ensureInstanceID writes a fresh UUID to app_settings if the key is absent,
// and returns the instance ID in all cases.
func ensureInstanceID(repo *settings.Repository) string {
	ctx := context.Background()
	val, found, err := repo.Get(ctx, telemetry.SettingsKeyInstanceID)
	if err == nil && found {
		return val
	}
	id := generateUUID()
	_ = repo.Set(ctx, telemetry.SettingsKeyInstanceID, id)
	return id
}

// generateUUID returns a random v4 UUID string.
func generateUUID() string {
	var uuid [16]byte
	if _, err := crand.Read(uuid[:]); err != nil {
		panic("telemetry: crypto/rand read failed: " + err.Error())
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
