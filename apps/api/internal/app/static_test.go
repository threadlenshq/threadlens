package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// buildTempDist creates a minimal frontend dist directory for testing.
func buildTempDist(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// index.html
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html><body>App</body></html>"), 0644); err != nil {
		t.Fatal(err)
	}

	// assets/app.js
	assetsDir := filepath.Join(dir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "app.js"), []byte("console.log('app')"), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func TestStaticAssetServing(t *testing.T) {
	distDir := buildTempDist(t)
	database := testhelpers.OpenTestDB(t)

	cfg := LoadConfig()
	cfg.FrontendDist = distDir
	a := New(cfg, database)

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /assets/app.js: status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if body != "console.log('app')" {
		t.Fatalf("GET /assets/app.js: body = %q, want js content", body)
	}
}

func TestStaticSPAFallback(t *testing.T) {
	distDir := buildTempDist(t)
	database := testhelpers.OpenTestDB(t)

	cfg := LoadConfig()
	cfg.FrontendDist = distDir
	a := New(cfg, database)

	req := httptest.NewRequest(http.MethodGet, "/some/client/route", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /some/client/route: status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if body != "<html><body>App</body></html>" {
		t.Fatalf("GET /some/client/route: body = %q, want index.html content", body)
	}
}

func TestStaticAPINotFoundJSON(t *testing.T) {
	distDir := buildTempDist(t)
	database := testhelpers.OpenTestDB(t)

	cfg := LoadConfig()
	cfg.FrontendDist = distDir
	a := New(cfg, database)

	req := httptest.NewRequest(http.MethodGet, "/api/missing", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /api/missing: status = %d, want 404", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("GET /api/missing: response is not JSON: %v", err)
	}
	if body["error"] != "Not found" {
		t.Fatalf("GET /api/missing: error = %q, want \"Not found\"", body["error"])
	}
}
