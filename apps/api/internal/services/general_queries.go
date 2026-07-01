package services

import (
	"context"
	"net/http"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

type GeneralQueryService struct {
	repo *repository.Repository
}

func NewGeneralQueryService(repo *repository.Repository) *GeneralQueryService {
	return &GeneralQueryService{repo: repo}
}

type GeneralQueryRequest struct {
	QueryText string   `json:"query_text"`
	Angle     string   `json:"angle"`
	Platforms []string `json:"platforms"`
	Enabled   *bool    `json:"enabled"`
}

func (s *GeneralQueryService) List(ctx context.Context, projectID string) ([]domain.GeneralQuery, int, string) {
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		code, msg := mapError(err)
		return nil, code, msg
	}
	gqs, err := s.repo.ListGeneralQueries(ctx, projectID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal server error"
	}
	return gqs, http.StatusOK, ""
}

// validateAndAdapt checks the request and returns the per-platform (platform, queryURL)
// pairs to materialize, or an HTTP error.
func (s *GeneralQueryService) validateAndAdapt(body GeneralQueryRequest) ([][2]string, int, string) {
	text := strings.TrimSpace(body.QueryText)
	angle := strings.TrimSpace(body.Angle)
	if text == "" || angle == "" {
		return nil, http.StatusBadRequest, "query_text and angle are required"
	}
	if len(body.Platforms) == 0 {
		return nil, http.StatusBadRequest, "at least one platform is required"
	}
	pairs := make([][2]string, 0, len(body.Platforms))
	for _, p := range body.Platforms {
		if !validPlatforms[p] {
			return nil, http.StatusBadRequest, "platform must be reddit, bluesky, google, or hackernews"
		}
		queryURL, err := AdaptQueryURL(p, text)
		if err != nil {
			return nil, http.StatusBadRequest, err.Error()
		}
		pairs = append(pairs, [2]string{p, queryURL})
	}
	return pairs, http.StatusOK, ""
}

// checkCollisions returns a 409 message if any (platform, normalized query_url) pair
// already exists among the project's queries, excluding rows owned by excludeGQID.
func (s *GeneralQueryService) checkCollisions(ctx context.Context, projectID string, pairs [][2]string, excludeGQID int64) (int, string) {
	existing, err := s.repo.ListAllQueries(ctx, projectID)
	if err != nil {
		return http.StatusInternalServerError, "Internal server error"
	}
	seen := map[string]bool{}
	for _, q := range existing {
		if q.GeneralQueryID != nil && *q.GeneralQueryID == excludeGQID {
			continue
		}
		seen[q.Platform+"|"+NormalizeQueryURL(q.Platform, q.QueryURL)] = true
	}
	for _, pair := range pairs {
		key := pair[0] + "|" + NormalizeQueryURL(pair[0], pair[1])
		if seen[key] {
			return http.StatusConflict, "A " + pair[0] + " query for this already exists; it's already covered"
		}
	}
	return http.StatusOK, ""
}

func (s *GeneralQueryService) Create(ctx context.Context, projectID string, body GeneralQueryRequest) (domain.GeneralQuery, int, string) {
	if _, err := s.repo.GetProject(ctx, projectID); err != nil {
		code, msg := mapError(err)
		return domain.GeneralQuery{}, code, msg
	}
	pairs, code, msg := s.validateAndAdapt(body)
	if msg != "" {
		return domain.GeneralQuery{}, code, msg
	}
	if code, msg := s.checkCollisions(ctx, projectID, pairs, 0); msg != "" {
		return domain.GeneralQuery{}, code, msg
	}

	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	angle := strings.TrimSpace(body.Angle)
	gq, err := s.repo.CreateGeneralQuery(ctx, projectID, strings.TrimSpace(body.QueryText), angle, body.Platforms, enabled)
	if err != nil {
		return domain.GeneralQuery{}, http.StatusInternalServerError, "Internal server error"
	}
	for _, pair := range pairs {
		if _, err := s.repo.InsertMaterializedQuery(ctx, projectID, gq.ID, pair[0], pair[1], angle, enabled); err != nil {
			return domain.GeneralQuery{}, http.StatusInternalServerError, "Internal server error"
		}
	}
	return gq, http.StatusCreated, ""
}

func (s *GeneralQueryService) Update(ctx context.Context, projectID string, id int64, body GeneralQueryRequest) (domain.GeneralQuery, int, string) {
	if _, err := s.repo.GetGeneralQuery(ctx, projectID, id); err != nil {
		code, msg := mapError(err)
		return domain.GeneralQuery{}, code, msg
	}
	pairs, code, msg := s.validateAndAdapt(body)
	if msg != "" {
		return domain.GeneralQuery{}, code, msg
	}
	// Check collisions before deleting - excludeGQID skips this query's own rows,
	// so no false self-collision occurs and no data is lost on a rejected update.
	if code, msg := s.checkCollisions(ctx, projectID, pairs, id); msg != "" {
		return domain.GeneralQuery{}, code, msg
	}
	if err := s.repo.DeleteMaterializedQueries(ctx, id); err != nil {
		return domain.GeneralQuery{}, http.StatusInternalServerError, "Internal server error"
	}

	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	angle := strings.TrimSpace(body.Angle)
	gq, err := s.repo.UpdateGeneralQueryMeta(ctx, projectID, id, strings.TrimSpace(body.QueryText), angle, body.Platforms, enabled)
	if err != nil {
		return domain.GeneralQuery{}, http.StatusInternalServerError, "Internal server error"
	}
	for _, pair := range pairs {
		if _, err := s.repo.InsertMaterializedQuery(ctx, projectID, id, pair[0], pair[1], angle, enabled); err != nil {
			return domain.GeneralQuery{}, http.StatusInternalServerError, "Internal server error"
		}
	}
	return gq, http.StatusOK, ""
}

func (s *GeneralQueryService) Delete(ctx context.Context, projectID string, id int64) (int, string) {
	if err := s.repo.DeleteGeneralQuery(ctx, projectID, id); err != nil {
		return mapError(err)
	}
	return http.StatusNoContent, ""
}
