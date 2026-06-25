package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// ManualScoutHandler handles manual scouting of individual posts.
type ManualScoutHandler struct {
	svc *services.ManualScoutService
}

// NewManualScoutHandler creates a new ManualScoutHandler.
func NewManualScoutHandler(svc *services.ManualScoutService) *ManualScoutHandler {
	return &ManualScoutHandler{svc: svc}
}

// Mount registers the manual scout routes on the provided router.
func (h *ManualScoutHandler) Mount(r chi.Router) {
	r.Post("/api/projects/{id}/manual-scout", h.handleScoutPost)
	r.Post("/api/projects/{id}/manual-scout/commit", h.handleCommitDecision)
}

func (h *ManualScoutHandler) handleScoutPost(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	var body struct {
		URL      string `json:"url"`
		Platform string `json:"platform"`
	}
	_ = httpx.DecodeJSON(r, &body)

	result, status, msg := h.svc.ScoutPost(r.Context(), projectID, body.URL, body.Platform)
	if msg != "" {
		httpx.WriteError(w, status, msg)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *ManualScoutHandler) handleCommitDecision(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	var body struct {
		Decision string            `json:"decision"`
		Post     services.PostData `json:"post"`
	}
	_ = httpx.DecodeJSON(r, &body)

	result, status, msg := h.svc.CommitDecision(r.Context(), projectID, body.Decision, body.Post)
	if msg != "" {
		httpx.WriteError(w, status, msg)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// MountManualScoutRoutes registers all manual scout routes onto the provided router.
// It is a convenience wrapper around ManualScoutHandler for consistency with other
// handler packages.
func MountManualScoutRoutes(r chi.Router, svc *services.ManualScoutService) {
	h := NewManualScoutHandler(svc)
	h.Mount(r)
}
