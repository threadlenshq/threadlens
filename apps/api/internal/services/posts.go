package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// AIService is the minimal interface PostService needs for draft generation.
// ai.Service satisfies this interface.
type AIService interface {
	GenerateForTask(ctx context.Context, taskID string, systemPrompt string, userMessage string) (string, string, error)
	StripMarkdown(text string) string
}

// RedditContextFetcher fetches Reddit post context for draft generation.
// pipeline.FetchRedditContext satisfies this as a function adapter; tests can
// provide a stub via RedditContextFetcherFunc.
type RedditContextFetcher interface {
	FetchRedditContext(ctx context.Context, postURL string) (RedditContext, error)
}

// RedditContext mirrors pipeline.RedditContext to avoid an import cycle between
// services and pipeline.
type RedditContext struct {
	FullBody    string
	TopComments []RedditTopComment
}

// RedditTopComment is a single comment entry returned by FetchRedditContext.
type RedditTopComment struct {
	Author string
	Body   string
	Score  int
}

// BlueskyReplier can post a reply to a Bluesky post.
// pipeline.PostBlueskyReply satisfies this as a function adapter.
type BlueskyReplier interface {
	PostBlueskyReply(ctx context.Context, handle, appPassword, text, parentURI, parentCID string) (json.RawMessage, error)
}

// PostStatuses matches Express POST_STATUSES from @scout/shared.
var PostStatuses = []string{"new", "drafted", "commented", "skipped", "reviewed", "starred", "excluded"}

type PostService struct {
	repo           *repository.Repository
	aiSvc          AIService
	redditFetcher  RedditContextFetcher
	blueskyReplier BlueskyReplier
}

func NewPostService(repo *repository.Repository) *PostService {
	return &PostService{repo: repo}
}

// NewPostServiceWithAI creates a PostService with AI generation support.
func NewPostServiceWithAI(repo *repository.Repository, ai AIService, reddit RedditContextFetcher) *PostService {
	return &PostService{repo: repo, aiSvc: ai, redditFetcher: reddit}
}

// NewPostServiceFull creates a PostService with all optional dependencies.
func NewPostServiceFull(repo *repository.Repository, ai AIService, reddit RedditContextFetcher, bsky BlueskyReplier) *PostService {
	return &PostService{repo: repo, aiSvc: ai, redditFetcher: reddit, blueskyReplier: bsky}
}

// ListPosts returns all posts (no pagination) with dm_targets attached.
func (s *PostService) ListPosts(ctx context.Context, projectID string, filters repository.PostFilters) ([]domain.Post, int, string) {
	posts, err := s.repo.ListPosts(ctx, projectID, filters)
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal server error"
	}
	return posts, http.StatusOK, ""
}

// ListPostsPage returns paginated posts.
func (s *PostService) ListPostsPage(ctx context.Context, projectID string, filters repository.PostFilters, page int, limit int) (domain.PagedPosts, int, string) {
	paged, err := s.repo.ListPostsPage(ctx, projectID, filters, page, limit)
	if err != nil {
		return domain.PagedPosts{}, http.StatusInternalServerError, "Internal server error"
	}
	return paged, http.StatusOK, ""
}

// GetPost returns a single post with dm_targets.
func (s *PostService) GetPost(ctx context.Context, projectID string, postID string) (domain.Post, int, string) {
	post, err := s.repo.GetPost(ctx, projectID, postID)
	if err != nil {
		code, msg := mapEntityError(err, "Post not found")
		return domain.Post{}, code, msg
	}
	return post, http.StatusOK, ""
}

// PatchPostRequest is the deserialized request body for patching a post.
type PatchPostBody struct {
	Status       *string `json:"status"`
	DraftComment *string `json:"draft_comment"`
}

// PatchPost updates a post and returns the updated post.
func (s *PostService) PatchPost(ctx context.Context, projectID string, postID string, body PatchPostBody) (domain.Post, int, string) {
	req := repository.PatchPostRequest{
		Status:       body.Status,
		DraftComment: body.DraftComment,
	}
	post, err := s.repo.PatchPost(ctx, projectID, postID, req)
	if err != nil {
		code, msg := mapEntityError(err, "Post not found")
		return domain.Post{}, code, msg
	}
	return post, http.StatusOK, ""
}

// BulkPatchBody is the deserialized body for bulk patch.
type BulkPatchBody struct {
	IDs    []string `json:"ids"`
	Status string   `json:"status"`
}

// BulkPatch validates and applies a bulk status update.
func (s *PostService) BulkPatch(ctx context.Context, projectID string, body BulkPatchBody) (int64, int, string) {
	if len(body.IDs) == 0 || body.Status == "" {
		return 0, http.StatusBadRequest, "ids (array) and status are required"
	}
	if !isValidStatus(body.Status) {
		return 0, http.StatusBadRequest, "Invalid status. Must be one of: " + strings.Join(PostStatuses, ", ")
	}
	updated, err := s.repo.BulkPatchPosts(ctx, projectID, body.IDs, body.Status)
	if err != nil {
		return 0, http.StatusInternalServerError, "Internal server error"
	}
	return updated, http.StatusOK, ""
}

