package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// ────────────────────────────────────────────────────────────────
// Domain types
// ────────────────────────────────────────────────────────────────

// SuggestRequest is the request body for POST /queries/suggest.
type SuggestRequest struct {
	Refinement string `json:"refinement"`
}

// SuggestResponse is the response for POST /queries/suggest.
type SuggestResponse struct {
	Suggestions []SuggestedQuery `json:"suggestions"`
	Notice      string           `json:"notice,omitempty"`
}

// SuggestedQuery is one AI-generated suggestion.
type SuggestedQuery struct {
	Platform string `json:"platform"`
	QueryURL string `json:"query_url"`
	Angle    string `json:"angle"`
}

// RefineRequest is the request body for POST /queries/refine.
type RefineRequest struct {
	Refinement     string `json:"refinement"`
	SocialReportID int64  `json:"social_report_id"`
	GoogleReportID int64  `json:"google_report_id"`
}

// RefineResponse is the response for POST /queries/refine.
type RefineResponse struct {
	Summary         string                 `json:"summary"`
	Context         RefineContext          `json:"context"`
	Recommendations []RefineRecommendation `json:"recommendations"`
}

// RefineContext carries metadata about what reports were used.
type RefineContext struct {
	QueryCount        int              `json:"query_count"`
	EnabledQueryCount int              `json:"enabled_query_count"`
	SocialReport      *RefineReportRef `json:"social_report"`
	GoogleReport      *RefineReportRef `json:"google_report"`
}

// RefineReportRef is a reference to a report used in refinement.
type RefineReportRef struct {
	ID     int64  `json:"id"`
	Source string `json:"source"`
}

// RefineRecommendation is a single recommendation from refine.
type RefineRecommendation struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"`
	Reason          string          `json:"reason"`
	Sources         []string        `json:"sources"`
	Query           json.RawMessage `json:"query"`
	ReplacesQueryID *int64          `json:"replaces_query_id,omitempty"`
}

// ────────────────────────────────────────────────────────────────
// Text/scoring helpers (ported from routes/queries.js)
// ────────────────────────────────────────────────────────────────

var httpPrefixRe = regexp.MustCompile(`(?i)^https?://`)
var nonAlphaNumRe = regexp.MustCompile(`[^a-z0-9]+`)
var httpURLTokenRe = regexp.MustCompile(`https?://\S+`)

func parseJSON(value string, fallback any) any {
	var out any
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return fallback
	}
	return out
}

func normalizeQueryValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func buildQueryKey(platform, queryURL string) string {
	return strings.ToLower(strings.TrimSpace(platform)) + "::" + normalizeQueryValue(queryURL)
}

func makeRecommendationID(rtype string, platform string, queryURL string, queryID *int64) string {
	if rtype == "disable" && queryID != nil {
		return fmt.Sprintf("disable:%d", *queryID)
	}
	return rtype + ":" + buildQueryKey(platform, queryURL)
}

func trimText(value string, maxLength int) string {
	text := strings.TrimSpace(value)
	if len([]rune(text)) > maxLength {
		runes := []rune(text)
		return string(runes[:maxLength-3]) + "..."
	}
	return text
}

func normalizeText(value string) string {
	s := strings.ToLower(value)
	s = httpURLTokenRe.ReplaceAllString(s, " ")
	s = nonAlphaNumRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func tokenize(value string) []string {
	parts := strings.Fields(normalizeText(value))
	var out []string
	for _, p := range parts {
		if len(p) >= 3 {
			out = append(out, p)
		}
	}
	return out
}

func decodeTokenSource(value string) string {
	return strings.ReplaceAll(value, "+", " ")
}

func tokenizeQueryURL(value string) []string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return tokenize(raw)
	}
	if parsed.Scheme == "" {
		return tokenize(raw)
	}

	tokenSet := map[string]struct{}{}
	qValue := parsed.Query().Get("q")
	if qValue != "" {
		for _, t := range tokenize(decodeTokenSource(qValue)) {
			tokenSet[t] = struct{}{}
		}
	}

	ignoredPathTokens := map[string]bool{
		"search": true, "json": true, "comments": true,
		"new": true, "top": true, "hot": true,
	}
	pathSegmentExtRe := regexp.MustCompile(`\.[a-z0-9]+$`)
	for _, seg := range strings.Split(parsed.Path, "/") {
		cleaned := pathSegmentExtRe.ReplaceAllString(decodeTokenSource(seg), "")
		for _, t := range tokenize(cleaned) {
			if !ignoredPathTokens[t] {
				tokenSet[t] = struct{}{}
			}
		}
	}

	var out []string
	for t := range tokenSet {
		out = append(out, t)
	}
	return out
}

