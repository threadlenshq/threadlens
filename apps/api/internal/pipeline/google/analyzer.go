package google

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// SearchResult is the input search result structure to AnalyzeResult.
// It also carries the raw provider fields (rank, result_type, published_at, author, content_hash)
// so the runner can persist them without a separate struct.
type SearchResult struct {
	Title        string   `json:"title"`
	Snippet      string   `json:"snippet"`
	URL          string   `json:"url"`
	DisplayURL   string   `json:"display_url"`
	PageText     string   `json:"page_text"`
	Rank         *float64 `json:"rank,omitempty"`
	ResultType   string   `json:"result_type,omitempty"`
	PublishedAt  *string  `json:"published_at,omitempty"`
	Author       string   `json:"author,omitempty"`
	ContentHash  string   `json:"content_hash,omitempty"`
	CanonicalURL string   `json:"canonical_url,omitempty"`
}

// AnalysisContext provides root keyword and intent context.
type AnalysisContext struct {
	RootKeyword   string `json:"rootKeyword"`
	IntentType    string `json:"intentType"`
	ProductIntent bool   `json:"productIntent"`
}

// ScoreBreakdown mirrors score_breakdown from analyzer.js.
type ScoreBreakdown struct {
	Problem       int `json:"problem"`
	Audience      int `json:"audience"`
	Workflow      int `json:"workflow"`
	Actionability int `json:"actionability"`
}

// AnalyzedResult mirrors the return shape of analyzeResult() from analyzer.js.
type AnalyzedResult struct {
	// Input fields (preserved)
	Title             string   `json:"title,omitempty"`
	Snippet           string   `json:"snippet,omitempty"`
	URL               string   `json:"url,omitempty"`
	DisplayURL        string   `json:"display_url,omitempty"`
	Rank              *float64 `json:"rank,omitempty"`
	Domain            string   `json:"domain,omitempty"`
	CanonicalURL      string   `json:"canonical_url,omitempty"`
	Sources           []string `json:"sources,omitempty"`
	ContentHash       string   `json:"content_hash,omitempty"`
	Summary           string   `json:"summary,omitempty"`
	Appearances       int      `json:"appearances,omitempty"`
	DuplicateURLs     []string `json:"duplicate_urls,omitempty"`
	RelevanceScore    float64  `json:"relevance_score,omitempty"`
	RelevanceFit      string   `json:"relevance_fit,omitempty"`
	OpportunityTypes  []string `json:"opportunity_types,omitempty"`
	OutreachCandidate int      `json:"outreach_candidate"`
	MentionedProducts []string `json:"mentioned_products,omitempty"`

	// Analyzer-computed fields
	ContentType          string         `json:"content_type"`
	IntentType           string         `json:"intent_type"`
	ConfidenceScore      float64        `json:"confidence_score"`
	KeegoingFitReasons   []string       `json:"keepgoing_fit_reasons"`
	Disqualifiers        []string       `json:"disqualifiers"`
	ActionRecommendation string         `json:"action_recommendation"`
	ScoreBreakdown       ScoreBreakdown `json:"score_breakdown"`
}

var contentTypeRules = []struct {
	contentType string
	test        func(url, text string) bool
}{
	{"forum", func(url, _ string) bool {
		return regexp.MustCompile(`reddit\.com|news\.ycombinator\.com|stackoverflow\.com`).MatchString(url)
	}},
	{"guide", func(_, text string) bool {
		return regexp.MustCompile(`\b(guide|tutorial|checklist|step[- ]by[- ]step|how to)\b`).MatchString(text)
	}},
	{"comparison", func(_, text string) bool {
		return regexp.MustCompile(`\b(best|compare|comparison|vs\.?|alternative|review)\b`).MatchString(text)
	}},
	{"product", func(_, text string) bool {
		return regexp.MustCompile(`\b(pricing|product|signup|features|demo)\b`).MatchString(text)
	}},
	{"article", func(_, _ string) bool { return true }},
}

var intentTypeRules = []struct {
	intentType string
	test       func(title, text string) bool
}{
	{"problem", func(title, text string) bool {
		return regexp.MustCompile(`\?$`).MatchString(title) || regexp.MustCompile(`\b(help|struggle|stuck|pain|problem|difficult|hard)\b`).MatchString(text)
	}},
	{"evaluation", func(_, text string) bool {
		return regexp.MustCompile(`\b(best|compare|comparison|vs\.?|alternative|review)\b`).MatchString(text)
	}},
	{"solution", func(_, text string) bool {
		return regexp.MustCompile(`\b(guide|tutorial|template|checklist|step[- ]by[- ]step|solution|workflow)\b`).MatchString(text)
	}},
	{"informational", func(_, _ string) bool { return true }},
}