// PatchDMBody is the deserialized body for patching a DM target.
type PatchDMBody struct {
	DraftDM  *string `json:"draft_dm"`
	DMStatus *string `json:"dm_status"`
}

// PatchDMTarget updates a DM target and returns it.
func (s *PostService) PatchDMTarget(ctx context.Context, projectID string, postID string, username string, body PatchDMBody) (domain.DMTarget, int, string) {
	// Verify post belongs to project first.
	if _, err := s.repo.GetPost(ctx, projectID, postID); err != nil {
		code, msg := mapEntityError(err, "Post not found")
		return domain.DMTarget{}, code, msg
	}

	target, err := s.repo.PatchDMTarget(ctx, projectID, postID, username, body.DraftDM, body.DMStatus)
	if err != nil {
		code, msg := mapEntityError(err, "DM target not found")
		return domain.DMTarget{}, code, msg
	}
	return target, http.StatusOK, ""
}

// ParseFilters reads filter query params from a raw query string map.
func ParseFilters(q map[string]string) repository.PostFilters {
	f := repository.PostFilters{
		Status:         q["status"],
		Platform:       q["platform"],
		EngagementType: q["engagement_type"],
		SignalType:     q["signal_type"],
	}
	if v, ok := q["min_score"]; ok && v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			f.MinScore = &parsed
		}
	}
	if v, ok := q["max_score"]; ok && v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			f.MaxScore = &parsed
		}
	}
	if strings.EqualFold(q["dm"], "true") {
		f.HasDMTargets = true
	}
	return f
}

