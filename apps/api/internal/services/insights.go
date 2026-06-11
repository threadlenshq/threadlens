package services

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "that": true, "this": true,
	"with": true, "are": true, "was": true, "they": true, "have": true,
	"from": true, "not": true, "but": true, "had": true, "has": true,
	"its": true, "into": true, "our": true, "can": true, "all": true,
	"their": true, "there": true, "when": true, "what": true, "been": true,
	"who": true, "more": true, "also": true, "just": true, "like": true,
	"about": true, "than": true, "out": true, "any": true, "she": true,
	"him": true, "her": true, "his": true, "one": true, "two": true,
	"how": true, "you": true, "your": true, "very": true, "even": true,
	"because": true, "really": true, "would": true, "could": true,
	"should": true, "want": true, "need": true, "will": true, "then": true,
	"them": true, "some": true, "were": true, "which": true,
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9\s]`)

// InsightsService provides cross-project aggregation insights.
type InsightsService struct {
	repo *repository.Repository
}

func NewInsightsService(repo *repository.Repository) *InsightsService {
	return &InsightsService{repo: repo}
}

// InsightsFilter holds optional filter parameters.
type InsightsFilter struct {
	ProjectID string
	Since     string // e.g. "7d", "30d", "1h"
	MinScore  *float64
}

// BuildInsights executes aggregate SQL and returns the insights map matching Express.
func (s *InsightsService) BuildInsights(ctx context.Context, f InsightsFilter) (map[string]any, error) {
	conditions := []string{}
	params := []any{}

	if f.ProjectID != "" {
		conditions = append(conditions, "project_id = ?")
		params = append(params, f.ProjectID)
	}

	if f.Since != "" {
		interval, ok := parseSince(f.Since)
		if !ok {
			return nil, errInvalidSince
		}
		conditions = append(conditions, "found_at >= datetime('now', ?)")
		params = append(params, interval)
	}

	if f.MinScore != nil {
		conditions = append(conditions, "final_score >= ?")
		params = append(params, *f.MinScore)
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	db := s.repo.DB

	// Total count
	totalSQL := "SELECT COUNT(*) FROM posts " + where
	var total int64
	if err := db.QueryRowContext(ctx, totalSQL, params...).Scan(&total); err != nil {
		return nil, err
	}

	// By angle - top 20
	byAngleSQL := `SELECT angle, COUNT(*) as count, AVG(final_score) as avg_score
		FROM posts ` + where + `
		GROUP BY angle
		ORDER BY count DESC
		LIMIT 20`
	byAngleRows, err := db.QueryContext(ctx, byAngleSQL, params...)
	if err != nil {
		return nil, err
	}
	defer byAngleRows.Close()
	byAngle := []map[string]any{}
	for byAngleRows.Next() {
		var angle *string
		var count int64
		var avgScore *float64
		if err := byAngleRows.Scan(&angle, &count, &avgScore); err != nil {
			return nil, err
		}
		entry := map[string]any{
			"angle":     angle,
			"count":     count,
			"avg_score": avgScore,
		}
		byAngle = append(byAngle, entry)
	}
	if err := byAngleRows.Err(); err != nil {
		return nil, err
	}

	// By type
	byTypeSQL := `SELECT engagement_type, COUNT(*) as count
		FROM posts ` + where + `
		GROUP BY engagement_type`
	byTypeRows, err := db.QueryContext(ctx, byTypeSQL, params...)
	if err != nil {
		return nil, err
	}
	defer byTypeRows.Close()
	byType := []map[string]any{}
	for byTypeRows.Next() {
		var engType *string
		var count int64
		if err := byTypeRows.Scan(&engType, &count); err != nil {
			return nil, err
		}
		byType = append(byType, map[string]any{
			"engagement_type": engType,
			"count":           count,
		})
	}
	if err := byTypeRows.Err(); err != nil {
		return nil, err
	}

	// Score distribution
	distSQL := `SELECT
		SUM(CASE WHEN final_score >= 8 AND final_score <= 10 THEN 1 ELSE 0 END) as high,
		SUM(CASE WHEN final_score >= 5 AND final_score <= 7 THEN 1 ELSE 0 END) as medium,
		SUM(CASE WHEN final_score >= 2 AND final_score <= 4 THEN 1 ELSE 0 END) as low
		FROM posts ` + where
	var high, medium, low *int64
	if err := db.QueryRowContext(ctx, distSQL, params...).Scan(&high, &medium, &low); err != nil {
		return nil, err
	}
	scoreDistribution := map[string]any{
		"high":   high,
		"medium": medium,
		"low":    low,
	}

	// Top posts - top 10 by final_score
	topPostsSQL := `SELECT id, title, body, platform, final_score, angle, why, author, url, found_at
		FROM posts ` + where + `
		ORDER BY final_score DESC
		LIMIT 10`
	topPostRows, err := db.QueryContext(ctx, topPostsSQL, params...)
	if err != nil {
		return nil, err
	}
	defer topPostRows.Close()
	topPosts := []map[string]any{}
	var whyTexts []string
	for topPostRows.Next() {
		var id, title, body, platform, author, url, foundAt string
		var finalScore float64
		var angle, why *string
		if err := topPostRows.Scan(&id, &title, &body, &platform, &finalScore, &angle, &why, &author, &url, &foundAt); err != nil {
			return nil, err
		}
		topPosts = append(topPosts, map[string]any{
			"id":          id,
			"title":       title,
			"body":        body,
			"platform":    platform,
			"final_score": finalScore,
			"angle":       angle,
			"why":         why,
			"author":      author,
			"url":         url,
			"found_at":    foundAt,
		})
		if why != nil {
			whyTexts = append(whyTexts, *why)
		}
	}
	if err := topPostRows.Err(); err != nil {
		return nil, err
	}

	// Top keywords: fetch all "why" fields for the filter, compute in Go
	whySQL := "SELECT why FROM posts " + where
	whyRows, err := db.QueryContext(ctx, whySQL, params...)
	if err != nil {
		return nil, err
	}
	defer whyRows.Close()
	allWhy := []string{}
	for whyRows.Next() {
		var why *string
		if err := whyRows.Scan(&why); err != nil {
			return nil, err
		}
		if why != nil {
			allWhy = append(allWhy, *why)
		}
	}
	if err := whyRows.Err(); err != nil {
		return nil, err
	}
	topKeywords := computeTopKeywords(allWhy, 20)

	return map[string]any{
		"total":              total,
		"by_angle":           byAngle,
		"by_type":            byType,
		"score_distribution": scoreDistribution,
		"top_posts":          topPosts,
		"top_keywords":       topKeywords,
	}, nil
}

// errInvalidSince is returned for bad since values.
var errInvalidSince = &insightsError{"Invalid since format. Use e.g. \"7d\", \"30d\", \"1h\""}

type insightsError struct{ msg string }

func (e *insightsError) Error() string { return e.msg }

// IsInvalidSince returns true if the error is an invalid since error.
func IsInvalidSince(err error) bool {
	_, ok := err.(*insightsError)
	return ok
}

// matches e.g. "7d", "30d", "1h", "15m"
var sinceRe = regexp.MustCompile(`^(\d+)([dhm])$`)
var sinceUnits = map[string]string{"d": "days", "h": "hours", "m": "minutes"}

func parseSince(since string) (string, bool) {
	m := sinceRe.FindStringSubmatch(since)
	if m == nil {
		return "", false
	}
	return "-" + m[1] + " " + sinceUnits[m[2]], true
}

func computeTopKeywords(whyList []string, topN int) []map[string]any {
	freq := map[string]int{}
	for _, why := range whyList {
		lower := strings.ToLower(why)
		clean := nonAlphaNum.ReplaceAllString(lower, " ")
		words := strings.Fields(clean)
		for _, word := range words {
			if len(word) < 3 {
				continue
			}
			if stopWords[word] {
				continue
			}
			freq[word]++
		}
	}
	type kv struct {
		word  string
		count int
	}
	var sorted []kv
	for w, c := range freq {
		sorted = append(sorted, kv{w, c})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].count != sorted[j].count {
			return sorted[i].count > sorted[j].count
		}
		return sorted[i].word < sorted[j].word
	})
	if len(sorted) > topN {
		sorted = sorted[:topN]
	}
	result := make([]map[string]any, len(sorted))
	for i, kv := range sorted {
		result[i] = map[string]any{"word": kv.word, "count": kv.count}
	}
	return result
}