func normalizeTextLower(value string) string {
	return strings.ToLower(value)
}

type analysisCtx struct {
	title    string
	snippet  string
	pageText string
	url      string
	text     string
}

func gatherContext(sr SearchResult, fetched FetchResult) analysisCtx {
	title := strings.TrimSpace(sr.Title)
	snippet := strings.TrimSpace(sr.Snippet)
	pageText := strings.TrimSpace(fetched.PageText)
	if pageText == "" {
		pageText = strings.TrimSpace(sr.PageText)
	}
	rawURL := strings.TrimSpace(sr.URL)
	if rawURL == "" {
		rawURL = strings.TrimSpace(sr.DisplayURL)
	}
	if rawURL == "" {
		rawURL = fetched.FinalURL
	}
	text := strings.ToLower(fmt.Sprintf("%s %s %s", title, snippet, pageText))
	return analysisCtx{title: title, snippet: snippet, pageText: pageText, url: strings.ToLower(rawURL), text: text}
}

type rootKeywordSignal struct {
	rootKeyword        string
	rootKeywordMatched bool
	rootKeywordScore   float64
}

func getRootKeywordSignal(text string, kctx AnalysisContext) rootKeywordSignal {
	rootKeyword := strings.TrimSpace(normalizeTextLower(kctx.RootKeyword))
	if rootKeyword == "" {
		return rootKeywordSignal{}
	}
	if strings.Contains(text, rootKeyword) {
		return rootKeywordSignal{rootKeyword: rootKeyword, rootKeywordMatched: true, rootKeywordScore: 1}
	}
	tokens := []string{}
	for _, t := range strings.Fields(rootKeyword) {
		if len(t) > 2 {
			tokens = append(tokens, t)
		}
	}
	if len(tokens) == 0 {
		return rootKeywordSignal{rootKeyword: rootKeyword}
	}
	matches := 0
	for _, t := range tokens {
		if strings.Contains(text, t) {
			matches++
		}
	}
	matched := matches >= int(math.Ceil(float64(len(tokens))/2))
	score := 0.0
	if matched {
		score = 0.5
	}
	return rootKeywordSignal{rootKeyword: rootKeyword, rootKeywordMatched: matched, rootKeywordScore: score}
}

func countMatches(text string, terms []string) int {
	count := 0
	for _, term := range terms {
		if strings.Contains(text, strings.ToLower(term)) {
			count++
		}
	}
	return count
}

func scoreDimension(text string, terms []string, max int) int {
	n := countMatches(text, terms)
	if n > max {
		return max
	}
	return n
}

func detectContentType(url, text string) string {
	for _, rule := range contentTypeRules {
		if rule.test(url, text) {
			return rule.contentType
		}
	}
	return "article"
}

func detectIntentType(title, text string) string {
	for _, rule := range intentTypeRules {
		if rule.test(title, text) {
			return rule.intentType
		}
	}
	return "informational"
}

func getOpportunityTypes(problemScore, actionabilityScore int, intentType, contentType string, hasProductSignal bool) []string {
	var opportunities []string
	if problemScore >= 3 && actionabilityScore <= 1 {
		opportunities = append(opportunities, "content_gap")
	}
	if problemScore >= 3 && hasProductSignal {
		opportunities = append(opportunities, "product_gap")
	}
	if intentType == "evaluation" || contentType == "comparison" {
		opportunities = append(opportunities, "competitor_weakness")
	}
	if len(opportunities) == 0 {
		return []string{"none"}
	}
	return opportunities
}

func getFitReasons(text string, problemScore, audienceScore, workflowScore, actionabilityScore int, rootKeywordMatched bool) []string {
	var reasons []string
	if problemScore >= 2 {
		reasons = append(reasons, "clear pain signal language")
	}
	if audienceScore >= 1 {
		reasons = append(reasons, "targets developers directly")
	}
	if workflowScore >= 1 {
		reasons = append(reasons, "mentions restart/resume workflow friction")
	}
	if actionabilityScore >= 1 {
		reasons = append(reasons, "contains actionable solution patterns")
	}
	if regexp.MustCompile(`resume coding project|re-entry|lost context`).MatchString(text) {
		reasons = append(reasons, "strong keepgoing re-entry signal")
	}
	if rootKeywordMatched {
		reasons = append(reasons, "matches root keyword context")
	}
	return reasons
}

func getDisqualifiers(text, url string) []string {
	var disqualifiers []string
	if regexp.MustCompile(`job|hiring|salary`).MatchString(text) {
		disqualifiers = append(disqualifiers, "career-only topic")
	}
	if strings.Contains(url, "news") {
		disqualifiers = append(disqualifiers, "news-heavy source")
	}
	return disqualifiers
}

