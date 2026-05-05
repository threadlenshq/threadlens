package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kyle/scout/open-core/apps/api/internal/handlers"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/services"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newInsightsRouter(t *testing.T) (http.Handler, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	svc := services.NewInsightsService(repo)
	r := chi.NewRouter()
	handlers.MountInsightsRoutes(r, svc)
	return r, repo
}

func TestInsights_EmptyDB(t *testing.T) {
	r, _ := newInsightsRouter(t)
	rr := doRequest(t, r, http.MethodGet, "/api/insights", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	// Required keys
	for _, key := range []string{"total", "by_angle", "by_type", "score_distribution", "top_posts", "top_keywords"} {
		if _, ok := result[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
	if result["total"].(float64) != 0 {
		t.Fatalf("expected total=0, got %v", result["total"])
	}
}

func TestInsights_Aggregates(t *testing.T) {
	r, repo := newInsightsRouter(t)

	// Seed a project and posts directly.
	_, err := repo.DB.Exec(`INSERT INTO projects (id, name, mode) VALUES ('p1', 'Test', 'research')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.DB.Exec(`
		INSERT INTO posts (id, project_id, platform, title, body, final_score, angle, why, engagement_type)
		VALUES
		  ('post1', 'p1', 'reddit', 'Title1', 'Body1', 9.0, 'SaaS', 'pain frustration workaround tool', 'karma'),
		  ('post2', 'p1', 'reddit', 'Title2', 'Body2', 6.0, 'SaaS', 'pain seeking solution', 'karma'),
		  ('post3', 'p1', 'bluesky', 'Title3', 'Body3', 3.0, 'SEO', 'workaround tool pain', 'product')
	`)
	if err != nil {
		t.Fatal(err)
	}

	rr := doRequest(t, r, http.MethodGet, "/api/insights", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if result["total"].(float64) != 3 {
		t.Fatalf("expected total=3, got %v", result["total"])
	}

	// score_distribution: high=1 (9.0), medium=1 (6.0), low=1 (3.0)
	dist := result["score_distribution"].(map[string]any)
	if dist["high"].(float64) != 1 {
		t.Fatalf("expected high=1, got %v", dist["high"])
	}
	if dist["medium"].(float64) != 1 {
		t.Fatalf("expected medium=1, got %v", dist["medium"])
	}
	if dist["low"].(float64) != 1 {
		t.Fatalf("expected low=1, got %v", dist["low"])
	}

	// by_angle has entries
	byAngle := result["by_angle"].([]any)
	if len(byAngle) == 0 {
		t.Fatal("expected non-empty by_angle")
	}

	// top_posts: max 10 items
	topPosts := result["top_posts"].([]any)
	if len(topPosts) != 3 {
		t.Fatalf("expected 3 top posts, got %d", len(topPosts))
	}
	// First post should have highest score (9.0)
	firstPost := topPosts[0].(map[string]any)
	if firstPost["final_score"].(float64) != 9.0 {
		t.Fatalf("expected first post final_score=9.0, got %v", firstPost["final_score"])
	}

	// top_keywords: "pain" appears 3 times
	topKW := result["top_keywords"].([]any)
	if len(topKW) == 0 {
		t.Fatal("expected non-empty top_keywords")
	}
	first := topKW[0].(map[string]any)
	if first["word"].(string) != "pain" {
		t.Fatalf("expected first keyword=pain, got %v", first["word"])
	}
}

func TestInsights_ProjectFilter(t *testing.T) {
	r, repo := newInsightsRouter(t)

	repo.DB.Exec(`INSERT INTO projects (id, name, mode) VALUES ('pa', 'A', 'research'), ('pb', 'B', 'research')`)
	repo.DB.Exec(`
		INSERT INTO posts (id, project_id, platform, title, body, final_score)
		VALUES
		  ('posta1', 'pa', 'reddit', 'T', 'B', 5.0),
		  ('postb1', 'pb', 'reddit', 'T', 'B', 5.0)
	`)

	rr := doRequest(t, r, http.MethodGet, "/api/insights?project_id=pa", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["total"].(float64) != 1 {
		t.Fatalf("expected total=1 with project filter, got %v", result["total"])
	}
}

func TestInsights_InvalidSince(t *testing.T) {
	r, _ := newInsightsRouter(t)
	rr := doRequest(t, r, http.MethodGet, "/api/insights?since=invalid", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid since, got %d", rr.Code)
	}
}

func TestInsights_InvalidMinScore(t *testing.T) {
	r, _ := newInsightsRouter(t)
	rr := doRequest(t, r, http.MethodGet, "/api/insights?min_score=abc", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid min_score, got %d", rr.Code)
	}
}

func TestInsights_MinScoreFilter(t *testing.T) {
	r, repo := newInsightsRouter(t)
	repo.DB.Exec(`INSERT INTO projects (id, name, mode) VALUES ('pm', 'M', 'research')`)
	repo.DB.Exec(`
		INSERT INTO posts (id, project_id, platform, title, body, final_score)
		VALUES
		  ('pm1', 'pm', 'reddit', 'T', 'B', 9.0),
		  ('pm2', 'pm', 'reddit', 'T', 'B', 3.0)
	`)

	rr := doRequest(t, r, http.MethodGet, "/api/insights?min_score=8", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result map[string]any
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result["total"].(float64) != 1 {
		t.Fatalf("expected total=1 with min_score=8 filter, got %v", result["total"])
	}
}
