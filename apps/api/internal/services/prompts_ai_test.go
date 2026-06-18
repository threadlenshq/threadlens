package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	testingpkg "github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

// stubAIProvider is a controllable fake for ai.Provider.
type stubAIProvider struct {
	name   string
	result string
	err    error
}

func (f *stubAIProvider) Name() string    { return f.name }
func (f *stubAIProvider) Available() bool { return true }
func (f *stubAIProvider) Generate(_ context.Context, _ string, _ string, _ string, _ time.Duration) (string, error) {
	return f.result, f.err
}

// recordingProvider captures the user message passed to Generate.
type recordingProvider struct {
	name        string
	result      string
	err         error
	lastUserMsg string
}

func (f *recordingProvider) Name() string    { return f.name }
func (f *recordingProvider) Available() bool { return true }
func (f *recordingProvider) Generate(_ context.Context, _ string, _ string, userMsg string, _ time.Duration) (string, error) {
	f.lastUserMsg = userMsg
	return f.result, f.err
}

func newTestPromptService(t *testing.T, aiResult string, aiErr error) (*PromptService, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	provider := &stubAIProvider{name: "copilot", result: aiResult, err: aiErr}
	aiSvc := ai.NewServiceWithProviders([]ai.Provider{provider})
	return NewPromptService(repo, aiSvc), repo
}

func newTestPromptServiceWithRecorder(t *testing.T, rec *recordingProvider) (*PromptService, *repository.Repository) {
	t.Helper()
	db := testingpkg.OpenTestDB(t)
	repo := repository.New(db)
	aiSvc := ai.NewServiceWithProviders([]ai.Provider{rec})
	return NewPromptService(repo, aiSvc), repo
}

func validSuggestionJSON() string {
	return `[
		{"text":"Write a Reddit post about finding relief from chronic back pain with our ergonomic chair.","label":"Back pain angle"},
		{"text":"Draft a Reddit post highlighting how our standing desk converter helps remote workers stay active.","label":"Standing desk"},
		{"text":"Create a Reddit post about the hidden costs of cheap office chairs and why quality matters.","label":"Quality matters"}
	]`
}

// Test 1: Missing project returns 404
func TestPromptSuggest_MissingProject_Returns404(t *testing.T) {
	svc, _ := newTestPromptService(t, validSuggestionJSON(), nil)
	_, status, msg := svc.Suggest(context.Background(), "nonexistent", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if status != 404 {
		t.Fatalf("status = %d, want 404; msg = %q", status, msg)
	}
	if msg != "Project not found" {
		t.Fatalf("msg = %q, want 'Project not found'", msg)
	}
}

// Test 2: Invalid platform returns 400
func TestPromptSuggest_InvalidPlatform_Returns400(t *testing.T) {
	svc, repo := newTestPromptService(t, validSuggestionJSON(), nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	_, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "google", Type: "product"})
	if status != 400 {
		t.Fatalf("status = %d, want 400; msg = %q", status, msg)
	}
}

// Test 3: Invalid type returns 400
func TestPromptSuggest_InvalidType_Returns400(t *testing.T) {
	svc, repo := newTestPromptService(t, validSuggestionJSON(), nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	_, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "marketing"})
	if status != 400 {
		t.Fatalf("status = %d, want 400; msg = %q", status, msg)
	}
}

// Test 4: Happy path returns 3 suggestions
func TestPromptSuggest_HappyPath_Returns3(t *testing.T) {
	svc, repo := newTestPromptService(t, validSuggestionJSON(), nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(resp.Suggestions))
	}
	for i, s := range resp.Suggestions {
		if s.Text == "" {
			t.Fatalf("suggestion[%d].Text is empty", i)
		}
		if s.Label == "" {
			t.Fatalf("suggestion[%d].Label is empty", i)
		}
	}
}

// Test 5: All-providers-failed returns 200 with empty list and notice
func TestPromptSuggest_AllProvidersFailed_ReturnsNotice(t *testing.T) {
	svc, repo := newTestPromptService(t, "", errors.New("all AI providers failed: no provider configured"))
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error msg: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 0 {
		t.Fatalf("expected 0 suggestions, got %d", len(resp.Suggestions))
	}
	if resp.Notice == "" {
		t.Fatal("expected non-empty notice")
	}
}

// Test 6: Generic AI error returns 200 with notice (GenerateForTask wraps all provider errors into "all AI providers failed")
func TestPromptSuggest_GenericError_ReturnsNotice(t *testing.T) {
	svc, repo := newTestPromptService(t, "", errors.New("upstream timeout"))
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error msg: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200; msg = %q", status, msg)
	}
	if len(resp.Suggestions) != 0 {
		t.Fatalf("expected 0 suggestions, got %d", len(resp.Suggestions))
	}
	if resp.Notice == "" {
		t.Fatal("expected non-empty notice")
	}
}

// Test 7: Malformed AI JSON returns 500
func TestPromptSuggest_MalformedJSON_Returns500(t *testing.T) {
	svc, repo := newTestPromptService(t, "not json at all!!!", nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	_, status, _ := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if status != 500 {
		t.Fatalf("status = %d, want 500", status)
	}
}

// Test 8: JSON wrapped in markdown fences parses correctly
func TestPromptSuggest_FencedJSON_ParsesCorrectly(t *testing.T) {
	fenced := "```json\n" + validSuggestionJSON() + "\n```"
	svc, repo := newTestPromptService(t, fenced, nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(resp.Suggestions))
	}
}

