package services

import (
	"context"
	"encoding/json"
	"fmt"
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

const promptSuggestionSystemPrompt = `You are a social media outreach prompt engineer. You write system prompts that
guide a downstream AI to generate Reddit and Bluesky marketing content for a
specific project. The downstream AI will read your prompt and use it as
instructions for finding pain points, writing karma-building comments, or
drafting DMs.

The user message contains:
- project name and short description
- the target platform (reddit or bluesky) and prompt type (product, karma, dm)
- the project's existing prompts (for tone and vocabulary, but you may diverge)
- the project's existing search queries (for the topics the project cares about)

Generate exactly 3 distinct, complete prompt drafts. Each must:
- Be a self-contained instruction the downstream AI can act on directly
- Match the platform's tone and format (Reddit is more conversational, Bluesky
  is more terse and direct)
- Match the prompt type's purpose:
  - product: find pain points where the product fits naturally
  - karma: write high-upvote comments that build community credibility
  - dm: draft direct messages that lead with empathy and never pitch first
- Reference the project's topics and vocabulary so suggestions feel project-specific
- Be substantially different from each other (different angles or framings)

Return ONLY a valid JSON array, no markdown fencing, no commentary, no extra keys:
[{"text":"<full prompt body>","label":"<2-5 word label>"}, ...]`

var validPromptPlatforms = map[string]bool{"reddit": true, "bluesky": true}
var validPromptTypes = map[string]bool{"product": true, "karma": true, "dm": true}

// Suggest generates AI-powered prompt suggestions for a given platform+type slot.
func (s *PromptService) Suggest(ctx context.Context, projectID string, req SuggestPromptRequest) (SuggestPromptResponse, int, string) {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		code, msg := mapError(err)
		return SuggestPromptResponse{}, code, msg
	}

	platform := strings.TrimSpace(req.Platform)
	typ := strings.TrimSpace(req.Type)
	if !validPromptPlatforms[platform] {
		return SuggestPromptResponse{}, http.StatusBadRequest, "Invalid platform"
	}
	if !validPromptTypes[typ] {
		return SuggestPromptResponse{}, http.StatusBadRequest, "Invalid type"
	}

	prompts, err := s.repo.ListPrompts(ctx, projectID)
	if err != nil {
		return SuggestPromptResponse{}, http.StatusInternalServerError, "Internal server error"
	}

	queries, err := s.repo.ListAllQueries(ctx, projectID)
	if err != nil {
		return SuggestPromptResponse{}, http.StatusInternalServerError, "Internal server error"
	}

	// Build existing-key dedup set from current prompts for the same (platform, type).
	existingKeys := map[string]struct{}{}
	var existingPromptText string
	var allPromptSummaries []map[string]string
	for _, p := range prompts {
		normalized := strings.TrimSpace(p.PromptText)
		if normalized != "" {
			allPromptSummaries = append(allPromptSummaries, map[string]string{
				"platform": p.Platform,
				"type":     p.Type,
				"text":     normalized,
			})
		}
		if p.Platform == platform && p.Type == typ {
			existingPromptText = normalized
			if normalized != "" {
				existingKeys[strings.ToLower(normalized)] = struct{}{}
			}
		}
	}

	// Build user message.
	userMsg := fmt.Sprintf(`Project: "%s"`, project.Name)
	if project.Description != nil && *project.Description != "" {
		userMsg += "\nDescription: " + *project.Description
	}
	userMsg += fmt.Sprintf("\n\nTarget: platform=%s, type=%s", platform, typ)
	if existingPromptText != "" {
		userMsg += "\n\nCurrent prompt for this slot:\n" + existingPromptText
	}
	if len(allPromptSummaries) > 0 {
		enc, _ := json.Marshal(allPromptSummaries)
		userMsg += "\n\nAll existing prompts:\n" + string(enc)
	}
	if len(queries) > 0 {
		var querySummaries []map[string]string
		for _, q := range queries {
			querySummaries = append(querySummaries, map[string]string{
				"platform": q.Platform,
				"query":    q.QueryURL,
				"angle":    q.Angle,
			})
		}
		enc, _ := json.Marshal(querySummaries)
		userMsg += "\n\nExisting search queries:\n" + string(enc)
	}

	raw, _, err := s.ai.GenerateForTask(ctx, "prompt_suggestion", promptSuggestionSystemPrompt, userMsg)
	if err != nil {
		if strings.Contains(err.Error(), "all AI providers failed") {
			return SuggestPromptResponse{
				Suggestions: []PromptSuggestion{},
				Notice:      "AI suggestions are currently unavailable in this runtime because no provider is configured. Add a provider in host settings or set ANTHROPIC_API_KEY / GEMINI_API_KEY to enable suggestions.",
			}, http.StatusOK, ""
		}
		return SuggestPromptResponse{}, http.StatusInternalServerError, "Failed to generate prompt suggestions, try again"
	}

	cleaned := sanitizeAIJSON(raw)
	parsed, ok := parseSuggestionArray(cleaned)
	if !ok {
		return SuggestPromptResponse{}, http.StatusInternalServerError, "Failed to generate prompt suggestions, try again"
	}

	var out []PromptSuggestion
	seenTexts := map[string]struct{}{}
	for _, entry := range parsed {
		text := strings.TrimSpace(mapKeyAny(entry, "text", "Text"))
		label := strings.TrimSpace(mapKeyAny(entry, "label", "Label"))

		if text == "" {
			continue
		}
		if len(text) > 2000 {
			text = text[:2000]
		}

		if label == "" {
			// Derive label from first sentence of text.
			label = text
			if idx := strings.IndexAny(label, ".!?\n"); idx > 0 {
				label = label[:idx]
			}
			label = strings.TrimSpace(label)
		}
		if len(label) > 60 {
			label = label[:60]
		}
		if label == "" {
			continue
		}

		// Dedup against existing prompts.
		if _, exists := existingKeys[strings.ToLower(text)]; exists {
			continue
		}
		// Cross-suggestion dedup.
		textKey := strings.ToLower(text)
		if _, seen := seenTexts[textKey]; seen {
			continue
		}
		seenTexts[textKey] = struct{}{}

		out = append(out, PromptSuggestion{Text: text, Label: label})
		if len(out) >= 3 {
			break
		}
	}

	if out == nil {
		out = []PromptSuggestion{}
	}
	return SuggestPromptResponse{Suggestions: out}, http.StatusOK, ""
}
