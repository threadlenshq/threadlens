package google

import (
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	mdImageRe       = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	mdLinkRe        = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	mdLinkTextRe    = regexp.MustCompile(`\[([^\]]*)\]`)
	scaffoldURLRe   = regexp.MustCompile(`https?://\S+`)
	sectionTitleRe  = regexp.MustCompile(`(?i)section title\s*:?`)
	contentLabelRe  = regexp.MustCompile(`(?i)\bcontent\s*:`)
	numEntityRe     = regexp.MustCompile(`&#x?[0-9a-fA-F]+;`)
	namedEntityRe   = regexp.MustCompile(`&[a-z]+;`)
	scaffoldPunctRe = regexp.MustCompile(`[>•·|#]+`)
	nonAlphaNumRe   = regexp.MustCompile(`[^a-z0-9]+`)
)

// contentPlatforms mirrors CONTENT_PLATFORMS from report-builder.js.
var contentPlatforms = map[string]bool{
	"reddit.com": true, "old.reddit.com": true, "www.reddit.com": true,
	"medium.com":           true,
	"dev.to":               true,
	"github.com":           true,
	"docs.github.com":      true,
	"gist.github.com":      true,
	"stackoverflow.com":    true,
	"stackexchange.com":    true,
	"quora.com":            true,
	"youtube.com":          true,
	"youtu.be":             true,
	"twitter.com":          true,
	"x.com":                true,
	"news.ycombinator.com": true,
	"hackernews.com":       true,
	"substack.com":         true,
	"linkedin.com":         true,
	"facebook.com":         true,
	"producthunt.com":      true,
	"indiehackers.com":     true,
}

var stopwords = map[string]bool{
	"about": true, "above": true, "after": true, "again": true, "against": true, "along": true,
	"already": true, "alone": true, "among": true, "article": true, "author": true,
	"because": true, "before": true, "being": true, "below": true, "between": true, "beyond": true,
	"cannot": true, "could": true, "content": true,
	"during": true,
	"every":  true, "everyone": true,
	"first": true, "further": true,
	"going":  true,
	"having": true, "heading": true, "header": true,
	"image": true, "images": true,
	"might": true, "more": true, "most": true,
	"never": true, "nothing": true,
	"often": true, "other": true, "others": true, "ourselves": true,
	"posts": true, "post": true,
	"really": true, "reading": true, "rights": true, "reserved": true,
	"section": true, "should": true, "since": true, "source": true, "start": true, "still": true,
	"tags": true, "their": true, "there": true, "these": true, "thing": true, "things": true,
	"those": true, "through": true, "title": true,
	"under": true, "until": true, "using": true,
	"website": true, "where": true, "which": true, "while": true, "whose": true, "would": true, "write": true, "writer": true,
}

// stripScaffolding mirrors stripScaffolding() from report-builder.js.
func stripScaffolding(text string) string {
	out := text
	out = mdImageRe.ReplaceAllString(out, " ")
	out = mdLinkRe.ReplaceAllStringFunc(out, func(m string) string {
		inner := mdLinkTextRe.FindStringSubmatch(m)
		if len(inner) > 1 {
			return inner[1] + " "
		}
		return " "
	})
	out = scaffoldURLRe.ReplaceAllString(out, " ")
	out = sectionTitleRe.ReplaceAllString(out, " ")
	out = contentLabelRe.ReplaceAllString(out, " ")
	out = numEntityRe.ReplaceAllString(out, " ")
	out = namedEntityRe.ReplaceAllString(out, " ")
	out = scaffoldPunctRe.ReplaceAllString(out, " ")
	out = whitespaceRe.ReplaceAllString(out, " ")
	return strings.TrimSpace(out)
}

func cleanLabel(text string, max int) string {
	s := stripScaffolding(text)
	if len(s) > max {
		return s[:max]
	}
	return s
}

// KeywordSummary represents a keyword performance summary row.
type KeywordSummary struct {
	Keyword           string  `json:"keyword,omitempty"`
	RootKeyword       string  `json:"root_keyword,omitempty"`
	Query             string  `json:"query,omitempty"`
	OpportunityScore  float64 `json:"opportunityScore,omitempty"`
	OpportunityScore2 float64 `json:"opportunity_score,omitempty"`
	AvgRelevanceScore float64 `json:"avg_relevance_score,omitempty"`
	AvgRelevance      float64 `json:"avg_relevance,omitempty"`
	HitCount          float64 `json:"hitCount,omitempty"`
	HitCount2         float64 `json:"hit_count,omitempty"`
	RepeatCount       float64 `json:"repeat_count,omitempty"`
	TotalResults      float64 `json:"total_results,omitempty"`
}