func isValidStatus(status string) bool {
	for _, s := range PostStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// GenerateDraft generates an AI draft comment for a post and stores it.
// Mirrors Express POST /:pid/generate-draft.
func (s *PostService) GenerateDraft(ctx context.Context, projectID string, postID string) (domain.Post, int, string) {
	if s.aiSvc == nil {
		return domain.Post{}, http.StatusInternalServerError, "AI service not configured"
	}

	post, err := s.repo.GetPost(ctx, projectID, postID)
	if err != nil {
		code, msg := mapEntityError(err, "Post not found")
		return domain.Post{}, code, msg
	}

	promptType := "karma"
	if post.EngagementType == "product" {
		promptType = "product"
	}

	prompt, err := s.repo.GetPromptForPost(ctx, projectID, promptType, post.Platform)
	if err != nil {
		return domain.Post{}, http.StatusBadRequest, fmt.Sprintf("No %s prompt configured for %s", promptType, post.Platform)
	}

	var userMessage string
	if post.Platform == "reddit" {
		var fullBody string
		var topComments []RedditTopComment

		if s.redditFetcher != nil {
			rc, fetchErr := s.redditFetcher.FetchRedditContext(ctx, post.URL)
			if fetchErr == nil {
				fullBody = rc.FullBody
				topComments = rc.TopComments
			}
		}

		var commentLines []string
		for _, c := range topComments {
			commentLines = append(commentLines, fmt.Sprintf("- u/%s (score: %d): %s", c.Author, c.Score, c.Body))
		}

		body := fullBody
		if body == "" {
			body = post.Body
		}

		parts := []string{
			fmt.Sprintf("Title: %s", post.Title),
			fmt.Sprintf("Subreddit: %s", strOrEmpty(post.Subreddit)),
			fmt.Sprintf("Why this post: %s", post.Why),
			fmt.Sprintf("Body: %s", body),
		}
		if len(commentLines) > 0 {
			parts = append(parts, "Top comments:\n"+strings.Join(commentLines, "\n"))
		}
		userMessage = strings.Join(parts, "\n\n")
	} else {
		userMessage = strings.Join([]string{
			fmt.Sprintf("Author: %s", post.Author),
			fmt.Sprintf("Why this post: %s", post.Why),
			fmt.Sprintf("Body: %s", post.Body),
		}, "\n\n")
	}

	raw, modelID, genErr := s.aiSvc.GenerateForTask(ctx, "draft_generation", prompt.PromptText, userMessage)
	if genErr != nil {
		return domain.Post{}, http.StatusInternalServerError, genErr.Error()
	}
	draft := s.aiSvc.StripMarkdown(raw)

	drafted := "drafted"
	updated, err := s.repo.PatchPost(ctx, projectID, postID, repository.PatchPostRequest{
		DraftComment:  &draft,
		DraftProvider: &modelID,
		Status:        &drafted,
	})
	if err != nil {
		return domain.Post{}, http.StatusInternalServerError, "Internal server error"
	}
	return updated, http.StatusOK, ""
}

// GenerateDMDraft generates an AI DM draft for a dm_target and stores it.
// Mirrors Express POST /:pid/dm/:username/generate-draft.
func (s *PostService) GenerateDMDraft(ctx context.Context, projectID string, postID string, username string) (domain.DMTarget, int, string) {
	if s.aiSvc == nil {
		return domain.DMTarget{}, http.StatusInternalServerError, "AI service not configured"
	}

	post, err := s.repo.GetPost(ctx, projectID, postID)
	if err != nil {
		code, msg := mapEntityError(err, "Post not found")
		return domain.DMTarget{}, code, msg
	}

	target, err := s.repo.GetDMTarget(ctx, postID, username)
	if err != nil {
		code, msg := mapEntityError(err, "DM target not found")
		return domain.DMTarget{}, code, msg
	}

	prompt, err := s.repo.GetPromptForPost(ctx, projectID, "dm", "reddit")
	if err != nil {
		return domain.DMTarget{}, http.StatusBadRequest, "No DM prompt configured for reddit"
	}

	userMessage := strings.Join([]string{
		fmt.Sprintf("Username: u/%s", target.Username),
		fmt.Sprintf("Intent score: %s", formatFloat(target.IntentScore)),
		fmt.Sprintf("Signal: %s", target.Signal),
		fmt.Sprintf("Context: %s", target.Context),
		fmt.Sprintf("Approach: %s", target.Approach),
		fmt.Sprintf("Post title: %s", post.Title),
		fmt.Sprintf("Why this post: %s", post.Why),
	}, "\n\n")

	raw, modelID, genErr := s.aiSvc.GenerateForTask(ctx, "draft_generation", prompt.PromptText, userMessage)
	if genErr != nil {
		return domain.DMTarget{}, http.StatusInternalServerError, genErr.Error()
	}
	draft := s.aiSvc.StripMarkdown(raw)

	updated, err := s.repo.PatchDMTarget(ctx, projectID, postID, username, &draft, nil)
	if err != nil {
		return domain.DMTarget{}, http.StatusInternalServerError, "Internal server error"
	}
	// Also store draft_provider on dm_targets
	_, _ = s.repo.UpdateDMTargetProvider(ctx, target.ID, modelID)

	// Re-fetch to get the provider stored
	final, err := s.repo.GetDMTarget(ctx, postID, username)
	if err != nil {
		return updated, http.StatusOK, ""
	}
	return final, http.StatusOK, ""
}

// PostReplyBody is the deserialized request body for post-reply.
type PostReplyBody struct {
	Text string `json:"text"`
}

// PostReply posts a reply to a Bluesky post and marks the post as commented.
// Mirrors Express POST /:pid/post-reply.
func (s *PostService) PostReply(ctx context.Context, projectID string, postID string, body PostReplyBody) (domain.Post, int, string) {
	if body.Text == "" {
		return domain.Post{}, http.StatusBadRequest, "text is required"
	}

	post, err := s.repo.GetPost(ctx, projectID, postID)
	if err != nil {
		code, msg := mapEntityError(err, "Post not found")
		return domain.Post{}, code, msg
	}

	if post.BlueskyURI == nil || post.BlueskyCID == nil || *post.BlueskyURI == "" || *post.BlueskyCID == "" {
		return domain.Post{}, http.StatusBadRequest, "Post is missing Bluesky uri or cid"
	}

	handle := os.Getenv("BLUESKY_HANDLE")
	password := os.Getenv("BLUESKY_PASSWORD")
	if handle == "" || password == "" {
		return domain.Post{}, http.StatusInternalServerError, "BLUESKY_HANDLE or BLUESKY_PASSWORD not configured"
	}

	replier := s.blueskyReplier
	if replier == nil {
		return domain.Post{}, http.StatusInternalServerError, "Bluesky replier not configured"
	}

	if _, replyErr := replier.PostBlueskyReply(ctx, handle, password, body.Text, *post.BlueskyURI, *post.BlueskyCID); replyErr != nil {
		return domain.Post{}, http.StatusInternalServerError, replyErr.Error()
	}

	commented := "commented"
	updated, err := s.repo.PatchPost(ctx, projectID, postID, repository.PatchPostRequest{
		Status: &commented,
	})
	if err != nil {
		return domain.Post{}, http.StatusInternalServerError, "Internal server error"
	}
	return updated, http.StatusOK, ""
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// BlueskyReplierFunc is a function adapter that satisfies BlueskyReplier.
// Use this in app wiring to wrap pipeline.PostBlueskyReply, and in tests to inject stubs.
type BlueskyReplierFunc func(ctx context.Context, handle, appPassword, text, parentURI, parentCID string) (json.RawMessage, error)

func (f BlueskyReplierFunc) PostBlueskyReply(ctx context.Context, handle, appPassword, text, parentURI, parentCID string) (json.RawMessage, error) {
	return f(ctx, handle, appPassword, text, parentURI, parentCID)
}
