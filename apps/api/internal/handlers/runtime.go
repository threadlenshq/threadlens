package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

func MountRuntimeRoutes(r chi.Router, svc *services.RuntimeService) {
	r.Get("/api/runtime/capabilities", func(w http.ResponseWriter, r *http.Request) {
		snapshot, status, msg := svc.Snapshot(r.Context())
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, snapshot)
	})

	r.Get("/api/templates/prompt-packs", func(w http.ResponseWriter, r *http.Request) {
		packs, status, msg := svc.ListPromptPacks(r.Context())
		if msg != "" {
			httpx.WriteError(w, status, msg)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"packs": packs})
	})
}
