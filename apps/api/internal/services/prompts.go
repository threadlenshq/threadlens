package services

import (
	"context"
	"net/http"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

type PromptService struct {
	repo *repository.Repository
	ai   *ai.Service
}

func NewPromptService(repo *repository.Repository, aiSvc *ai.Service) *PromptService {
	return &PromptService{repo: repo, ai: aiSvc}
}

type PromptRequest struct {
	Type       string `json:"type"`
	Platform   string `json:"platform"`
	PromptText string `json:"prompt_text"`
}

// SuggestPromptRequest is the request body for POST /prompts/suggest.
type SuggestPromptRequest struct {
	Platform string `json:"platform"`
	Type     string `json:"type"`
}

// PromptSuggestion is one AI-generated prompt suggestion.
type PromptSuggestion struct {
	Text  string `json:"text"`
	Label string `json:"label"`
}

// SuggestPromptResponse is the response for POST /prompts/suggest.
type SuggestPromptResponse struct {
	Suggestions []PromptSuggestion `json:"suggestions"`
	Notice      string             `json:"notice,omitempty"`
}

func (s *PromptService) List(ctx context.Context, projectID string) ([]domain.Prompt, int, string) {
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		code, msg := mapError(err)
		return nil, code, msg
	}
	prompts, err := s.repo.ListPrompts(ctx, projectID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal server error"
	}
	return prompts, http.StatusOK, ""
}

func (s *PromptService) Create(ctx context.Context, projectID string, body PromptRequest) (domain.Prompt, int, string) {
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		code, msg := mapError(err)
		return domain.Prompt{}, code, msg
	}

	typ := strings.TrimSpace(body.Type)
	platform := strings.TrimSpace(body.Platform)

	if typ == "" || platform == "" {
		return domain.Prompt{}, http.StatusBadRequest, "type and platform are required"
	}

	p, err := s.repo.CreatePrompt(ctx, projectID, typ, platform, body.PromptText)
	if err != nil {
		return domain.Prompt{}, http.StatusInternalServerError, "Internal server error"
	}
	return p, http.StatusCreated, ""
}

func (s *PromptService) Patch(ctx context.Context, projectID string, promptID int64, body map[string]any) (domain.Prompt, int, string) {
	p, err := s.repo.PatchPrompt(ctx, projectID, promptID, body)
	if err != nil {
		code, msg := mapError(err)
		return domain.Prompt{}, code, msg
	}
	return p, http.StatusOK, ""
}

func (s *PromptService) Delete(ctx context.Context, projectID string, promptID int64) (int, string) {
	err := s.repo.DeletePrompt(ctx, projectID, promptID)
	if err != nil {
		return mapError(err)
	}
	return http.StatusNoContent, ""
}

// Suggest generates AI-powered prompt suggestions for a given platform+type slot.
// Stub: will be fully implemented in Task 5.
func (s *PromptService) Suggest(ctx context.Context, projectID string, req SuggestPromptRequest) (SuggestPromptResponse, int, string) {
	return SuggestPromptResponse{}, http.StatusNotImplemented, "Not implemented"
}
