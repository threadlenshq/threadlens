package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/httpx"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
)

// MountModelRoutes registers the /api/models routes on the given router.
func MountModelRoutes(r chi.Router, svc *services.ModelService) {
	r.Get("/api/models/catalog", makeHandleGetModelCatalog(svc))
	r.Get("/api/models/config", makeHandleGetModelConfig(svc))
	r.Put("/api/models/config/{taskId}", makeHandlePutModelConfig(svc))
	r.Delete("/api/models/config/{taskId}", makeHandleDeleteModelConfig(svc))
}

func makeHandleGetModelCatalog(svc *services.ModelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		catalog, err := svc.Catalog(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, catalog)
	}
}

func makeHandleGetModelConfig(svc *services.ModelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config, err := svc.GetConfig(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, config)
	}
}

func makeHandlePutModelConfig(svc *services.ModelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := chi.URLParam(r, "taskId")

		var body struct {
			ModelID string `json:"modelId"`
		}
		if err := httpx.DecodeJSON(r, &body); err != nil || body.ModelID == "" {
			httpx.WriteError(w, http.StatusBadRequest, "modelId is required")
			return
		}

		result, err := svc.SetConfig(r.Context(), taskID, body.ModelID)
		if err != nil {
			if ok, tid := isUnknownTaskError(err); ok {
				httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Unknown task: %s", tid))
				return
			}
			if ok, mid := isUnknownModelError(err); ok {
				httpx.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Unknown model: %s", mid))
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		httpx.WriteJSON(w, http.StatusOK, result)
	}
}

func makeHandleDeleteModelConfig(svc *services.ModelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := chi.URLParam(r, "taskId")

		err := svc.DeleteConfig(r.Context(), taskID)
		if err != nil {
			if ok, tid := isUnknownTaskError(err); ok {
				// Express returns the raw error: "Unknown task id: {taskId}"
				httpx.WriteError(w, http.StatusNotFound, fmt.Sprintf("Unknown task id: %s", tid))
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// isUnknownTaskError checks if the error is a ModelError with kind "unknownTask".
func isUnknownTaskError(err error) (bool, string) {
	if e, ok := err.(*services.ModelError); ok && e.Kind == "unknownTask" {
		return true, e.TaskID
	}
	return false, ""
}

// isUnknownModelError checks if the error is a ModelError with kind "unknownModel".
func isUnknownModelError(err error) (bool, string) {
	if e, ok := err.(*services.ModelError); ok && e.Kind == "unknownModel" {
		return true, e.ModelID
	}
	return false, ""
}
