package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	"github.com/kyle/scout/open-core/apps/api/internal/templates"
)

func newRuntimeRouter() http.Handler {
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	catalog := templates.NewLocalCatalog(resolver)
	svc := services.NewRuntimeService(entitlements.RuntimeModeSelfHosted, resolver, catalog)
	r := chi.NewRouter()
	handlers.MountRuntimeRoutes(r, svc)
	return r
}

func TestRuntimeCapabilitiesRoute(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "")
	router := newRuntimeRouter()
	rr := doRequest(t, router, http.MethodGet, "/api/runtime/capabilities", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["runtimeMode"] != "self_hosted" {
		t.Fatalf("runtimeMode = %v, want self_hosted", resp["runtimeMode"])
	}
	capabilities := resp["capabilities"].(map[string]any)
	if capabilities["core.scout.run.reddit"] != true {
		t.Fatalf("core.scout.run.reddit must be true: %+v", capabilities)
	}
	if capabilities["core.scout.run.google"] != false {
		t.Fatalf("core.scout.run.google must be false when PARALLEL_API_KEY is unset: %+v", capabilities)
	}
}

func TestRuntimeCapabilitiesRouteWithParallelKey(t *testing.T) {
	t.Setenv("PARALLEL_API_KEY", "parallel_test_key")
	router := newRuntimeRouter()
	rr := doRequest(t, router, http.MethodGet, "/api/runtime/capabilities", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	capabilities := resp["capabilities"].(map[string]any)
	if capabilities["core.scout.run.google"] != true {
		t.Fatalf("core.scout.run.google must be true when PARALLEL_API_KEY is set: %+v", capabilities)
	}
}

func TestPromptPackListRoute(t *testing.T) {
	router := newRuntimeRouter()
	rr := doRequest(t, router, http.MethodGet, "/api/templates/prompt-packs", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["packs"].([]any); !ok {
		t.Fatalf("packs must be an array: %+v", resp)
	}
}
