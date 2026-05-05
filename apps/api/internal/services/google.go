package services

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// GoogleService handles business logic for google reports.
type GoogleService struct {
	repo *repository.Repository
}

// NewGoogleService creates a new GoogleService.
func NewGoogleService(repo *repository.Repository) *GoogleService {
	return &GoogleService{repo: repo}
}

// ListGoogleReports returns all google reports for a project.
func (s *GoogleService) ListGoogleReports(ctx context.Context, projectID string) ([]domain.GoogleReport, int, string) {
	reports, err := s.repo.ListGoogleReports(ctx, projectID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal server error"
	}
	return reports, http.StatusOK, ""
}

// LatestGoogleReport returns the most recent google report for a project.
func (s *GoogleService) LatestGoogleReport(ctx context.Context, projectID string) (domain.GoogleReport, int, string) {
	rep, err := s.repo.LatestGoogleReport(ctx, projectID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Google report not found"
		}
		return domain.GoogleReport{}, code, msg
	}
	return rep, http.StatusOK, ""
}

// GetGoogleReport returns a single google report by ID.
func (s *GoogleService) GetGoogleReport(ctx context.Context, projectID string, reportID int64) (domain.GoogleReport, int, string) {
	rep, err := s.repo.GetGoogleReport(ctx, projectID, reportID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Google report not found"
		}
		return domain.GoogleReport{}, code, msg
	}
	return rep, http.StatusOK, ""
}

// ListGoogleKeywordSummaries returns keyword summaries for a report's run.
func (s *GoogleService) ListGoogleKeywordSummaries(ctx context.Context, projectID string, runID int64) ([]repository.GoogleKeywordSummary, int, string) {
	summaries, err := s.repo.ListGoogleKeywordSummaries(ctx, projectID, runID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal server error"
	}
	return summaries, http.StatusOK, ""
}

// GoogleResultsResponse is the response shape for the results endpoint.
type GoogleResultsResponse struct {
	Mode    string                    `json:"mode"`
	Results []repository.GoogleResult `json:"results"`
}

// GetGoogleResults returns ranked and filtered results for a report.
func (s *GoogleService) GetGoogleResults(ctx context.Context, projectID string, reportID int64, mode string, limit int) (GoogleResultsResponse, int, string) {
	const defaultLimit = 20
	const maxLimit = 100
	validModes := map[string]bool{"seo": true, "messaging": true, "competitor": true, "outreach": true}

	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "seo"
	}
	if !validModes[mode] {
		return GoogleResultsResponse{}, http.StatusBadRequest, "mode must be one of seo, messaging, competitor, outreach"
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	rep, err := s.repo.GetGoogleReport(ctx, projectID, reportID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Google report not found"
		}
		return GoogleResultsResponse{}, code, msg
	}

	results, err := s.repo.ListGoogleResults(ctx, projectID, rep.RunID)
	if err != nil {
		return GoogleResultsResponse{}, http.StatusInternalServerError, "Internal server error"
	}

	ranked, rankErr := RankGoogleResults(results, mode, limit)
	if rankErr != nil {
		return GoogleResultsResponse{}, http.StatusBadRequest, rankErr.Error()
	}

	return GoogleResultsResponse{Mode: mode, Results: ranked}, http.StatusOK, ""
}

// RankGoogleResults filters and sorts results by mode, returning up to limit items.
func RankGoogleResults(results []repository.GoogleResult, mode string, limit int) ([]repository.GoogleResult, error) {
	validModes := map[string]bool{"seo": true, "messaging": true, "competitor": true, "outreach": true}
	if !validModes[mode] {
		return nil, &rankError{"mode must be one of seo, messaging, competitor, outreach"}
	}

	var filtered []repository.GoogleResult

	switch mode {
	case "seo":
		for _, r := range results {
			if r.RelevanceFit == "direct_fit" {
				filtered = append(filtered, r)
			}
		}
		sort.SliceStable(filtered, func(i, j int) bool {
			si := asFloat(filtered[i].RelevanceScore)
			sj := asFloat(filtered[j].RelevanceScore)
			if si != sj {
				return si > sj
			}
			return asRank(filtered[i].Rank) < asRank(filtered[j].Rank)
		})

	case "messaging":
		for _, r := range results {
			if r.RelevanceFit != "weak_fit" {
				filtered = append(filtered, r)
			}
		}
		sort.SliceStable(filtered, func(i, j int) bool {
			ci := asFloat(filtered[i].ConfidenceScore)
			cj := asFloat(filtered[j].ConfidenceScore)
			if ci != cj {
				return ci > cj
			}
			return asFloat(filtered[i].RelevanceScore) > asFloat(filtered[j].RelevanceScore)
		})

	case "competitor":
		for _, r := range results {
			if containsCompetitorWeakness(r.OpportunityTypes) || containsComparison(r.ActionRecommendation) {
				filtered = append(filtered, r)
			}
		}
		sort.SliceStable(filtered, func(i, j int) bool {
			return asFloat(filtered[i].RelevanceScore) > asFloat(filtered[j].RelevanceScore)
		})

	case "outreach":
		for _, r := range results {
			if r.OutreachCandidate == 1 {
				filtered = append(filtered, r)
			}
		}
		sort.SliceStable(filtered, func(i, j int) bool {
			return asFloat(filtered[i].RelevanceScore) > asFloat(filtered[j].RelevanceScore)
		})
	}

	if filtered == nil {
		filtered = []repository.GoogleResult{}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

type rankError struct{ msg string }

func (e *rankError) Error() string { return e.msg }

func asFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func asRank(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func containsCompetitorWeakness(raw []byte) bool {
	// raw is a JSON array; check if it contains "competitor_weakness"
	s := string(raw)
	return strings.Contains(s, `"competitor_weakness"`)
}

func containsComparison(actionRec string) bool {
	return strings.Contains(strings.ToLower(actionRec), "comparison")
}
