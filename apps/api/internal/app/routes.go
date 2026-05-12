package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/onboarding"
)

func (a *App) mountRoutes() {
	a.Router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "api-go"})
	})

	handlers.MountRuntimeRoutes(a.Router, a.RuntimeService)
	a.ModuleRegistry.MountRoutes(a.Router)
	onboarding.MountRoutes(a.Router, a.OnboardingService)

	handlers.MountInsightsRoutes(a.Router, a.InsightsService)
	handlers.MountProjectRoutes(a.Router, a.ProjectService)
	handlers.MountQueryRoutes(a.Router, a.QueryService)
	handlers.MountPromptRoutes(a.Router, a.PromptService)
	handlers.MountPostRoutes(a.Router, a.PostService)
	handlers.MountModelRoutes(a.Router, a.ModelService)
	handlers.MountReportRoutes(a.Router, a.ReportService)
	handlers.MountGoogleRoutes(a.Router, a.GoogleService)
	handlers.MountScoutRoutes(a.Router, a.ScoutService)
	handlers.MountScheduleRoutes(a.Router, a.ScheduleService)

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