// Test 9: JSON embedded in prose parses correctly
func TestPromptSuggest_ProseJSON_ParsesCorrectly(t *testing.T) {
	prose := "Here you go: " + validSuggestionJSON() + " hope that helps!"
	svc, repo := newTestPromptService(t, prose, nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(resp.Suggestions))
	}
}

// Test 10: Empty label is derived from first sentence of text
func TestPromptSuggest_EmptyLabel_DerivedFromText(t *testing.T) {
	aiJSON := `[{"text":"Write about ergonomic chairs for remote workers. Focus on lumbar support.","label":""}]`
	svc, repo := newTestPromptService(t, aiJSON, nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(resp.Suggestions))
	}
	if resp.Suggestions[0].Label == "" {
		t.Fatal("expected derived label, got empty string")
	}
}

// Test 11: Dedup vs existing prompt text
func TestPromptSuggest_DedupVsExisting_DropsDuplicate(t *testing.T) {
	existingText := "Write a Reddit post about finding relief from chronic back pain with our ergonomic chair."
	aiJSON := `[
		{"text":"` + existingText + `","label":"Duplicate"},
		{"text":"Draft a post about standing desks.","label":"Standing desk"}
	]`
	svc, repo := newTestPromptService(t, aiJSON, nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	repo.CreatePrompt(context.Background(), "p1", "product", "reddit", existingText)
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion after dedup, got %d", len(resp.Suggestions))
	}
	if resp.Suggestions[0].Label != "Standing desk" {
		t.Fatalf("expected 'Standing desk', got %q", resp.Suggestions[0].Label)
	}
}

// Test 12: Cross-suggestion dedup
func TestPromptSuggest_CrossSuggestionDedup_ReturnsUnique(t *testing.T) {
	aiJSON := `[
		{"text":"Write about ergonomic chairs.","label":"Chairs"},
		{"text":"Write about ergonomic chairs.","label":"Chairs Again"},
		{"text":"Write about standing desks.","label":"Desks"},
		{"text":"Write about monitor arms.","label":"Monitor arms"}
	]`
	svc, repo := newTestPromptService(t, aiJSON, nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 3 {
		t.Fatalf("expected 3 unique suggestions, got %d", len(resp.Suggestions))
	}
}

// Test 13: Max 3 cap
func TestPromptSuggest_Max3Cap(t *testing.T) {
	aiJSON := `[
		{"text":"Prompt one about chairs.","label":"One"},
		{"text":"Prompt two about desks.","label":"Two"},
		{"text":"Prompt three about monitors.","label":"Three"},
		{"text":"Prompt four about keyboards.","label":"Four"},
		{"text":"Prompt five about mice.","label":"Five"}
	]`
	svc, repo := newTestPromptService(t, aiJSON, nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 3 {
		t.Fatalf("expected 3 suggestions (capped), got %d", len(resp.Suggestions))
	}
}

// Test 14: Labels truncated to 60 chars
func TestPromptSuggest_TruncateLabels(t *testing.T) {
	longLabel := strings.Repeat("a", 200)
	aiJSON := `[{"text":"Some prompt text.","label":"` + longLabel + `"}]`
	svc, repo := newTestPromptService(t, aiJSON, nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(resp.Suggestions))
	}
	if len(resp.Suggestions[0].Label) != 60 {
		t.Fatalf("label length = %d, want 60", len(resp.Suggestions[0].Label))
	}
}

// Test 15: Empty/blank text entries dropped
func TestPromptSuggest_EmptyTextEntries_Dropped(t *testing.T) {
	aiJSON := `[
		{"text":"","label":"Empty text"},
		{"text":"  ","label":"Blank text"},
		{"text":"Valid prompt text here.","label":"Valid"}
	]`
	svc, repo := newTestPromptService(t, aiJSON, nil)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "Test", Mode: "research"})
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(resp.Suggestions))
	}
	if resp.Suggestions[0].Label != "Valid" {
		t.Fatalf("expected 'Valid', got %q", resp.Suggestions[0].Label)
	}
}

// Test 16: Existing prompt context included in user message
func TestPromptSuggest_ExistingPromptContextIncluded(t *testing.T) {
	rec := &recordingProvider{name: "copilot", result: validSuggestionJSON(), err: nil}
	svc, repo := newTestPromptServiceWithRecorder(t, rec)
	repo.CreateProject(context.Background(), domain.Project{ID: "p1", Name: "ErgoChair", Mode: "research"})
	repo.CreatePrompt(context.Background(), "p1", "product", "reddit", "Find people complaining about back pain at work.")
	resp, status, msg := svc.Suggest(context.Background(), "p1", SuggestPromptRequest{Platform: "reddit", Type: "product"})
	if msg != "" {
		t.Fatalf("unexpected error: %q", msg)
	}
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(resp.Suggestions) == 0 {
		t.Fatal("expected at least 1 suggestion")
	}
	if !strings.Contains(rec.lastUserMsg, "Find people complaining about back pain at work.") {
		t.Fatalf("user message does not contain existing prompt text. Got:\n%s", rec.lastUserMsg)
	}
	if !strings.Contains(rec.lastUserMsg, "ErgoChair") {
		t.Fatalf("user message does not contain project name. Got:\n%s", rec.lastUserMsg)
	}
}