func flattenText(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []any:
		var parts []string
		for _, item := range v {
			if s := flattenText(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	case map[string]any:
		var parts []string
		for _, item := range v {
			if s := flattenText(item); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

func clampScore(value float64) *int {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil
	}
	s := int(math.Round(math.Max(0, math.Min(100, value))))
	return &s
}

// qualityLevel represents the quality level result.
type qualityLevel struct {
	Level string `json:"level"`
	Label string `json:"label"`
}

func qualityLevelFromScore(score *int) qualityLevel {
	if score == nil {
		return qualityLevel{Level: "unknown", Label: "No signal yet"}
	}
	s := *score
	if s >= 75 {
		return qualityLevel{Level: "strong", Label: "Strong"}
	}
	if s >= 45 {
		return qualityLevel{Level: "mixed", Label: "Mixed"}
	}
	return qualityLevel{Level: "weak", Label: "Weak"}
}

// ────────────────────────────────────────────────────────────────
// Compact report helpers
// ────────────────────────────────────────────────────────────────

type compactSocialClusterProductAngle struct {
	Idea        string `json:"idea"`
	TargetNiche string `json:"target_niche"`
	Why         string `json:"why"`
}

type compactSocialCluster struct {
	Name         string                            `json:"name"`
	PostCount    any                               `json:"post_count"`
	AvgPainScore any                               `json:"avg_pain_score"`
	Signals      any                               `json:"signals"`
	KeyQuotes    []string                          `json:"key_quotes"`
	ProductAngle *compactSocialClusterProductAngle `json:"product_angle"`
}

type compactSocialReportType struct {
	ID          int64                  `json:"id"`
	Title       string                 `json:"title"`
	Assessment  string                 `json:"assessment"`
	Clusters    []compactSocialCluster `json:"clusters"`
	CreatedAt   string                 `json:"created_at"`
	CompletedAt *string                `json:"completed_at"`
}

type compactGoogleReportType struct {
	ID               int64  `json:"id"`
	ExecutiveSummary any    `json:"executive_summary"`
	KeywordSummary   any    `json:"keyword_summary"`
	Opportunities    any    `json:"opportunities"`
	Risks            any    `json:"risks"`
	NextActions      any    `json:"next_actions"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func compactSocialReport(rep *repository.SocialReportForAI) *compactSocialReportType {
	if rep == nil {
		return nil
	}

	rawClusters := parseJSON(rep.Clusters, []any{})
	clusters, _ := rawClusters.([]any)

	var compact []compactSocialCluster
	limit := 8
	if len(clusters) < limit {
		limit = len(clusters)
	}
	for _, c := range clusters[:limit] {
		cm, _ := c.(map[string]any)
		if cm == nil {
			continue
		}
		var quotes []string
		if kq, ok := cm["key_quotes"].([]any); ok {
			maxQ := 3
			if len(kq) < maxQ {
				maxQ = len(kq)
			}
			for _, q := range kq[:maxQ] {
				quotes = append(quotes, trimText(fmt.Sprintf("%v", q), 240))
			}
		}
		var pa *compactSocialClusterProductAngle
		if paRaw, ok := cm["product_angle"].(map[string]any); ok {
			pa = &compactSocialClusterProductAngle{
				Idea:        trimText(fmt.Sprintf("%v", paRaw["idea"]), 200),
				TargetNiche: trimText(fmt.Sprintf("%v", paRaw["target_niche"]), 200),
				Why:         trimText(fmt.Sprintf("%v", paRaw["why"]), 300),
			}
		}
		compact = append(compact, compactSocialCluster{
			Name:         fmt.Sprintf("%v", cm["name"]),
			PostCount:    cm["post_count"],
			AvgPainScore: cm["avg_pain_score"],
			Signals:      cm["signals"],
			KeyQuotes:    quotes,
			ProductAngle: pa,
		})
	}

	var completedAt *string
	if rep.CompletedAt.Valid {
		completedAt = &rep.CompletedAt.String
	}

	return &compactSocialReportType{
		ID:          rep.ID,
		Title:       rep.Title,
		Assessment:  trimText(rep.Assessment, 1200),
		Clusters:    compact,
		CreatedAt:   rep.CreatedAt,
		CompletedAt: completedAt,
	}
}

func compactGoogleReport(rep *repository.GoogleReportForAI) *compactGoogleReportType {
	if rep == nil {
		return nil
	}
	sliceAny := func(v any, max int) any {
		if arr, ok := v.([]any); ok {
			if len(arr) > max {
				return arr[:max]
			}
			return arr
		}
		return v
	}
	return &compactGoogleReportType{
		ID:               rep.ID,
		ExecutiveSummary: parseJSON(rep.ExecutiveSummaryJSON, map[string]any{}),
		KeywordSummary:   sliceAny(parseJSON(rep.KeywordSummaryJSON, []any{}), 10),
		Opportunities:    sliceAny(parseJSON(rep.OpportunitiesJSON, []any{}), 10),
		Risks:            sliceAny(parseJSON(rep.RisksJSON, []any{}), 10),
		NextActions:      sliceAny(parseJSON(rep.NextActionsJSON, []any{}), 10),
		CreatedAt:        rep.CreatedAt,
		UpdatedAt:        rep.UpdatedAt,
	}
}

// ────────────────────────────────────────────────────────────────
// Google quality signal
// ────────────────────────────────────────────────────────────────

type googleQualityEntry struct {
	key    string
	tokens map[string]struct{}
}

type googleQualityData struct {
	reportID  *int64
	byKeyword map[string]keywordSignal
	entries   []googleQualityEntry
}

type keywordSignal struct {
	score   *int
	summary string
}

func buildGoogleQuality(rows []repository.GoogleKeywordRow, reportID int64) googleQualityData {
	byKeyword := make(map[string]keywordSignal, len(rows))
	entries := make([]googleQualityEntry, 0, len(rows))

	for _, row := range rows {
		total := float64(row.TotalResults)
		relevant := float64(row.RelevantResults)
		outreach := float64(row.OutreachCandidates)
		var relevantRatio, outreachRatio float64
		if total > 0 {
			relevantRatio = relevant / total
			outreachRatio = outreach / total
		}
		avgRelevance := row.AvgRelevanceScore
		avgConfidence := row.AvgConfidenceScore
		score := clampScore(relevantRatio*45 + outreachRatio*15 + avgRelevance*2.5 + avgConfidence*2)
		key := normalizeText(row.RootKeyword)
		byKeyword[key] = keywordSignal{
			score:   score,
			summary: fmt.Sprintf("Google: %d/%d relevant results, avg relevance %.1f.", row.RelevantResults, row.TotalResults, avgRelevance),
		}

		tokenSet := map[string]struct{}{}
		for _, t := range tokenize(row.RootKeyword) {
			tokenSet[t] = struct{}{}
		}
		entries = append(entries, googleQualityEntry{key: key, tokens: tokenSet})
	}

	rid := reportID
	return googleQualityData{reportID: &rid, byKeyword: byKeyword, entries: entries}
}

func getGoogleSignal(gq googleQualityData, queryURL string) *keywordSignal {
	exactKey := normalizeText(queryURL)
	if sig, ok := gq.byKeyword[exactKey]; ok {
		return &sig
	}

	queryTokens := map[string]struct{}{}
	for _, t := range tokenizeQueryURL(queryURL) {
		queryTokens[t] = struct{}{}
	}
	if len(queryTokens) == 0 {
		return nil
	}

	var bestKey string
	var bestRatio float64
	for _, entry := range gq.entries {
		if len(entry.tokens) == 0 {
			continue
		}
		overlap := 0
		for t := range queryTokens {
			if _, ok := entry.tokens[t]; ok {
				overlap++
			}
		}
		if overlap == 0 {
			continue
		}
		denominator := len(queryTokens)
		if len(entry.tokens) > denominator {
			denominator = len(entry.tokens)
		}
		ratio := float64(overlap) / float64(denominator)
		if ratio >= 0.5 && ratio > bestRatio {
			bestRatio = ratio
			bestKey = entry.key
		}
	}
	if bestKey == "" {
		return nil
	}
	sig := gq.byKeyword[bestKey]
	return &sig
}

// ────────────────────────────────────────────────────────────────
// Social quality context
// ────────────────────────────────────────────────────────────────

type socialClusterCtx struct {
	name      string
	postCount float64
	pain      float64
	tokens    map[string]struct{}
}

type socialQualityContext struct {
	clusters         []socialClusterCtx
	assessmentTokens map[string]struct{}
}

func buildSocialQualityContext(rep *repository.SocialReportForAI) *socialQualityContext {
	if rep == nil {
		return nil
	}
	rawClusters := parseJSON(rep.Clusters, []any{})
	clusters, _ := rawClusters.([]any)

	ctx := &socialQualityContext{}
	atTokens := map[string]struct{}{}
	for _, t := range tokenize(rep.Assessment) {
		atTokens[t] = struct{}{}
	}
	ctx.assessmentTokens = atTokens

	for _, c := range clusters {
		cm, _ := c.(map[string]any)
		if cm == nil {
			continue
		}
		name := fmt.Sprintf("%v", cm["name"])
		if strings.TrimSpace(name) == "" {
			name = "latest report theme"
		}
		var postCount, pain float64
		if pc, ok := cm["post_count"].(float64); ok {
			postCount = pc
		}
		if ap, ok := cm["avg_pain_score"].(float64); ok {
			pain = ap
		}
		tokenSet := map[string]struct{}{}
		for _, t := range tokenize(name) {
			tokenSet[t] = struct{}{}
		}
		for _, t := range tokenize(flattenText(cm["signals"])) {
			tokenSet[t] = struct{}{}
		}
		for _, t := range tokenize(flattenText(cm["key_quotes"])) {
			tokenSet[t] = struct{}{}
		}
		if pa, ok := cm["product_angle"].(map[string]any); ok {
			for _, t := range tokenize(fmt.Sprintf("%v", pa["idea"])) {
				tokenSet[t] = struct{}{}
			}
			for _, t := range tokenize(fmt.Sprintf("%v", pa["target_niche"])) {
				tokenSet[t] = struct{}{}
			}
			for _, t := range tokenize(fmt.Sprintf("%v", pa["why"])) {
				tokenSet[t] = struct{}{}
			}
		}
		ctx.clusters = append(ctx.clusters, socialClusterCtx{
			name: name, postCount: postCount, pain: pain, tokens: tokenSet,
		})
	}
	return ctx
}

type socialSignalResult struct {
	score   *int
	summary string
}

func buildLatestSocialQuality(ctx *socialQualityContext, queryURL, angle string) socialSignalResult {
	if ctx == nil {
		return socialSignalResult{score: nil, summary: "Social: no strong keyword overlap with the latest report themes."}
	}
	queryTokens := map[string]struct{}{}
	for _, t := range tokenizeQueryURL(queryURL) {
		queryTokens[t] = struct{}{}
	}
	for _, t := range tokenize(angle) {
		queryTokens[t] = struct{}{}
	}

	var bestScore *int
	var bestClusterName string
	var usedAssessmentOverlap bool

	for _, cluster := range ctx.clusters {
		overlap := 0
		for t := range queryTokens {
			if _, ok := cluster.tokens[t]; ok {
				overlap++
			}
		}
		if overlap == 0 {
			continue
		}
		var overlapRatio float64
		if len(queryTokens) > 0 {
			overlapRatio = float64(overlap) / float64(len(queryTokens))
		}
		postCount := cluster.postCount
		pain := cluster.pain
		score := clampScore(overlapRatio*55 + math.Min(postCount, 8)*3 + pain*4)
		if bestScore == nil || (score != nil && *score > *bestScore) {
			bestScore = score
			bestClusterName = cluster.name
		}
	}

	if bestClusterName == "" && len(queryTokens) > 0 {
		assessmentOverlap := 0
		for t := range queryTokens {
			if _, ok := ctx.assessmentTokens[t]; ok {
				assessmentOverlap++
			}
		}
		if assessmentOverlap > 0 {
			score := clampScore(float64(assessmentOverlap)/float64(len(queryTokens))*55 + 20)
			if bestScore == nil || (score != nil && *score > *bestScore) {
				bestScore = score
				usedAssessmentOverlap = true
			}
		}
	}

	if bestClusterName != "" {
		return socialSignalResult{
			score:   bestScore,
			summary: fmt.Sprintf(`Social: aligned with "%s" in the latest report.`, bestClusterName),
		}
	}
	if usedAssessmentOverlap && bestScore != nil {
		return socialSignalResult{
			score:   bestScore,
			summary: "Social: assessment language overlaps this query, but no direct cluster theme match was found.",
		}
	}
	return socialSignalResult{
		score:   nil,
		summary: "Social: no strong keyword overlap with the latest report themes.",
	}
}

// QueryQuality is the quality block added to each query.
type QueryQuality struct {
	Score   *int   `json:"score"`
	Level   string `json:"level"`
	Label   string `json:"label"`
	Summary string `json:"summary"`
	Sources struct {
		SocialReportID *int64 `json:"social_report_id"`
		GoogleReportID *int64 `json:"google_report_id"`
	} `json:"sources"`
}

func buildQueryQuality(
	platform, queryURL, angle string,
	socialRep *repository.SocialReportForAI,
	socialCtx *socialQualityContext,
	gq googleQualityData,
) QueryQuality {
	socialSignal := buildLatestSocialQuality(socialCtx, queryURL, angle)
	var googleSignalPtr *keywordSignal
	if platform == "google" {
		sig := getGoogleSignal(gq, queryURL)
		googleSignalPtr = sig
	}

	hasNoReports := socialRep == nil && gq.reportID == nil
	if hasNoReports {
		var q QueryQuality
		q.Score = nil
		q.Level = "unknown"
		q.Label = "No signal yet"
		q.Summary = "No completed social or Google reports yet."
		q.Sources.SocialReportID = nil
		q.Sources.GoogleReportID = nil
		return q
	}

	var googleSummary string
	if platform == "google" && gq.reportID != nil {
		if googleSignalPtr != nil {
			googleSummary = googleSignalPtr.summary
		} else {
			googleSummary = "Google: latest completed report available, no exact keyword summary match."
		}
	}

	var scores []int
	if socialSignal.score != nil {
		scores = append(scores, *socialSignal.score)
	}
	if googleSignalPtr != nil && googleSignalPtr.score != nil {
		scores = append(scores, *googleSignalPtr.score)
	}
	var score *int
	if len(scores) > 0 {
		sum := 0
		for _, s := range scores {
			sum += s
		}
		avg := float64(sum) / float64(len(scores))
		score = clampScore(avg)
	}

	ql := qualityLevelFromScore(score)

	onlyGoogleReportExists := socialRep == nil && gq.reportID != nil
	onlySocialReportExists := socialRep != nil && gq.reportID == nil

	var summary string
	switch {
	case platform == "google" && onlySocialReportExists:
		if socialSignal.summary != "" {
			summary = "Google: no completed Google report yet for this project; social report signal shown for context. " + socialSignal.summary
		} else {
			summary = "Google: no completed Google report yet for this project; social report is available."
		}
	case platform != "google" && onlyGoogleReportExists:
		summary = "Social: no completed social report yet for this project; Google report is available."
	default:
		var parts []string
		if googleSummary != "" {
			parts = append(parts, googleSummary)
		}
		if socialSignal.summary != "" {
			parts = append(parts, socialSignal.summary)
		}
		if len(parts) > 0 {
			summary = strings.Join(parts, " ")
		} else {
			summary = "No completed social or Google reports yet."
		}
	}

	var q QueryQuality
	q.Score = score
	q.Level = ql.Level
	q.Label = ql.Label
	q.Summary = summary
	if socialRep != nil {
		q.Sources.SocialReportID = &socialRep.ID
	}
	q.Sources.GoogleReportID = gq.reportID
	return q
}

// QueryWithQuality wraps a raw query map adding a quality field.
type QueryWithQuality struct {
	ID        int64        `json:"id"`
	ProjectID string       `json:"project_id"`
	Platform  string       `json:"platform"`
	QueryURL  string       `json:"query_url"`
	Angle     string       `json:"angle"`
	Enabled   int64        `json:"enabled"`
	CreatedAt string       `json:"created_at"`
	Quality   QueryQuality `json:"quality"`
}

// ────────────────────────────────────────────────────────────────
// Suggest / Refine methods on QueryService
// ────────────────────────────────────────────────────────────────

var validRefineSourceValues = map[string]bool{
	"current_queries":    true,
	"social_report":      true,
	"google_report":      true,
	"project_context":    true,
	"refinement_request": true,
}

var googleURLRe = regexp.MustCompile(`(?i)^https?://`)

const suggestSystemPrompt = `You are a social listening query expert. Given a project's context, generate 10 search queries to find relevant social media posts.

Rules for Reddit queries:
- DEFAULT to site-wide search: https://www.reddit.com/search.json?q={keywords}&sort=new&t=month&limit=100
- Only use subreddit-restricted search (https://www.reddit.com/r/{sub}/search.json?q=...&restrict_sr=on&sort=new&t=month&limit=100) when ALL of the following hold:
  1. The subreddit has >500k members AND is highly on-topic (e.g. r/entrepreneur, r/SaaS, r/smallbusiness)
  2. The keyword is 1-3 short tokens that naturally appear inside that community
  3. You can name the subreddit with confidence — never guess
- Keyword construction:
  - Keep total query string to ≤5 tokens. Reddit search is token-AND inside each OR clause.
  - Prefer ONE 2-3 token phrase (e.g. "brandwatch alternative") over multi-phrase OR chains
  - At most 2 short clauses joined by OR; NEVER chain 3+ multi-word phrases with OR
  - Use vocabulary users actually type ("brandwatch alternative", "cheaper than sprout"), NOT marketer framings ("cant afford brandwatch")
  - Quote multi-word phrases that must appear together: "sprout social" alternative
- Use t=month for niche/low-volume topics; t=week only for high-velocity general topics
- Before finalising a Reddit query, ask: "Would this keyword appear in at least 5 posts on reddit.com/search in the last month?" If no, rewrite shorter or switch to site-wide search

Rules for Bluesky queries:
- Use plain keyword search strings (NOT URLs), e.g. "running knee pain"
- Keep them 2-5 words, focused on the pain point

Rules for Google queries:
- Use plain root keywords only (NOT URLs), e.g. "resume coding project"
- Keep each root keyword concise (2-6 words)
- Focus on the problem phrasing users would search for

General rules:
- Auto-select the best platform for each query based on where that conversation naturally happens
- Each query gets a concise "angle" describing the pain point it targets (2-4 words)
- Avoid duplicating any existing queries provided in context
- Return ONLY a valid JSON array, no markdown fencing, no extra text

Response format:
[{"platform":"reddit","query_url":"...","angle":"..."},{"platform":"bluesky","query_url":"...","angle":"..."},{"platform":"google","query_url":"...","angle":"..."}]`

const refineSystemPrompt = `You are a senior research strategist improving an existing social listening query set.

You will receive:
- project context
- current queries (including whether they are enabled, and run performance stats)
- the latest social research report when available
- the latest Google report when available
- an optional human refinement note

Your job is to propose a safe, reviewable refinement plan for the query list.

Rules for recommendations:
- Return ONLY valid JSON, no markdown or commentary
- Recommend only two action types: "disable" and "add"
- AUTOMATICALLY recommend "disable" for any query where recent_runs_with_zero >= 3 — this is the strongest signal, stronger than any report finding
- Use "disable" when an enabled query is clearly too broad, weak-fit, redundant, outdated, contradicted by report findings, or persistently returns zero results
- Use "add" when a more targeted query should be introduced based on repeated themes, stronger language, better platform fit, or clear report opportunities
- When disabling a zero-hit Reddit query, always pair it with an "add" replacement using shorter keywords or site-wide search
- Never mutate or remove existing queries directly; only recommend actions
- Prefer specific, higher-signal replacements over generic broad terms
- Keep disable recommendations conservative for queries with results; be aggressive for queries with zero results across 3+ runs
- Use concise reasons tied to the supplied evidence

Query format rules:
- Reddit query_url must be a full JSON search URL
- DEFAULT to site-wide: https://www.reddit.com/search.json?q={keywords}&sort=new&t=month&limit=100
- Only use subreddit-restricted search when the subreddit has >500k members, the keyword is ≤3 tokens, and the fit is clear
- Keep Reddit query keywords to ≤5 tokens total; prefer 1-3 token phrases over multi-phrase OR chains
- Bluesky query_url must be a plain keyword search string, 2-5 words
- Google query_url must be a plain root keyword, not a URL, usually 2-6 words
- Each add recommendation must include a concise angle

Response format:
{
  "summary": "Short overall summary",
  "recommendations": [
    {
      "type": "disable",
      "reason": "Why this query should be disabled",
      "sources": ["current_queries", "social_report", "google_report"],
      "query": { "id": 12 }
    },
    {
      "type": "add",
      "reason": "Why this query should be added",
      "sources": ["social_report"],
      "replaces_query_id": 12,
      "query": { "platform": "google", "query_url": "developer onboarding burnout", "angle": "onboarding burnout" }
    }
  ]
}`

// Suggest generates AI-powered query suggestions for a project.
func (s *QueryService) Suggest(ctx context.Context, projectID string, req SuggestRequest) (SuggestResponse, int, string) {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		code, msg := mapError(err)
		return SuggestResponse{}, code, msg
	}

	existingQueries, err := s.repo.ListAllQueries(ctx, projectID)
	if err != nil {
		return SuggestResponse{}, http.StatusInternalServerError, "Internal server error"
	}

	refinement := strings.TrimSpace(req.Refinement)

	userMsg := fmt.Sprintf(`Project: "%s"`, project.Name)
	if project.ScoringPrompt != nil && *project.ScoringPrompt != "" {
		userMsg += "\n\nScoring criteria:\n" + *project.ScoringPrompt
	}

	var enabledQueries []map[string]string
	existingKeys := map[string]struct{}{}
	for _, q := range existingQueries {
		existingKeys[buildQueryKey(q.Platform, q.QueryURL)] = struct{}{}
		if q.Enabled != 0 {
			enabledQueries = append(enabledQueries, map[string]string{
				"platform":  q.Platform,
				"query_url": q.QueryURL,
				"angle":     q.Angle,
			})
		}
	}

	if len(enabledQueries) > 0 {
		enc, _ := json.Marshal(enabledQueries)
		userMsg += "\n\nExisting queries (avoid duplicates):\n" + string(enc)
	}
	if refinement != "" {
		userMsg += "\n\nRefinement request:\n" + refinement
	}

	raw, _, err := s.ai.GenerateForTask(ctx, "query_suggestion", suggestSystemPrompt, userMsg)
	if err != nil {
		if strings.Contains(err.Error(), "all AI providers failed") {
			return SuggestResponse{
				Suggestions: []SuggestedQuery{},
				Notice:      "AI suggestions are currently unavailable in this runtime because no provider is configured. Add a provider in host settings or set ANTHROPIC_API_KEY / GEMINI_API_KEY to enable suggestions.",
			}, http.StatusOK, ""
		}
		return SuggestResponse{}, http.StatusInternalServerError, "Failed to generate suggestions, try again"
	}

	cleaned := sanitizeAIJSON(raw)

	suggestions, ok := parseSuggestionArray(cleaned)
	if !ok {
		return SuggestResponse{}, http.StatusInternalServerError, "Failed to generate suggestions, try again"
	}

	var out []SuggestedQuery
	for _, s := range suggestions {
		platform := mapKeyAny(s, "platform", "Platform")
		queryURL := mapKeyAny(s, "query_url", "queryUrl", "queryURL", "QueryURL")
		angle := mapKeyAny(s, "angle", "Angle")
		if platform == "" || queryURL == "" || angle == "" {
			continue
		}
		if !validPlatforms[platform] {
			continue
		}
		if platform == "google" && googleURLRe.MatchString(strings.TrimSpace(queryURL)) {
			continue
		}
		key := buildQueryKey(platform, queryURL)
		if _, exists := existingKeys[key]; exists {
			continue
		}
		out = append(out, SuggestedQuery{Platform: platform, QueryURL: queryURL, Angle: angle})
	}
	if out == nil {
		out = []SuggestedQuery{}
	}
	return SuggestResponse{Suggestions: out}, http.StatusOK, ""
}

// existingQueryRow is a minimal query shape used in refine.
type existingQueryRow struct {
	ID       int64
	Platform string
	QueryURL string
	Angle    string
	Enabled  bool
}

// Refine generates AI-powered query refinement recommendations.
func (s *QueryService) Refine(ctx context.Context, projectID string, req RefineRequest) (RefineResponse, int, string) {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		code, msg := mapError(err)
		return RefineResponse{}, code, msg
	}

	existingQueries, err := s.repo.ListAllQueries(ctx, projectID)
	if err != nil {
		return RefineResponse{}, http.StatusInternalServerError, "Internal server error"
	}

	refinement := strings.TrimSpace(req.Refinement)

	// Resolve social report
	var selectedReportID int64
	if project.SelectedReportID != nil {
		selectedReportID = *project.SelectedReportID
	}
	socialRep, socialSource, err := s.repo.PickSocialReport(ctx, projectID, req.SocialReportID, selectedReportID)
	if err != nil {
		return RefineResponse{}, http.StatusInternalServerError, "Internal server error"
	}

	googleRep, googleSource, err := s.repo.PickGoogleReport(ctx, projectID, req.GoogleReportID)
	if err != nil {
		return RefineResponse{}, http.StatusInternalServerError, "Internal server error"
	}

	compactSocial := compactSocialReport(socialRep)
	compactGoogle := compactGoogleReport(googleRep)

	// Fetch run performance stats (last 10 completed runs per platform).
	platformStats, _ := s.repo.RecentPlatformStats(ctx, projectID, 10)
	type platformStatInput struct {
		Platform     string `json:"platform"`
		TotalRuns    int    `json:"total_runs"`
		RunsWithZero int    `json:"runs_with_zero"`
		MaxFound     int64  `json:"max_posts_found"`
	}
	var runStatsInput []platformStatInput
	for _, ps := range platformStats {
		runStatsInput = append(runStatsInput, platformStatInput{
			Platform:     ps.Platform,
			TotalRuns:    ps.TotalRuns,
			RunsWithZero: ps.RunsWithZero,
			MaxFound:     ps.LastPostsFound,
		})
	}

	// Build user message
	type queryInput struct {
		ID               int64  `json:"id"`
		Platform         string `json:"platform"`
		QueryURL         string `json:"query_url"`
		Angle            string `json:"angle"`
		Enabled          bool   `json:"enabled"`
		RecentRunsWithZero int  `json:"recent_runs_with_zero"`
	}
	// Build platform zero-run lookup for annotation.
	platformZeroRuns := map[string]int{}
	for _, ps := range platformStats {
		platformZeroRuns[ps.Platform] = ps.RunsWithZero
	}
	var enabledQs []queryInput
	for _, q := range existingQueries {
		if q.Enabled != 0 {
			enabledQs = append(enabledQs, queryInput{
				ID: q.ID, Platform: q.Platform,
				QueryURL: q.QueryURL, Angle: q.Angle, Enabled: true,
				RecentRunsWithZero: platformZeroRuns[q.Platform],
			})
		}
	}

	type projInput struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		ScoringPrompt string `json:"scoring_prompt"`
	}
	scoringPrompt := ""
	if project.ScoringPrompt != nil {
		scoringPrompt = *project.ScoringPrompt
	}
	payload := map[string]any{
		"project":            projInput{ID: project.ID, Name: project.Name, ScoringPrompt: scoringPrompt},
		"current_queries":    enabledQs,
		"platform_run_stats": runStatsInput,
		"social_report":      compactSocial,
		"google_report":      compactGoogle,
		"refinement_request": nilIfEmpty(refinement),
	}
	if enabledQs == nil {
		payload["current_queries"] = []queryInput{}
	}
	if runStatsInput == nil {
		payload["platform_run_stats"] = []platformStatInput{}
	}

	userMsg, _ := json.MarshalIndent(payload, "", "  ")

	raw, _, err := s.ai.GenerateForTask(ctx, "query_refinement", refineSystemPrompt, string(userMsg))
	if err != nil {
		if strings.Contains(err.Error(), "all AI providers failed") {
			enabledCount := 0
			for _, q := range existingQueries {
				if q.Enabled != 0 {
					enabledCount++
				}
			}

			var socialRef *RefineReportRef
			if socialRep != nil {
				socialRef = &RefineReportRef{ID: socialRep.ID, Source: socialSource}
			}
			var googleRef *RefineReportRef
			if googleRep != nil {
				googleRef = &RefineReportRef{ID: googleRep.ID, Source: googleSource}
			}

			return RefineResponse{
				Summary: "AI refinement is unavailable (no AI provider is configured in this runtime). Returning query context without recommendations.",
				Context: RefineContext{
					QueryCount:        len(existingQueries),
					EnabledQueryCount: enabledCount,
					SocialReport:      socialRef,
					GoogleReport:      googleRef,
				},
				Recommendations: []RefineRecommendation{},
			}, http.StatusOK, ""
		}
		return RefineResponse{}, http.StatusInternalServerError, "Failed to generate refinement suggestions, try again"
	}

	cleaned := sanitizeAIJSON(raw)

	parsed, ok := parseRefineObject(cleaned)
	if !ok {
		return RefineResponse{}, http.StatusInternalServerError, "Failed to generate refinement suggestions, try again"
	}

	// Build existingById map
	existingByID := map[int64]existingQueryRow{}
	seenAddKeys := map[string]struct{}{}
	for _, q := range existingQueries {
		existingByID[q.ID] = existingQueryRow{
			ID: q.ID, Platform: q.Platform,
			QueryURL: q.QueryURL, Angle: q.Angle,
			Enabled: q.Enabled != 0,
		}
		seenAddKeys[buildQueryKey(q.Platform, q.QueryURL)] = struct{}{}
	}
	seenDisableIDs := map[int64]struct{}{}

	var recommendations []RefineRecommendation
	if rawRecs, ok := parsed["recommendations"].([]any); ok {
		for _, item := range rawRecs {
			rec := sanitizeRefineRecommendation(item, existingByID, seenAddKeys, seenDisableIDs)
			if rec != nil {
				recommendations = append(recommendations, *rec)
			}
		}
	}
	if recommendations == nil {
		recommendations = []RefineRecommendation{}
	}

	var socialRef *RefineReportRef
	if socialRep != nil {
		socialRef = &RefineReportRef{ID: socialRep.ID, Source: socialSource}
	}
	var googleRef *RefineReportRef
	if googleRep != nil {
		googleRef = &RefineReportRef{ID: googleRep.ID, Source: googleSource}
	}

	enabledCount := 0
	for _, q := range existingQueries {
		if q.Enabled != 0 {
			enabledCount++
		}
	}

	resp := RefineResponse{
		Summary: trimText(fmt.Sprintf("%v", parsed["summary"]), 400),
		Context: RefineContext{
			QueryCount:        len(existingQueries),
			EnabledQueryCount: enabledCount,
			SocialReport:      socialRef,
			GoogleReport:      googleRef,
		},
		Recommendations: recommendations,
	}
	return resp, http.StatusOK, ""
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func sanitizeAIJSON(value string) string {
	cleaned := strings.TrimSpace(value)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = regexp.MustCompile("(?s)^```(?:json)?\\n?").ReplaceAllString(cleaned, "")
		cleaned = regexp.MustCompile("\\n?```$").ReplaceAllString(cleaned, "")
		cleaned = strings.TrimSpace(cleaned)
	}
	return cleaned
}

func extractJSONArray(value string) (string, bool) {
	start := strings.Index(value, "[")
	end := strings.LastIndex(value, "]")
	if start < 0 || end < 0 || end <= start {
		return "", false
	}
	return strings.TrimSpace(value[start : end+1]), true
}

func extractJSONObject(value string) (string, bool) {
	start := strings.Index(value, "{")
	end := strings.LastIndex(value, "}")
	if start < 0 || end < 0 || end <= start {
		return "", false
	}
	return strings.TrimSpace(value[start : end+1]), true
}

func parseSuggestionArray(value string) ([]map[string]any, bool) {
	var suggestions []map[string]any
	if err := json.Unmarshal([]byte(value), &suggestions); err == nil {
		return suggestions, true
	}
	extracted, ok := extractJSONArray(value)
	if !ok {
		return nil, false
	}
	if err := json.Unmarshal([]byte(extracted), &suggestions); err != nil {
		return nil, false
	}
	return suggestions, true
}

func parseRefineObject(value string) (map[string]any, bool) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(value), &parsed); err == nil {
		return parsed, true
	}
	extracted, ok := extractJSONObject(value)
	if !ok {
		return nil, false
	}
	if err := json.Unmarshal([]byte(extracted), &parsed); err != nil {
		return nil, false
	}
	return parsed, true
}