func mapActionRecommendation(relevanceFit, intentType, contentType string) string {
	if relevanceFit == "direct_fit" && intentType == "problem" {
		return "create_solution_content"
	}
	if relevanceFit == "direct_fit" && (intentType == "evaluation" || contentType == "comparison") {
		return "build_comparison_landing"
	}
	if relevanceFit == "direct_fit" {
		return "monitor_and_aggregate_patterns"
	}
	if relevanceFit == "adjacent_fit" {
		return "track_for_trend_confirmation"
	}
	return "ignore_for_now"
}

// AnalyzeResult analyzes a search result and returns scoring/classification.
// Mirrors analyzeResult() from analyzer.js.
func AnalyzeResult(result SearchResult, fetched FetchResult, kctx AnalysisContext) AnalyzedResult {
	ctx := gatherContext(result, fetched)
	contentType := detectContentType(ctx.url, ctx.text)
	intentType := detectIntentType(ctx.title, ctx.text)
	rkSignal := getRootKeywordSignal(ctx.text, kctx)

	problemScore := scoreDimension(ctx.text, PROBLEM_TERMS, 4)
	audienceScore := scoreDimension(ctx.text, AUDIENCE_TERMS, 2)
	workflowScore := scoreDimension(ctx.text, WORKFLOW_TERMS, 2)
	actionabilityScore := scoreDimension(ctx.text, ACTIONABILITY_TERMS, 2)

	hasProductSignal := regexp.MustCompile(`\b(tool|app|product|saas|platform|pricing|demo|features|signup)\b`).MatchString(ctx.text) ||
		kctx.IntentType == "product" || kctx.ProductIntent

	rawRelevance := float64(problemScore+audienceScore+workflowScore+actionabilityScore) + rkSignal.rootKeywordScore
	relevanceScore := math.Round(rawRelevance*100) / 100

	relevanceFit := "weak_fit"
	if relevanceScore >= 8 {
		relevanceFit = "direct_fit"
	} else if relevanceScore >= 5 {
		relevanceFit = "adjacent_fit"
	}

	textLength := len(ctx.pageText) + len(ctx.snippet) + len(ctx.title)
	titlePart := 0.0
	if ctx.title != "" {
		titlePart = 0.3
	}
	snippetPart := 0.0
	if ctx.snippet != "" {
		snippetPart = 0.3
	}
	pagePart := 0.0
	if len(ctx.pageText) >= 120 {
		pagePart = 0.4
	} else if ctx.pageText != "" {
		pagePart = 0.2
	}
	fetchQuality := math.Min(1, titlePart+snippetPart+pagePart)
	denom := math.Max(4, math.Ceil(float64(textLength)/80))
	signalDensity := math.Min(1, float64(problemScore+audienceScore+workflowScore+actionabilityScore)/denom)
	confidenceScore := math.Round(10*(fetchQuality*0.6+signalDensity*0.4)*100) / 100

	opportunityTypes := getOpportunityTypes(problemScore, actionabilityScore, intentType, contentType, hasProductSignal)
	keepgoingFitReasons := getFitReasons(ctx.text, problemScore, audienceScore, workflowScore, actionabilityScore, rkSignal.rootKeywordMatched)
	disqualifiers := getDisqualifiers(ctx.text, ctx.url)

	actionRecommendation := mapActionRecommendation(relevanceFit, intentType, contentType)
	outreachCandidate := 0
	if relevanceFit == "direct_fit" && len(disqualifiers) == 0 {
		outreachCandidate = 1
	}

	summaryParts := []string{}
	if ctx.title != "" {
		summaryParts = append(summaryParts, ctx.title)
	}
	if ctx.snippet != "" {
		summaryParts = append(summaryParts, ctx.snippet)
	}
	summary := strings.Join(summaryParts, " — ")
	if len(summary) > 300 {
		summary = summary[:300]
	}

	if keepgoingFitReasons == nil {
		keepgoingFitReasons = []string{}
	}
	if disqualifiers == nil {
		disqualifiers = []string{}
	}

	return AnalyzedResult{
		ContentType:          contentType,
		IntentType:           intentType,
		RelevanceFit:         relevanceFit,
		RelevanceScore:       relevanceScore,
		ConfidenceScore:      confidenceScore,
		OpportunityTypes:     opportunityTypes,
		KeegoingFitReasons:   keepgoingFitReasons,
		Disqualifiers:        disqualifiers,
		Summary:              summary,
		ActionRecommendation: actionRecommendation,
		OutreachCandidate:    outreachCandidate,
		ScoreBreakdown: ScoreBreakdown{
			Problem:       problemScore,
			Audience:      audienceScore,
			Workflow:      workflowScore,
			Actionability: actionabilityScore,
		},
	}
}
