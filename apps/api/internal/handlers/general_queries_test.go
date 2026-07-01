package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newGeneralQueryRouter(t *testing.T) http.Handler {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	if _, err := repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Proj", Mode: "research"}); err != nil {
		t.Fatal(err)
	}
	svc := services.NewGeneralQueryService(repo)
	r := chi.NewRouter()
	handlers.MountGeneralQueryRoutes(r, svc)
	return r
}

func TestGeneralQueryRoutes_CreateAndList(t *testing.T) {
	router := newGeneralQueryRouter(t)

	body, _ := json.Marshal(map[string]any{
		"query_text": "manual invoicing",
		"angle":      "pain",
		"platforms":  []string{"reddit", "google"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/projects/p1/general-queries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/projects/p1/general-queries", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: want 200, got %d", rr.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 general query, got %d", len(got))
	}
}