func sanitizeRefineRecommendation(
	rawAny any,
	existingByID map[int64]existingQueryRow,
	seenAddKeys map[string]struct{},
	seenDisableIDs map[int64]struct{},
) *RefineRecommendation {
	raw, _ := rawAny.(map[string]any)
	if raw == nil {
		return nil
	}

	rtype, _ := raw["type"].(string)
	if rtype != "disable" && rtype != "add" {
		return nil
	}

	reason := trimText(fmt.Sprintf("%v", raw["reason"]), 500)
	var sources []string
	if srcArr, ok := raw["sources"].([]any); ok {
		seen := map[string]bool{}
		for _, src := range srcArr {
			s, _ := src.(string)
			if validRefineSourceValues[s] && !seen[s] {
				sources = append(sources, s)
				seen[s] = true
			}
		}
	}
	if sources == nil {
		sources = []string{}
	}

	if rtype == "disable" {
		queryMap, _ := raw["query"].(map[string]any)
		var queryID int64
		if qid, ok := queryMap["id"].(float64); ok {
			queryID = int64(qid)
		} else if qidStr, ok := raw["query_id"].(float64); ok {
			queryID = int64(qidStr)
		}
		if queryID == 0 {
			return nil
		}
		existing, ok := existingByID[queryID]
		if !ok || !existing.Enabled {
			return nil
		}
		if _, seen := seenDisableIDs[queryID]; seen {
			return nil
		}
		seenDisableIDs[queryID] = struct{}{}

		qJSON, _ := json.Marshal(map[string]any{
			"id":        existing.ID,
			"platform":  existing.Platform,
			"query_url": existing.QueryURL,
			"angle":     existing.Angle,
			"enabled":   existing.Enabled,
		})
		recID := fmt.Sprintf("disable:%d", queryID)
		return &RefineRecommendation{
			ID:      recID,
			Type:    "disable",
			Reason:  reason,
			Sources: sources,
			Query:   json.RawMessage(qJSON),
		}
	}

	// type == "add"
	qMap, _ := raw["query"].(map[string]any)
	if qMap == nil {
		return nil
	}
	platform, _ := qMap["platform"].(string)
	queryURL, _ := qMap["query_url"].(string)
	queryURL = strings.TrimSpace(queryURL)
	angle, _ := qMap["angle"].(string)
	angle = strings.TrimSpace(angle)

	if !validPlatforms[platform] || queryURL == "" || angle == "" {
		return nil
	}
	if platform == "google" && googleURLRe.MatchString(queryURL) {
		return nil
	}

	addKey := buildQueryKey(platform, queryURL)
	if _, seen := seenAddKeys[addKey]; seen {
		return nil
	}
	seenAddKeys[addKey] = struct{}{}

	qJSON, _ := json.Marshal(map[string]string{
		"platform":  platform,
		"query_url": queryURL,
		"angle":     angle,
	})

	recID := "add:" + buildQueryKey(platform, queryURL)
	rec := &RefineRecommendation{
		ID:      recID,
		Type:    "add",
		Reason:  reason,
		Sources: sources,
		Query:   json.RawMessage(qJSON),
	}

	if replacesRaw, ok := raw["replaces_query_id"].(float64); ok {
		replacesID := int64(replacesRaw)
		if _, exists := existingByID[replacesID]; exists {
			rec.ReplacesQueryID = &replacesID
		}
	}

	return rec
}

// mapKeyAny returns the string value for the first matching key in the map,
// trying each candidate in order. Returns "" if none match.
func mapKeyAny(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok {
			return v
		}
	}
	return ""
}