// DomainStat represents a domain frequency stat.
type DomainStat struct {
	Domain      string  `json:"domain"`
	Count       float64 `json:"count,omitempty"`
	ResultCount float64 `json:"result_count,omitempty"`
	RepeatCount float64 `json:"repeat_count,omitempty"`
}

// ReportInput is the input struct for BuildGoogleReport.
type ReportInput struct {
	AnalyzedResults  []AnalyzedResult `json:"analyzedResults"`
	KeywordSummaries []KeywordSummary `json:"keywordSummaries"`
	DomainStats      []DomainStat     `json:"domainStats"`
	RootKeywords     []string         `json:"rootKeywords"`
}

// Opportunity represents a recommended opportunity.
type Opportunity struct {
	Kind   string  `json:"kind"`
	Label  string  `json:"label"`
	URL    string  `json:"url,omitempty"`
	Score  float64 `json:"score,omitempty"`
	Domain string  `json:"domain,omitempty"`
	Poster *string `json:"poster"`
	Count  int     `json:"count,omitempty"`
}

// NextAction represents a recommended next action.
type NextAction struct {
	Action string `json:"action"`
	Target string `json:"target"`
	URL    string `json:"url,omitempty"`
	Reason string `json:"reason"`
}

// Risk represents a report risk.
type Risk struct {
	Level  string `json:"level"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
}

// KeywordScore is a keyword with opportunity score.
type KeywordScore struct {
	Keyword          string  `json:"keyword"`
	OpportunityScore float64 `json:"opportunityScore"`
}

// PainTheme is a recurring pain theme entry.
type PainTheme struct {
	Value    string `json:"value"`
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// PhraseCount is a phrase with count.
type PhraseCount struct {
	Phrase string `json:"phrase"`
	Count  int    `json:"count"`
}

// OutreachCandidate is an outreach candidate with title and URL.
type OutreachCandidate struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// MentionedProduct is an aggregated product mention.
type MentionedProduct struct {
	Kind      string   `json:"kind"`
	Name      string   `json:"name"`
	Count     int      `json:"count"`
	ResultIDs []string `json:"result_ids"`
}

// Report is the output of BuildGoogleReport.
type Report struct {
	RecommendedOpportunities     []Opportunity       `json:"recommended_opportunities"`
	RecommendedNextActions       []NextAction        `json:"recommended_next_actions"`
	ReportRisks                  []Risk              `json:"report_risks"`
	MentionedProducts            []MentionedProduct  `json:"mentioned_products"`
	TopInsights                  []string            `json:"top_insights"`
	BestKeywordsForSEO           []KeywordScore      `json:"best_keywords_for_seo"`
	StrongestRecurringPainThemes []PainTheme         `json:"strongest_recurring_pain_themes"`
	BestContentGapOpportunities  []string            `json:"best_content_gap_opportunities"`
	PotentialCompetitorTargets   []string            `json:"potential_competitor_comparison_targets"`
	PotentialOutreachCandidates  []OutreachCandidate `json:"potential_outreach_candidates"`
	LowValueKeywordsToDrop       []string            `json:"low_value_keywords_to_drop"`
	InterestingPhrases           []PhraseCount       `json:"interesting_phrases_for_messaging"`
}

func keywordSummaryOpportunityScore(kw KeywordSummary) float64 {
	for _, v := range []float64{kw.OpportunityScore, kw.OpportunityScore2, kw.AvgRelevanceScore, kw.AvgRelevance} {
		if v != 0 {
			return v
		}
	}
	return 0
}

func keywordSummaryHitCount(kw KeywordSummary) float64 {
	for _, v := range []float64{kw.HitCount, kw.HitCount2, kw.RepeatCount, kw.TotalResults} {
		if v != 0 {
			return v
		}
	}
	return 0
}

func keywordSummaryValue(kw KeywordSummary) string {
	for _, s := range []string{kw.Keyword, kw.RootKeyword, kw.Query} {
		if s != "" {
			return s
		}
	}
	return ""
}

func domainStatCount(d DomainStat) float64 {
	for _, v := range []float64{d.Count, d.ResultCount, d.RepeatCount} {
		if v != 0 {
			return v
		}
	}
	return 0
}

func relevanceScoreOf(r AnalyzedResult) float64 {
	return r.RelevanceScore
}

func filterByOpportunityType(items []AnalyzedResult, t string) []AnalyzedResult {
	var out []AnalyzedResult
	for _, r := range items {
		for _, ot := range r.OpportunityTypes {
			if ot == t {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

func topByRelevance(items []AnalyzedResult, limit int) []AnalyzedResult {
	sorted := make([]AnalyzedResult, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		return relevanceScoreOf(sorted[i]) > relevanceScoreOf(sorted[j])
	})
	if limit < len(sorted) {
		return sorted[:limit]
	}
	return sorted
}

func topDomainsByCount(domains []DomainStat, limit int) []DomainStat {
	sorted := make([]DomainStat, len(domains))
	copy(sorted, domains)
	sort.SliceStable(sorted, func(i, j int) bool {
		return domainStatCount(sorted[i]) > domainStatCount(sorted[j])
	})
	if limit < len(sorted) {
		return sorted[:limit]
	}
	return sorted
}

func topKeywordsByScore(keywords []KeywordSummary, limit int) []KeywordSummary {
	sorted := make([]KeywordSummary, len(keywords))
	copy(sorted, keywords)
	sort.SliceStable(sorted, func(i, j int) bool {
		return keywordSummaryOpportunityScore(sorted[i]) > keywordSummaryOpportunityScore(sorted[j])
	})
	if limit < len(sorted) {
		return sorted[:limit]
	}
	return sorted
}

func getURL(r AnalyzedResult) string {
	if r.URL != "" {
		return r.URL
	}
	return r.DisplayURL
}

// ExtractPoster extracts the poster/author identifier from a URL.
// Mirrors extractPoster() from report-builder.js.
func ExtractPoster(rawURL string) *string {
	if rawURL == "" {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	hostname := strings.ToLower(strings.TrimPrefix(parsed.Hostname(), "www."))
	segments := []string{}
	for _, s := range strings.Split(parsed.Path, "/") {
		if s != "" {
			segments = append(segments, s)
		}
	}

	if hostname == "reddit.com" || hostname == "old.reddit.com" {
		if len(segments) >= 2 && segments[0] == "r" {
			s := "r/" + segments[1]
			return &s
		}
		if len(segments) >= 2 && (segments[0] == "user" || segments[0] == "u") {
			s := "u/" + segments[1]
			return &s
		}
		return nil
	}

	if hostname == "github.com" {
		excluded := map[string]bool{"orgs": true, "users": true, "topics": true, "explore": true, "marketplace": true}
		if len(segments) >= 2 && !excluded[segments[0]] {
			return &segments[0]
		}
		return nil
	}

	if hostname == "medium.com" {
		if len(segments) > 0 {
			return &segments[0]
		}
		return nil
	}

	if hostname == "dev.to" {
		if len(segments) > 0 {
			return &segments[0]
		}
		return nil
	}

	if strings.HasSuffix(hostname, ".substack.com") && hostname != "substack.com" {
		s := strings.TrimSuffix(hostname, ".substack.com")
		return &s
	}

	if hostname == "youtube.com" {
		if len(segments) > 0 && strings.HasPrefix(segments[0], "@") {
			return &segments[0]
		}
		if len(segments) >= 2 && segments[0] == "c" {
			return &segments[1]
		}
		if len(segments) >= 2 && segments[0] == "user" {
			return &segments[1]
		}
		return nil
	}

	return nil
}

type topSourceEntry struct {
	domain string
	poster *string
	count  int
}

func buildTopSourceEntries(analyzedResults []AnalyzedResult, platformDomainStats []DomainStat) []topSourceEntry {
	grouped := map[string]*topSourceEntry{}
	var order []string

	for _, r := range analyzedResults {
		if !contentPlatforms[r.Domain] {
			continue
		}
		rawURL := r.CanonicalURL
		if rawURL == "" {
			rawURL = r.URL
		}
		poster := ExtractPoster(rawURL)
		var key string
		if poster != nil {
			key = r.Domain + "|" + *poster
		} else {
			key = r.Domain
		}
		if _, exists := grouped[key]; !exists {
			grouped[key] = &topSourceEntry{domain: r.Domain, poster: poster, count: 0}
			order = append(order, key)
		}
		grouped[key].count++
	}

	if len(grouped) == 0 && len(platformDomainStats) > 0 {
		for _, stat := range platformDomainStats {
			if _, exists := grouped[stat.Domain]; !exists {
				grouped[stat.Domain] = &topSourceEntry{domain: stat.Domain, poster: nil, count: int(domainStatCount(stat))}
				order = append(order, stat.Domain)
			}
		}
	}

	entries := make([]topSourceEntry, 0, len(order))
	for _, k := range order {
		entries = append(entries, *grouped[k])
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})
	if len(entries) > 5 {
		entries = entries[:5]
	}
	return entries
}

func buildOpportunities(topContentGaps []AnalyzedResult, domainStats []DomainStat, analyzedResults []AnalyzedResult) []Opportunity {
	var opportunities []Opportunity

	for _, item := range topContentGaps {
		label := cleanLabel(item.Title, 160)
		if label == "" {
			label = cleanLabel(item.Summary, 160)
		}
		if label == "" {
			continue
		}
		opportunities = append(opportunities, Opportunity{
			Kind:  "content_gap",
			Label: label,
			URL:   getURL(item),
			Score: relevanceScoreOf(item),
		})
	}

	if len(domainStats) > 0 {
		var platformDomainStats []DomainStat
		var sites []DomainStat
		for _, d := range domainStats {
			if contentPlatforms[d.Domain] {
				platformDomainStats = append(platformDomainStats, d)
			} else {
				sites = append(sites, d)
			}
		}

		if len(platformDomainStats) > 0 {
			topSourceEntries := buildTopSourceEntries(analyzedResults, platformDomainStats)
			for _, entry := range topSourceEntries {
				label := entry.domain
				if entry.poster != nil {
					label = *entry.poster
				}
				opportunities = append(opportunities, Opportunity{
					Kind:   "top_source",
					Domain: entry.domain,
					Poster: entry.poster,
					Label:  label,
					Count:  entry.count,
				})
			}
		}

		for _, domain := range topDomainsByCount(sites, 5) {
			if domain.Domain == "" {
				continue
			}
			opportunities = append(opportunities, Opportunity{
				Kind:  "competitor",
				Label: domain.Domain,
				Count: int(domainStatCount(domain)),
			})
		}
	}

	if len(opportunities) == 0 {
		competitors := filterByOpportunityType(analyzedResults, "competitor_weakness")
		if len(competitors) > 5 {
			competitors = competitors[:5]
		}
		for _, item := range competitors {
			label := cleanLabel(item.Summary, 160)
			if label == "" {
				label = cleanLabel(item.Title, 160)
			}
			opportunities = append(opportunities, Opportunity{
				Kind:  "competitor",
				Label: label,
				URL:   getURL(item),
			})
		}
	}

	return opportunities
}

func buildNextActions(outreachCandidates, topContentGaps []AnalyzedResult) []NextAction {
	var actions []NextAction

	for _, item := range topByRelevance(outreachCandidates, 5) {
		target := cleanLabel(item.Title, 160)
		if target == "" {
			target = cleanLabel(item.Summary, 160)
		}
		if target == "" {
			continue
		}
		actions = append(actions, NextAction{
			Action: "Engage author",
			Target: target,
			URL:    getURL(item),
			Reason: "Direct fit + outreach candidate (no disqualifiers)",
		})
	}

	gapSlice := topContentGaps
	if len(gapSlice) > 3 {
		gapSlice = gapSlice[:3]
	}
	for _, item := range gapSlice {
		target := cleanLabel(item.Title, 160)
		if target == "" {
			target = cleanLabel(item.Summary, 160)
		}
		if target == "" {
			continue
		}
		actions = append(actions, NextAction{
			Action: "Create content filling gap",
			Target: target,
			URL:    getURL(item),
			Reason: "High problem signal, low actionability in existing result",
		})
	}

	if len(actions) == 0 {
		actions = append(actions, NextAction{
			Action: "Rerun with tighter queries",
			Target: "No direct-fit outreach candidates or content gaps were identified",
			Reason: "Current query set produced mostly weak or adjacent fits",
		})
	}

	return actions
}

func buildRisks(analyzedResults []AnalyzedResult, outreachCandidates []AnalyzedResult, keywordSummaries []KeywordSummary) []Risk {
	var risks []Risk
	total := len(analyzedResults)
	if total == 0 {
		return []Risk{{Level: "high", Label: "No results were analyzed", Detail: "Pipeline returned zero rows. Check provider and queries."}}
	}

	weakFit := 0
	shortSummaries := 0
	for _, item := range analyzedResults {
		if item.RelevanceFit == "weak_fit" {
			weakFit++
		}
		if len(item.Summary) < 120 {
			shortSummaries++
		}
	}

	weakFitPct := int(math.Round(float64(weakFit) / float64(total) * 100))
	if weakFitPct >= 50 {
		level := "medium"
		if weakFitPct >= 70 {
			level = "high"
		}
		risks = append(risks, Risk{
			Level:  level,
			Label:  strings.Replace("X% of results are weak fit", "X", strconv.Itoa(weakFitPct), 1),
			Detail: "Consider tighter root keywords or adding forum-biased queries.",
		})
	}

	outreach := len(outreachCandidates)
	if outreach == 0 {
		risks = append(risks, Risk{
			Level:  "high",
			Label:  "No outreach candidates identified",
			Detail: "All direct-fit results were disqualified, or none reached direct_fit.",
		})
	} else if outreach < 5 {
		risks = append(risks, Risk{
			Level:  "low",
			Label:  "Only " + strconv.Itoa(outreach) + " outreach candidates",
			Detail: "Low signal for engagement work on this run.",
		})
	}

	shortPct := int(math.Round(float64(shortSummaries) / float64(total) * 100))
	if shortPct >= 30 {
		risks = append(risks, Risk{
			Level:  "medium",
			Label:  strings.Replace("X% of results have thin content", "X", strconv.Itoa(shortPct), 1),
			Detail: "Fetcher likely failed or pages are JS-heavy. Relevance scoring is biased toward titles.",
		})
	}

	count := 0
	for _, kw := range keywordSummaries {
		if count >= 5 {
			break
		}
		oScore := keywordSummaryOpportunityScore(kw)
		hCount := keywordSummaryHitCount(kw)
		if oScore <= 2 || hCount == 0 {
			name := keywordSummaryValue(kw)
			if name == "" {
				continue
			}
			risks = append(risks, Risk{
				Level:  "low",
				Label:  "Drop weak keyword: " + name,
				Detail: "Low average relevance or zero meaningful hits.",
			})
			count++
		}
	}

	return risks
}

func tokenizeKeywords(keywords []string) map[string]bool {
	set := make(map[string]bool)
	for _, kw := range keywords {
		tokens := nonAlphaNumRe.Split(strings.ToLower(kw), -1)
		for _, t := range tokens {
			if len(t) >= 3 {
				set[t] = true
			}
		}
	}
	return set
}

func rootKeywordsFromSummaries(keywordSummaries []KeywordSummary) []string {
	var out []string
	for _, kw := range keywordSummaries {
		v := keywordSummaryValue(kw)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func buildPainThemeCounts(results []AnalyzedResult) map[string]int {
	counts := make(map[string]int)
	type termEntry struct {
		term     string
		category string
	}
	var terms []termEntry
	for _, t := range PROBLEM_TERMS {
		terms = append(terms, termEntry{strings.ToLower(t), "problem"})
	}
	for _, t := range WORKFLOW_TERMS {
		terms = append(terms, termEntry{strings.ToLower(t), "workflow"})
	}

	for _, item := range results {
		haystack := strings.ToLower(stripScaffolding(item.Summary))
		if haystack == "" {
			continue
		}
		seen := map[string]bool{}
		for _, te := range terms {
			if seen[te.term] {
				continue
			}
			if strings.Contains(haystack, te.term) {
				seen[te.term] = true
				key := te.category + ":" + te.term
				counts[key]++
			}
		}
	}
	return counts
}

func buildPhraseCounts(results []AnalyzedResult, excludedTokens map[string]bool) map[string]int {
	counts := make(map[string]int)
	for _, item := range results {
		stripped := strings.ToLower(stripScaffolding(item.Summary))
		if stripped == "" {
			continue
		}
		tokens := nonAlphaNumRe.Split(stripped, -1)
		for _, token := range tokens {
			if len(token) < 5 {
				continue
			}
			if stopwords[token] {
				continue
			}
			if excludedTokens[token] {
				continue
			}
			counts[token]++
		}
	}
	return counts
}

// BuildMentionedProducts aggregates mentioned products across results.
// Mirrors buildMentionedProducts() from report-builder.js.
func BuildMentionedProducts(analyzedResults []AnalyzedResult) []MentionedProduct {
	canonicalName := make(map[string]string)
	resultIDs := make(map[string][]string)
	var mentions []string

	for _, result := range analyzedResults {
		id := result.URL
		if id == "" {
			id = result.CanonicalURL
		}
		for _, name := range result.MentionedProducts {
			key := strings.TrimSpace(strings.ToLower(name))
			if key == "" {
				continue
			}
			if _, exists := canonicalName[key]; !exists {
				canonicalName[key] = strings.TrimSpace(name)
				resultIDs[key] = []string{}
			}
			if id != "" {
				resultIDs[key] = append(resultIDs[key], id)
			}
			mentions = append(mentions, key)
		}
	}

	// Count
	counts := make(map[string]int)
	for _, k := range mentions {
		counts[k]++
	}

	// Sort by count desc
	type pair struct {
		key   string
		count int
	}
	var pairs []pair
	for k, c := range counts {
		pairs = append(pairs, pair{k, c})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})
	if len(pairs) > 10 {
		pairs = pairs[:10]
	}

	out := make([]MentionedProduct, 0, len(pairs))
	for _, p := range pairs {
		// Dedupe result IDs
		seen := map[string]bool{}
		var uniqIDs []string
		for _, id := range resultIDs[p.key] {
			if !seen[id] {
				seen[id] = true
				uniqIDs = append(uniqIDs, id)
			}
		}
		out = append(out, MentionedProduct{
			Kind:      "mentioned_product",
			Name:      canonicalName[p.key],
			Count:     p.count,
			ResultIDs: uniqIDs,
		})
	}
	return out
}

// BuildGoogleReport builds a full Google scout report.
// Mirrors buildGoogleReport() from report-builder.js.
func BuildGoogleReport(input ReportInput) Report {
	analyzedResults := input.AnalyzedResults
	if analyzedResults == nil {
		analyzedResults = []AnalyzedResult{}
	}
	keywordSummaries := input.KeywordSummaries
	if keywordSummaries == nil {
		keywordSummaries = []KeywordSummary{}
	}
	domainStats := input.DomainStats
	if domainStats == nil {
		domainStats = []DomainStat{}
	}

	var directFits []AnalyzedResult
	var outreachCandidates []AnalyzedResult
	for _, r := range analyzedResults {
		if r.RelevanceFit == "direct_fit" {
			directFits = append(directFits, r)
		}
		if r.OutreachCandidate == 1 {
			outreachCandidates = append(outreachCandidates, r)
		}
	}

	effectiveRootKeywords := input.RootKeywords
	if len(effectiveRootKeywords) == 0 {
		effectiveRootKeywords = rootKeywordsFromSummaries(keywordSummaries)
	}
	excludedTokens := tokenizeKeywords(effectiveRootKeywords)

	painThemeCounts := buildPainThemeCounts(analyzedResults)
	phraseCounts := buildPhraseCounts(analyzedResults, excludedTokens)

	var competitorTargets []string
	if len(domainStats) > 0 {
		for _, d := range topDomainsByCount(domainStats, 5) {
			if d.Domain != "" {
				competitorTargets = append(competitorTargets, d.Domain)
			}
		}
	} else {
		competitors := filterByOpportunityType(analyzedResults, "competitor_weakness")
		if len(competitors) > 5 {
			competitors = competitors[:5]
		}
		for _, item := range competitors {
			v := item.Summary
			if v == "" {
				v = item.Title
			}
			if v != "" {
				competitorTargets = append(competitorTargets, v)
			}
		}
	}

	topContentGaps := topByRelevance(filterByOpportunityType(directFits, "content_gap"), 5)

	recommendedOpportunities := buildOpportunities(topContentGaps, domainStats, analyzedResults)
	recommendedNextActions := buildNextActions(outreachCandidates, topContentGaps)
	reportRisks := buildRisks(analyzedResults, outreachCandidates, keywordSummaries)
	mentionedProducts := BuildMentionedProducts(analyzedResults)

	// Top insights
	topInsights := []string{}
	for _, item := range topByRelevance(analyzedResults, 5) {
		s := item.Summary
		if s == "" {
			s = item.Title
		}
		s = stripScaffolding(s)
		if s != "" {
			topInsights = append(topInsights, s)
		}
	}

	// Best keywords for SEO
	bestKeywords := []KeywordScore{}
	for _, kw := range topKeywordsByScore(keywordSummaries, 5) {
		v := keywordSummaryValue(kw)
		if v != "" {
			bestKeywords = append(bestKeywords, KeywordScore{
				Keyword:          v,
				OpportunityScore: keywordSummaryOpportunityScore(kw),
			})
		}
	}

	// Strongest recurring pain themes
	type kv struct {
		key   string
		count int
	}
	var painPairs []kv
	for k, c := range painThemeCounts {
		painPairs = append(painPairs, kv{k, c})
	}
	sort.SliceStable(painPairs, func(i, j int) bool {
		if painPairs[i].count != painPairs[j].count {
			return painPairs[i].count > painPairs[j].count
		}
		return painPairs[i].key < painPairs[j].key
	})
	if len(painPairs) > 7 {
		painPairs = painPairs[:7]
	}
	painThemes := make([]PainTheme, 0, len(painPairs))
	for _, p := range painPairs {
		parts := strings.SplitN(p.key, ":", 2)
		if len(parts) == 2 {
			painThemes = append(painThemes, PainTheme{Value: parts[1], Category: parts[0], Count: p.count})
		}
	}

	// Best content gap opportunities
	bestContentGaps := []string{}
	for _, item := range topContentGaps {
		s := item.Summary
		if s == "" {
			s = item.Title
		}
		s = stripScaffolding(s)
		if s != "" {
			bestContentGaps = append(bestContentGaps, s)
		}
	}

	// Potential competitor targets
	if competitorTargets == nil {
		competitorTargets = []string{}
	}

	// Potential outreach candidates
	outreachList := []OutreachCandidate{}
	for _, item := range topByRelevance(outreachCandidates, 5) {
		outreachList = append(outreachList, OutreachCandidate{
			Title: item.Title,
			URL:   getURL(item),
		})
	}

	// Low value keywords
	lowValueKeywords := []string{}
	count := 0
	for _, kw := range keywordSummaries {
		if count >= 5 {
			break
		}
		oScore := keywordSummaryOpportunityScore(kw)
		hCount := keywordSummaryHitCount(kw)
		if oScore <= 2 || hCount == 0 {
			v := keywordSummaryValue(kw)
			if v != "" {
				lowValueKeywords = append(lowValueKeywords, v)
				count++
			}
		}
	}

	// Interesting phrases
	type phraseKV struct {
		phrase string
		count  int
	}
	var phrasePairs []phraseKV
	for p, c := range phraseCounts {
		phrasePairs = append(phrasePairs, phraseKV{p, c})
	}
	sort.SliceStable(phrasePairs, func(i, j int) bool {
		if phrasePairs[i].count != phrasePairs[j].count {
			return phrasePairs[i].count > phrasePairs[j].count
		}
		return phrasePairs[i].phrase < phrasePairs[j].phrase
	})
	if len(phrasePairs) > 12 {
		phrasePairs = phrasePairs[:12]
	}
	interestingPhrases := make([]PhraseCount, 0, len(phrasePairs))
	for _, p := range phrasePairs {
		interestingPhrases = append(interestingPhrases, PhraseCount{Phrase: p.phrase, Count: p.count})
	}

	// Ensure nil slices become empty slices
	if recommendedOpportunities == nil {
		recommendedOpportunities = []Opportunity{}
	}
	if mentionedProducts == nil {
		mentionedProducts = []MentionedProduct{}
	}
	if outreachList == nil {
		outreachList = []OutreachCandidate{}
	}

	return Report{
		RecommendedOpportunities:     recommendedOpportunities,
		RecommendedNextActions:       recommendedNextActions,
		ReportRisks:                  reportRisks,
		MentionedProducts:            mentionedProducts,
		TopInsights:                  topInsights,
		BestKeywordsForSEO:           bestKeywords,
		StrongestRecurringPainThemes: painThemes,
		BestContentGapOpportunities:  bestContentGaps,
		PotentialCompetitorTargets:   competitorTargets,
		PotentialOutreachCandidates:  outreachList,
		LowValueKeywordsToDrop:       lowValueKeywords,
		InterestingPhrases:           interestingPhrases,
	}
}
