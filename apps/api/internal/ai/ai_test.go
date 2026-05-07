package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/usage"
)

// ---------------------------------------------------------------------------
// fakeProvider – controllable test double
// ---------------------------------------------------------------------------

type fakeProvider struct {
	name      string
	available bool
	result    string
	err       error
	calls     int
}

func (f *fakeProvider) Name() string    { return f.name }
func (f *fakeProvider) Available() bool { return f.available }
func (f *fakeProvider) Generate(_ context.Context, _ string, _ string, _ string, _ time.Duration) (string, error) {
	f.calls++
	return f.result, f.err
}

// newTestService builds a Service with the given fake providers and a NoopMeter.
func newTestService(providers []Provider) *Service {
	return &Service{providers: providers, meter: usage.NoopMeter{}}
}

// ---------------------------------------------------------------------------
// GenerateAuto tests
// ---------------------------------------------------------------------------

func TestGenerateAuto_UsesCachedProvider(t *testing.T) {
	p := &fakeProvider{name: "copilot", available: true, result: "hello"}
	svc := newTestService([]Provider{p})
	svc.cachedProvider = p // prime the cache

	got, err := svc.GenerateAuto(context.Background(), "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
	if p.calls != 1 {
		t.Errorf("expected 1 call to cached provider, got %d", p.calls)
	}
}

func TestGenerateAuto_ClearsCache_WhenCachedProviderFails(t *testing.T) {
	bad := &fakeProvider{name: "copilot", available: true, err: errors.New("boom")}
	good := &fakeProvider{name: "claude", available: true, result: "ok"}
	svc := newTestService([]Provider{bad, good})
	svc.cachedProvider = bad // prime with provider that will fail

	got, err := svc.GenerateAuto(context.Background(), "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" {
		t.Errorf("expected 'ok', got %q", got)
	}
	// Cache should now point to 'claude' (fallback that worked)
	if svc.cachedProvider == nil || svc.cachedProvider.Name() != "claude" {
		t.Errorf("expected cache to be 'claude', got %v", svc.cachedProvider)
	}
}

func TestGenerateAuto_FallbackOrder(t *testing.T) {
	// All unavailable except sdk:haiku
	copilot := &fakeProvider{name: "copilot", available: false}
	claude := &fakeProvider{name: "claude", available: false}
	sdk := &fakeProvider{name: "sdk", available: true, result: "sdk-result"}
	gemini := &fakeProvider{name: "gemini", available: true, result: "gemini-result"}

	svc := newTestService([]Provider{copilot, claude, sdk, gemini})

	got, err := svc.GenerateAuto(context.Background(), "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sdk-result" {
		t.Errorf("expected 'sdk-result', got %q", got)
	}
	// sdk should now be cached
	if svc.cachedProvider == nil || svc.cachedProvider.Name() != "sdk" {
		t.Errorf("expected cache='sdk', got %v", svc.cachedProvider)
	}
}

func TestGenerateAuto_FallbackOrderComplete(t *testing.T) {
	// Verify the exact order the fallbacks are tried by recording call sequence.
	var order []string
	makeProvider := func(name string, available bool) *fakeProvider {
		return &fakeProvider{
			name:      name,
			available: available,
			err:       errors.New("fail"),
		}
	}
	_ = order

	copilot := makeProvider("copilot", true)
	claude := makeProvider("claude", true)
	sdk := makeProvider("sdk", true)
	gemini := &fakeProvider{name: "gemini", available: true, result: "last"}

	svc := newTestService([]Provider{copilot, claude, sdk, gemini})
	got, err := svc.GenerateAuto(context.Background(), "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "last" {
		t.Errorf("expected 'last' from gemini, got %q", got)
	}
	// Each of copilot, claude, sdk should have been tried exactly once.
	for _, p := range []*fakeProvider{copilot, claude, sdk} {
		if p.calls != 1 {
			t.Errorf("provider %q: expected 1 call, got %d", p.name, p.calls)
		}
	}
}

// ---------------------------------------------------------------------------
// GenerateForTask tests
// ---------------------------------------------------------------------------

func TestGenerateForTask_TriesPrimaryFirst(t *testing.T) {
	// post_scoring default is copilot:gpt-5-mini
	copilot := &fakeProvider{name: "copilot", available: true, result: "scored"}
	svc := newTestService([]Provider{copilot})

	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "scored" {
		t.Errorf("expected 'scored', got %q", result)
	}
	if usedID != "copilot:gpt-5-mini" {
		t.Errorf("expected usedID='copilot:gpt-5-mini', got %q", usedID)
	}
}

func TestGenerateForTask_FallsBackWhenPrimaryFails(t *testing.T) {
	copilot := &fakeProvider{name: "copilot", available: true, err: errors.New("copilot down")}
	claude := &fakeProvider{name: "claude", available: true, result: "fallback-result"}

	svc := newTestService([]Provider{copilot, claude})
	result, usedID, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "fallback-result" {
		t.Errorf("expected 'fallback-result', got %q", result)
	}
	if usedID != "claude-cli:haiku" {
		t.Errorf("expected usedID='claude-cli:haiku', got %q", usedID)
	}
}

func TestGenerateForTask_SkipsUnavailableFallbacks(t *testing.T) {
	copilot := &fakeProvider{name: "copilot", available: true, err: errors.New("down")}
	claude := &fakeProvider{name: "claude", available: false} // unavailable
	sdk := &fakeProvider{name: "sdk", available: false}       // unavailable
	gemini := &fakeProvider{name: "gemini", available: true, result: "gemini-ok"}

	svc := newTestService([]Provider{copilot, claude, sdk, gemini})
	result, _, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "gemini-ok" {
		t.Errorf("expected 'gemini-ok', got %q", result)
	}
}

func TestGenerateForTask_UnknownTask(t *testing.T) {
	svc := newTestService(nil)
	_, _, err := svc.GenerateForTask(context.Background(), "nonexistent_task", "sys", "msg")
	if err == nil {
		t.Fatal("expected error for unknown task")
	}
}

// ---------------------------------------------------------------------------
// StripMarkdown tests
// ---------------------------------------------------------------------------

func TestStripMarkdown_Bold(t *testing.T) {
	got := StripMarkdown("**hello**")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_BoldUnderscore(t *testing.T) {
	got := StripMarkdown("__hello__")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_Italic(t *testing.T) {
	got := StripMarkdown("*hello*")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_ItalicUnderscore(t *testing.T) {
	got := StripMarkdown("_hello_")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_Code(t *testing.T) {
	got := StripMarkdown("`hello`")
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestStripMarkdown_EmDash(t *testing.T) {
	got := StripMarkdown("foo\u2014bar")
	if got != "foo, bar" {
		t.Errorf("expected 'foo, bar', got %q", got)
	}
}

func TestStripMarkdown_Mixed(t *testing.T) {
	input := "**Bold** and _italic_ and `code` \u2014 done"
	// Express replaces \u2014 with ", " (comma+space), so surrounding spaces are preserved.
	want := "Bold and italic and code ,  done"
	got := StripMarkdown(input)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

// ---------------------------------------------------------------------------
// GenerateForTask usage metering tests
// ---------------------------------------------------------------------------

func TestGenerateForTaskRecordsUsageEvent(t *testing.T) {
	meter := usage.NewMemoryMeter()
	copilot := &fakeProvider{name: "copilot", available: true, result: "scored"}
	svc := &Service{providers: []Provider{copilot}, meter: meter}

	_, _, err := svc.GenerateForTask(context.Background(), "post_scoring", "sys", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := meter.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 usage event, got %d", len(events))
	}
	ev := events[0]
	if ev.TaskID != "post_scoring" {
		t.Errorf("expected TaskID='post_scoring', got %q", ev.TaskID)
	}
	if ev.ModelID == "" {
		t.Errorf("expected non-empty ModelID")
	}
	if ev.Operation != "ai.generate_for_task" {
		t.Errorf("expected Operation='ai.generate_for_task', got %q", ev.Operation)
	}
	if !ev.Success {
		t.Errorf("expected Success=true")
	}
}
