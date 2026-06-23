package services

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

var httpURLRegexp = regexp.MustCompile(`(?i)^https?://`)

var validPlatforms = map[string]bool{
	"reddit":  true,
	"bluesky": true,
	"google":  true,
}

type QueryService struct {
	repo *repository.Repository
	ai   *ai.Service
}

func NewQueryService(repo *repository.Repository, ai *ai.Service) *QueryService {
	return &QueryService{repo: repo, ai: ai}
}

type QueryRequest struct {
	Platform string `json:"platform"`
	QueryURL string `json:"query_url"`
	Angle    string `json:"angle"`
	Enabled  *bool  `json:"enabled"`
}

func (s *QueryService) List(ctx context.Context, projectID string) ([]QueryWithQuality, int, string) {
	// Verify project exists
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		code, msg := mapError(err)
		return nil, code, msg
	}
	queries, err := s.repo.ListQueries(ctx, projectID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal server error"
	}

	socialRep, _, _ := s.repo.PickSocialReport(ctx, projectID, 0, 0)
	socialCtx := buildSocialQualityContext(socialRep)

	googleRep, _, _ := s.repo.PickGoogleReport(ctx, projectID, 0)
	var gq googleQualityData
	if googleRep != nil {
		rows, _ := s.repo.ListGoogleKeywordRows(ctx, projectID, googleRep.RunID)
		gq = buildGoogleQuality(rows, googleRep.ID)
	}

	result := make([]QueryWithQuality, 0, len(queries))
	for _, q := range queries {
		result = append(result, QueryWithQuality{
			ID:        q.ID,
			ProjectID: q.ProjectID,
			Platform:  q.Platform,
			QueryURL:  q.QueryURL,
			Angle:     q.Angle,
			Enabled:   q.Enabled,
			CreatedAt: q.CreatedAt,
			Quality:   buildQueryQuality(q.Platform, q.QueryURL, q.Angle, socialRep, socialCtx, gq),
		})
	}
	return result, http.StatusOK, ""
}

func (s *QueryService) Create(ctx context.Context, projectID string, body QueryRequest) (domain.Query, int, string) {
	// Verify project exists
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		code, msg := mapError(err)
		return domain.Query{}, code, msg
	}

	platform := strings.TrimSpace(body.Platform)
	queryURL := strings.TrimSpace(body.QueryURL)
	angle := strings.TrimSpace(body.Angle)

	if platform == "" || queryURL == "" || angle == "" {
		return domain.Query{}, http.StatusBadRequest, "platform, query_url, and angle are required"
	}
	if !validPlatforms[platform] {
		return domain.Query{}, http.StatusBadRequest, "platform must be reddit, bluesky, or google"
	}
	if platform == "google" && httpURLRegexp.MatchString(queryURL) {
		return domain.Query{}, http.StatusBadRequest, "google query_url must be a root keyword, not a URL"
	}

	enabledVal := true
	if body.Enabled != nil {
		enabledVal = *body.Enabled
	}
	q, err := s.repo.CreateQuery(ctx, projectID, platform, queryURL, angle, enabledVal)
	if err != nil {
		return domain.Query{}, http.StatusInternalServerError, "Internal server error"
	}
	return q, http.StatusCreated, ""
}

func (s *QueryService) Patch(ctx context.Context, projectID string, queryID int64, body map[string]any) (domain.Query, int, string) {
	platform, hasPlatform := stringField(body, "platform")
	if hasPlatform && !validPlatforms[platform] {
		return domain.Query{}, http.StatusBadRequest, "platform must be reddit, bluesky, or google"
	}
	if hasPlatform {
		body["platform"] = platform
	}
	queryURL, hasQueryURL := stringField(body, "query_url")
	if hasQueryURL {
		body["query_url"] = queryURL
	}

	if hasPlatform || hasQueryURL {
		existing, err := s.repo.GetQuery(ctx, projectID, queryID)
		if err != nil {
			code, msg := mapError(err)
			return domain.Query{}, code, msg
		}
		effectivePlatform := existing.Platform
		if hasPlatform {
			effectivePlatform = platform
		}
		effectiveQueryURL := existing.QueryURL
		if hasQueryURL {
			effectiveQueryURL = queryURL
		}
		if effectivePlatform == "google" && httpURLRegexp.MatchString(effectiveQueryURL) {
			return domain.Query{}, http.StatusBadRequest, "google query_url must be a root keyword, not a URL"
		}
	}

	q, err := s.repo.PatchQuery(ctx, projectID, queryID, body)
	if err != nil {
		code, msg := mapError(err)
		return domain.Query{}, code, msg
	}
	return q, http.StatusOK, ""
}

func stringField(body map[string]any, key string) (string, bool) {
	raw, ok := body[key]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	if !ok {
		return "", true
	}
	return strings.TrimSpace(value), true
}

func (s *QueryService) Delete(ctx context.Context, projectID string, queryID int64) (int, string) {
	err := s.repo.DeleteQuery(ctx, projectID, queryID)
	if err != nil {
		return mapError(err)
	}
	return http.StatusNoContent, ""
}
