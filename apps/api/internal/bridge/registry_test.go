package bridge

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRuntime struct {
	id       string
	status   RuntimeStatus
	text     string
	err      error
	detects  int
	requests []GenerateRequest
}

func (r *fakeRuntime) ID() string { return r.id }

func (r *fakeRuntime) Detect(context.Context) RuntimeStatus {
	r.detects++
	return r.status
}

func (r *fakeRuntime) Generate(_ context.Context, req GenerateRequest) (string, error) {
	r.requests = append(r.requests, req)
	if r.err != nil {
		return "", r.err
	}
	return r.text, nil
}

func TestRegistryAvailableRuntimeIDsOnlyIncludesAvailable(t *testing.T) {
	copilot := &fakeRuntime{id: "copilot", status: RuntimeStatus{ID: "copilot", Available: true}}
	claude := &fakeRuntime{id: "claude-cli", status: RuntimeStatus{ID: "claude-cli", Available: false, Message: "not authenticated"}}
	reg := NewRegistry(1*time.Minute, copilot, claude)

	ids := reg.AvailableRuntimeIDs(context.Background())
	if len(ids) != 1 || ids[0] != "copilot" {
		t.Fatalf("expected only copilot, got %#v", ids)
	}
	if copilot.detects != 1 || claude.detects != 1 {
		t.Fatalf("expected one detect per runtime, got copilot=%d claude=%d", copilot.detects, claude.detects)
	}
	_ = reg.AvailableRuntimeIDs(context.Background())
	if copilot.detects != 1 || claude.detects != 1 {
		t.Fatalf("expected cached detects, got copilot=%d claude=%d", copilot.detects, claude.detects)
	}
}

func TestRegistryGenerateDispatchesByProvider(t *testing.T) {
	copilot := &fakeRuntime{id: "copilot", status: RuntimeStatus{ID: "copilot", Available: true}, text: "bridge text"}
	reg := NewRegistry(1*time.Minute, copilot)

	text, err := reg.Generate(context.Background(), GenerateRequest{Provider: "copilot", Model: "gpt-5-mini", UserMessage: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "bridge text" {
		t.Fatalf("expected bridge text, got %q", text)
	}
	if len(copilot.requests) != 1 || copilot.requests[0].Model != "gpt-5-mini" {
		t.Fatalf("request not passed through: %#v", copilot.requests)
	}
}

func TestRegistryGenerateRejectsUnknownUnavailableAndEmptyText(t *testing.T) {
	unavailable := &fakeRuntime{id: "copilot", status: RuntimeStatus{ID: "copilot", Available: false}}
	reg := NewRegistry(0, unavailable)
	if _, err := reg.Generate(context.Background(), GenerateRequest{Provider: "missing"}); err == nil {
		t.Fatal("expected unknown runtime error")
	}
	if _, err := reg.Generate(context.Background(), GenerateRequest{Provider: "copilot"}); err == nil {
		t.Fatal("expected unavailable runtime error")
	}

	empty := &fakeRuntime{id: "claude-cli", status: RuntimeStatus{ID: "claude-cli", Available: true}, text: "   "}
	reg = NewRegistry(0, empty)
	if _, err := reg.Generate(context.Background(), GenerateRequest{Provider: "claude-cli"}); err == nil {
		t.Fatal("expected empty text error")
	}

	broken := &fakeRuntime{id: "copilot", status: RuntimeStatus{ID: "copilot", Available: true}, err: errors.New("boom")}
	reg = NewRegistry(0, broken)
	if _, err := reg.Generate(context.Background(), GenerateRequest{Provider: "copilot"}); err == nil {
		t.Fatal("expected runtime error")
	}
}
