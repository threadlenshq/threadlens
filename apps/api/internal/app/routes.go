package app

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
	"github.com/kyle/scout/open-core/apps/api/internal/telemetry"
)

func (a *App) mountRoutes() {
	a.Router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "api-go"})
	})

	handlers.MountRuntimeRoutes(a.Router, a.RuntimeService)
	a.ModuleRegistry.MountRoutes(a.Router)
	onboarding.MountRoutes(a.Router, a.OnboardingService, a.TelemetryRecorder)
	telemetry.MountRoutes(a.Router, a.SettingsRepo, a.TelemetryRecorder, telemetry.TelemetryStatusConfig{
		EnvOptIn:       a.Config.Telemetry.EnvOptIn,
		ScoutVersion:   scoutVersion,
		DeploymentType: telemetry.DetectDeploymentType(),
		OSPlatform:     detectOSPlatform(),
	})

	handlers.MountInsightsRoutes(a.Router, a.InsightsService)
	handlers.MountProjectRoutes(a.Router, a.ProjectService)
	handlers.MountQueryRoutes(a.Router, a.QueryService, a.TelemetryRecorder)
	handlers.MountQueryReviewJobRoutes(a.Router, a.Repo, a.QueryService)
	handlers.MountFilterRoutes(a.Router, a.Repo, a.FilterClassifier, a.TelemetryRecorder)
	handlers.MountPromptRoutes(a.Router, a.PromptService)
	handlers.MountPostRoutes(a.Router, a.PostService)
	handlers.MountModelRoutes(a.Router, a.ModelService)
	handlers.MountReportRoutes(a.Router, a.ReportService, a.TelemetryRecorder)
	handlers.MountGoogleRoutes(a.Router, a.GoogleService)
	handlers.MountScoutRoutes(a.Router, a.ScoutService, a.TelemetryRecorder)
	handlers.MountScheduleRoutes(a.Router, a.ScheduleService, a.TelemetryRecorder)

	distDir := a.Config.FrontendDist
	fileServer := http.FileServer(http.Dir(distDir))

	a.Router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			httpx.WriteError(w, http.StatusNotFound, "Not found")
			return
		}

		// Try to serve an existing file from the dist directory
		filePath := filepath.Join(distDir, filepath.FromSlash(r.URL.Path))
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// For non-API GET requests, fall back to index.html (SPA routing)
		if r.Method == http.MethodGet {
			indexPath := filepath.Join(distDir, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
				return
			}
		}

		http.NotFound(w, r)
	})
}

func detectOSPlatform() string {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		return runtime.GOOS
	default:
		return "unknown"
	}
}
