package app

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/kyle/scout/open-core/apps/api/internal/templates"
	"github.com/kyle/scout/open-core/apps/api/internal/usage"
)

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
	sched := scheduler.New(repo, runner, cfg.Location)

	moduleRegistry := modules.NewRegistry(modules.CoreResearchModule{})
	entitlementResolver := entitlements.NewLocalResolver(cfg.RuntimeMode, moduleRegistry.Statuses())
	templateCatalog := templates.NewLocalCatalog(entitlementResolver)

	onboardingCfg, err := onboarding.LoadConfig()
	if err != nil {
		panic("onboarding: failed to load config: " + err.Error())
	}
	onboardingCfg.DBPath = cfg.DBPath
	onboardingSvc, err := onboarding.NewService(onboardingCfg, settings.NewRepository(db), repo)
	if err != nil {
		panic("onboarding: failed to construct service: " + err.Error())
	}

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
		ModelService:      services.NewModelService(repo, cfg.RuntimeMode, entitlementResolver),
		ReportService:     services.NewReportService(repo, db, aiSvc, cfg.RuntimeMode, entitlementResolver),
		GoogleService:     services.NewGoogleService(repo),
		ScoutService:      services.NewScoutService(repo, runner, cfg.RuntimeMode, entitlementResolver),
		ScheduleService:   services.NewScheduleService(repo, sched),
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
