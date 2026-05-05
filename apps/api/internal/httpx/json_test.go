package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONSetsStatusAndContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSON(rr, http.StatusCreated, map[string]any{"ok": true})

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	var body map[string]bool
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body["ok"] {
		t.Fatalf("body = %#v, want ok true", body)
	}
}

func TestWriteErrorUsesExpressShape(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusBadRequest, "id, name, and mode are required")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if got := rr.Body.String(); got != "{\"error\":\"id, name, and mode are required\"}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestParsePositiveInt(t *testing.T) {
	if got := ParsePositiveInt("abc", 20, 100); got != 20 {
		t.Fatalf("invalid = %d, want default 20", got)
	}
	if got := ParsePositiveInt("200", 20, 100); got != 100 {
		t.Fatalf("clamped = %d, want 100", got)
	}
	if got := ParsePositiveInt("7", 20, 100); got != 7 {
		t.Fatalf("valid = %d, want 7", got)
	}
}
