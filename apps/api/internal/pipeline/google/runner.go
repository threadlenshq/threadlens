package google

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/url"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// RunResult is the outcome of a Google scout run.
type RunResult struct {
	RunID        int64
	PostsChecked int64
	PostsFound   int64
}

// deriveObjective mirrors deriveObjective() from runner.js.
func deriveObjective(project domain.Project, rootKeyword string) string {
	var subject string
	if project.Description != nil && strings.TrimSpace(*project.Description) != "" {
		subject = strings.TrimSpace(*project.Description)
	} else if strings.TrimSpace(project.Name) != "" {
		subject = strings.TrimSpace(project.Name)
	} else if strings.TrimSpace(rootKeyword) != "" {
		subject = strings.TrimSpace(rootKeyword)
	} else {
		subject = "the topic"
	}
	return fmt.Sprintf(
		"Find web pages, forum posts, and articles where users discuss pain points, frustrations, or unmet needs related to %s. Prioritize first-person accounts and specific complaints over marketing content.",
		subject,
	)
}

// parseDomain extracts the domain (without www.) from a URL string.
func parseDomain(rawURL string) string {
	s := strings.TrimSpace(rawURL)
	if s == "" {
		return ""
	}
	parsed, err := url.Parse(s)
	if err != nil {
		// fallback: split on first / or ?
		host := strings.Split(s, "/")[0]
		host = strings.ToLower(strings.TrimPrefix(host, "www."))
		return host
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.TrimPrefix(host, "www.")
}

// average returns the average of a slice, or nil if empty.
func average(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	avg := math.Round(sum/float64(len(values))*1000) / 1000
	return &avg
}

// countBy builds a map[string]int from a slice using a key extractor.
func countBy(items []normalizedResult, key func(normalizedResult) string) map[string]int {
	counts := make(map[string]int)
	for _, item := range items {
		k := strings.TrimSpace(key(item))
		if k == "" {
			continue
		}
		counts[k]++
	}
	return counts
}

// FilterInput is the exported shape passed to a ResultFilter callback.
// It contains only the fields needed to classify a Google result.
type FilterInput struct {
	Title        string
	Snippet      string
	URL          string
	Domain       string
	CanonicalURL string
	PageText     string
}

// ResultFilter is a function that classifies a single Google search result.
// Returning a filtered decision causes the result to be persisted but excluded
// from report-visible summaries and counts.
type ResultFilter func(ctx context.Context, projectID string, input FilterInput) (domain.FilterDecision, error)


// mirroring the object shape from normalizeSearchResult() in runner.js.
type normalizedResult struct {
	RootKeyword          string
	Query                string
	Title                string
	URL                  string
	DisplayURL           string
	Snippet              string
	Rank                 *float64
	ResultType           string
	Domain               string
	PublishedAt          *string
	Author               string
	PageText             string
	ContentType          string
	IntentType           string
	RelevanceFit         string
	RelevanceScore       *float64
	ConfidenceScore      *float64
	OpportunityTypes     []string
	FitReasons           []string
	Disqualifiers        []string
	Summary              string
	ActionRecommendation string
	OutreachCandidate    int
	CanonicalURL         string
	ContentHash          string
	Sources              []string
	MentionedProducts    []string
	FilterDecision       domain.FilterDecision
}

// buildNormalizedResult mirrors normalizeSearchResult() from runner.js.
func buildNormalizedResult(result SearchResult, rootKeyword, query string, fetched FetchResult, analyzed AnalyzedResult) normalizedResult {
	normalizedURL := strings.TrimSpace(fetched.FinalURL)
	if normalizedURL == "" {
		normalizedURL = strings.TrimSpace(result.URL)
	}
	canonicalURL := strings.TrimSpace(result.CanonicalURL)
	if canonicalURL == "" {
		canonicalURL = CanonicalizeURL(normalizedURL)
	}
	domain := parseDomain(canonicalURL)
	if domain == "" {
		domain = parseDomain(normalizedURL)
	}

	nr := normalizedResult{
		RootKeyword:          rootKeyword,
		Query:                query,
		Title:                strings.TrimSpace(result.Title),
		URL:                  normalizedURL,
		DisplayURL:           strings.TrimSpace(result.DisplayURL),
		Snippet:              strings.TrimSpace(result.Snippet),
		Rank:                 result.Rank,
		ResultType:           strings.TrimSpace(result.ResultType),
		Domain:               domain,
		PublishedAt:          result.PublishedAt,
		Author:               strings.TrimSpace(result.Author),
		PageText:             strings.TrimSpace(fetched.PageText),
		ContentType:          analyzed.ContentType,
		IntentType:           analyzed.IntentType,
		RelevanceFit:         analyzed.RelevanceFit,
		OpportunityTypes:     analyzed.OpportunityTypes,
		FitReasons:           analyzed.KeegoingFitReasons,
		Disqualifiers:        analyzed.Disqualifiers,
		Summary:              analyzed.Summary,
		ActionRecommendation: analyzed.ActionRecommendation,
		OutreachCandidate:    analyzed.OutreachCandidate,
		CanonicalURL:         canonicalURL,
		ContentHash:          strings.TrimSpace(result.ContentHash),
		Sources:              []string{rootKeyword},
		MentionedProducts:    []string{},
	}
	if nr.ResultType == "" {
		nr.ResultType = "organic"
	}
	if analyzed.RelevanceScore != 0 {
		v := analyzed.RelevanceScore
		nr.RelevanceScore = &v
	}
	if analyzed.ConfidenceScore != 0 {
		v := analyzed.ConfidenceScore
		nr.ConfidenceScore = &v
	}
	if nr.OpportunityTypes == nil {
		nr.OpportunityTypes = []string{}
	}
	if nr.FitReasons == nil {
		nr.FitReasons = []string{}
	}
	if nr.Disqualifiers == nil {
		nr.Disqualifiers = []string{}
	}
	return nr
}

// getAttributedRootKeywords returns the set of root keywords for a normalizedResult.
func getAttributedRootKeywords(r normalizedResult) map[string]bool {
	kws := make(map[string]bool)
	if k := strings.TrimSpace(r.RootKeyword); k != "" {
		kws[k] = true
	}
	for _, s := range r.Sources {
		if k := strings.TrimSpace(s); k != "" {
			kws[k] = true
		}
	}
	return kws
}

// buildKeywordSummaries mirrors buildKeywordSummaries() from runner.js.
func buildKeywordSummaries(results []normalizedResult, rootKeywords []string) []repository.GoogleRunKeywordSummary {
	summaries := make([]repository.GoogleRunKeywordSummary, 0, len(rootKeywords))
	for _, kw := range rootKeywords {
		var kwResults []normalizedResult
		for _, r := range results {
			if getAttributedRootKeywords(r)[kw] {
				kwResults = append(kwResults, r)
			}
		}
		var relevantResults []normalizedResult
		var outreachCandidates []normalizedResult
		var relScores, confScores []float64
		for _, r := range kwResults {
			if r.RelevanceFit != "weak_fit" {
				relevantResults = append(relevantResults, r)
			}
			if r.OutreachCandidate == 1 {
				outreachCandidates = append(outreachCandidates, r)
			}
			if r.RelevanceScore != nil {
				relScores = append(relScores, *r.RelevanceScore)
			}
			if r.ConfidenceScore != nil {
				confScores = append(confScores, *r.ConfidenceScore)
			}
		}

		summaries = append(summaries, repository.GoogleRunKeywordSummary{
			RootKeyword:        kw,
			TotalResults:       len(kwResults),
			RelevantResults:    len(relevantResults),
			OutreachCandidates: len(outreachCandidates),
			AvgRelevanceScore:  average(relScores),
			AvgConfidenceScore: average(confScores),
			ResultTypesJSON:    countBy(kwResults, func(r normalizedResult) string { return r.ResultType }),
			ContentTypesJSON:   countBy(kwResults, func(r normalizedResult) string { return r.ContentType }),
			IntentTypesJSON:    countBy(kwResults, func(r normalizedResult) string { return r.IntentType }),
			RecommendationJSON: countBy(kwResults, func(r normalizedResult) string { return r.ActionRecommendation }),
		})
	}
	return summaries
}

type domainStatEntry struct {
	domain  string
	results []normalizedResult
}

// buildDomainStats mirrors buildDomainStats() from runner.js.
func buildDomainStats(results []normalizedResult) []repository.GoogleRunDomainStat {
	domainMap := make(map[string]*domainStatEntry)
	var order []string
	for _, r := range results {
		d := r.Domain
		if d == "" {
			continue
		}
		if _, ok := domainMap[d]; !ok {
			domainMap[d] = &domainStatEntry{domain: d}
			order = append(order, d)
		}
		domainMap[d].results = append(domainMap[d].results, r)
	}

	type topEntry struct {
		Key   string `json:"key"`
		Count int    `json:"count"`
	}
	toTopArray := func(counts map[string]int, limit int) []topEntry {
		type kv struct {
			key   string
			count int
		}
		var pairs []kv
		for k, c := range counts {
			pairs = append(pairs, kv{k, c})
		}
		// sort by count desc, then key asc
		for i := 0; i < len(pairs); i++ {
			for j := i + 1; j < len(pairs); j++ {
				if pairs[j].count > pairs[i].count || (pairs[j].count == pairs[i].count && pairs[j].key < pairs[i].key) {
					pairs[i], pairs[j] = pairs[j], pairs[i]
				}
			}
		}
		if limit < len(pairs) {
			pairs = pairs[:limit]
		}
		out := make([]topEntry, len(pairs))
		for i, p := range pairs {
			out[i] = topEntry{Key: p.key, Count: p.count}
		}
		return out
	}

	stats := make([]repository.GoogleRunDomainStat, 0, len(order))
	for _, d := range order {
		entry := domainMap[d]
		var relScores, confScores []float64
		var relevant, outreach int
		for _, r := range entry.results {
			if r.RelevanceFit != "weak_fit" {
				relevant++
			}
			if r.OutreachCandidate == 1 {
				outreach++
			}
			if r.RelevanceScore != nil {
				relScores = append(relScores, *r.RelevanceScore)
			}
			if r.ConfidenceScore != nil {
				confScores = append(confScores, *r.ConfidenceScore)
			}
		}
		intentCounts := countBy(entry.results, func(r normalizedResult) string { return r.IntentType })
		contentCounts := countBy(entry.results, func(r normalizedResult) string { return r.ContentType })

		stats = append(stats, repository.GoogleRunDomainStat{
			Domain:                 d,
			ResultCount:            len(entry.results),
			RelevantCount:          relevant,
			OutreachCandidateCount: outreach,
			AvgRelevanceScore:      average(relScores),
			AvgConfidenceScore:     average(confScores),
			TopIntentTypesJSON:     toTopArray(intentCounts, 5),
			TopContentTypesJSON:    toTopArray(contentCounts, 5),
		})
	}
	return stats
}

// toAnalyzedResult converts a normalizedResult back into an AnalyzedResult
// for the report builder, mapping the fields used by BuildGoogleReport.
func toAnalyzedResult(r normalizedResult) AnalyzedResult {
	ar := AnalyzedResult{
		Title:                r.Title,
		Snippet:              r.Snippet,
		URL:                  r.URL,
		DisplayURL:           r.DisplayURL,
		Rank:                 r.Rank,
		Domain:               r.Domain,
		CanonicalURL:         r.CanonicalURL,
		Sources:              r.Sources,
		ContentHash:          r.ContentHash,
		Summary:              r.Summary,
		ContentType:          r.ContentType,
		IntentType:           r.IntentType,
		RelevanceFit:         r.RelevanceFit,
		OpportunityTypes:     r.OpportunityTypes,
		KeegoingFitReasons:   r.FitReasons,
		Disqualifiers:        r.Disqualifiers,
		ActionRecommendation: r.ActionRecommendation,
		OutreachCandidate:    r.OutreachCandidate,
		MentionedProducts:    r.MentionedProducts,
	}
	if r.RelevanceScore != nil {
		ar.RelevanceScore = *r.RelevanceScore
	}
	if r.ConfidenceScore != nil {
		ar.ConfidenceScore = *r.ConfidenceScore
	}
	return ar
}

// RunGoogleScoutPipeline executes the full Google search pipeline, mirroring
// runGoogleScoutPipeline() from runner.js.
//
// Step labels match Express: "Loading google root keywords", "Expanding google queries",
// "Searching google", "Fetching google content", "Analyzing google results",
// "Deduplicating google results", "Extracting mentioned products", "Persisting google results".
func RunGoogleScoutPipeline(
	ctx context.Context,
	repo *repository.Repository,
	aiSvc *ai.Service,
	project domain.Project,
	projectID string,
	runID int64,
	provider SearchProvider,
	filters ...ResultFilter,
) (RunResult, error) {
	updateStep := func(label string) {
		_ = repo.UpdateScoutStep(ctx, runID, label)
	}

	// 1. Load root keywords
	updateStep("Loading google root keywords")
	queries, err := repo.EnabledQueries(ctx, projectID, "google")
	if err != nil {
		return RunResult{RunID: runID}, fmt.Errorf("google runner: load queries: %w", err)
	}

	// Deduplicate root keywords (query_url is the root keyword for google)
	seen := make(map[string]bool)
	var rootKeywords []string
	for _, q := range queries {
		kw := strings.TrimSpace(q.QueryURL)
		if kw != "" && !seen[kw] {
			seen[kw] = true
			rootKeywords = append(rootKeywords, kw)
		}
	}

	if len(rootKeywords) == 0 {
		if err := repo.CompleteScoutRun(ctx, runID, 0, 0, nil); err != nil {
			return RunResult{RunID: runID}, err
		}
		return RunResult{RunID: runID, PostsChecked: 0, PostsFound: 0}, nil
	}

	// 2. Expand queries
	updateStep("Expanding google queries")
	type expandedQuery struct {
		rootKeyword string
		query       string
	}
	var expanded []expandedQuery
	for _, kw := range rootKeywords {
		for _, q := range ExpandQueries(kw) {
			expanded = append(expanded, expandedQuery{rootKeyword: kw, query: q})
		}
	}
	if len(expanded) == 0 {
		if err := repo.CompleteScoutRun(ctx, runID, 0, 0, nil); err != nil {
			return RunResult{RunID: runID}, err
		}
		return RunResult{RunID: runID, PostsChecked: 0, PostsFound: 0}, nil
	}

	// 3. Search: group by rootKeyword, call searchBatch per keyword
	updateStep("Searching google")

	// Group queries by rootKeyword
	queriesByRoot := make(map[string][]string)
	var rootOrder []string
	for _, e := range expanded {
		if _, exists := queriesByRoot[e.rootKeyword]; !exists {
			rootOrder = append(rootOrder, e.rootKeyword)
		}
		queriesByRoot[e.rootKeyword] = append(queriesByRoot[e.rootKeyword], e.query)
	}

	type rawResult struct {
		rootKeyword string
		query       string
		result      SearchResult
	}
	var searchedResults []rawResult

	for i, kw := range rootOrder {
		if ctx.Err() != nil {
			_ = repo.FailScoutRun(ctx, runID, "Cancelled")
			return RunResult{RunID: runID}, nil
		}
		kwQueries := queriesByRoot[kw]
		updateStep(fmt.Sprintf("Searching google (%d/%d) - %s", i+1, len(rootOrder), kw))
		objective := deriveObjective(project, kw)
		results, err := provider.SearchBatch(ctx, kwQueries, SearchOptions{Objective: objective, Num: 10})
		if err != nil {
			return RunResult{RunID: runID}, fmt.Errorf("google runner: search batch: %w", err)
		}
		for _, res := range results {
			searchedResults = append(searchedResults, rawResult{rootKeyword: kw, query: kw, result: res})
		}
	}
	postsChecked := int64(len(searchedResults))

	// 4. Fetch content + analyze
	updateStep("Fetching google content")
	var analyzed []normalizedResult
	for i, item := range searchedResults {
		if ctx.Err() != nil {
			_ = repo.FailScoutRun(ctx, runID, "Cancelled")
			return RunResult{RunID: runID}, nil
		}
		updateStep(fmt.Sprintf("Analyzing google results (%d/%d)", i+1, len(searchedResults)))
		fetched, err := FetchPageContent(ctx, item.result.URL)
		if err != nil {
			fetched = FetchResult{}
		}
		kctx := AnalysisContext{
			RootKeyword: item.rootKeyword,
		}
		ar := AnalyzeResult(item.result, fetched, kctx)
		analyzed = append(analyzed, buildNormalizedResult(item.result, item.rootKeyword, item.query, fetched, ar))
	}

	// 5. Deduplicate
	updateStep("Deduplicating google results")
	// Convert to AnalyzedResult slice for dedupe, then convert back
	arSlice := make([]AnalyzedResult, len(analyzed))
	for i, r := range analyzed {
		arSlice[i] = toAnalyzedResult(r)
		// carry through extra fields
		arSlice[i].Title = r.Title
		arSlice[i].Snippet = r.Snippet
		arSlice[i].URL = r.URL
		arSlice[i].DisplayURL = r.DisplayURL
		arSlice[i].Rank = r.Rank
	}
	deduped := DedupeSearchResults(arSlice)

	// Rebuild normalizedResults from deduped, keeping extra non-AnalyzedResult fields
	// by building a lookup from URL -> normalizedResult
	origByURL := make(map[string]normalizedResult)
	for _, r := range analyzed {
		if r.URL != "" {
			origByURL[r.URL] = r
		}
	}
	dedupedNorm := make([]normalizedResult, len(deduped))
	for i, ar := range deduped {
		base, ok := origByURL[ar.URL]
		if !ok {
			// fallback: construct from AnalyzedResult
			base = normalizedResult{
				MentionedProducts: []string{},
			}
		}
		// Merge domain from dedupe if available
		domain := base.Domain
		if domain == "" {
			domain = parseDomain(ar.CanonicalURL)
		}
		if domain == "" {
			domain = parseDomain(ar.URL)
		}
		base.Domain = domain
		base.CanonicalURL = ar.CanonicalURL
		dedupedNorm[i] = base
	}

	// 6. Extract mentioned products
	updateStep("Extracting mentioned products")
	dedupedArSlice := make([]AnalyzedResult, len(dedupedNorm))
	for i, r := range dedupedNorm {
		dedupedArSlice[i] = toAnalyzedResult(r)
	}
	var withProducts []AnalyzedResult
	if aiSvc != nil {
		var extractErr error
		withProducts, extractErr = ExtractProductsFromResults(ctx, dedupedArSlice, aiSvc)
		if extractErr != nil {
			log.Printf("[google-runner] product extraction failed, continuing without: %v", extractErr)
			withProducts = dedupedArSlice
			for i := range withProducts {
				if withProducts[i].MentionedProducts == nil {
					withProducts[i].MentionedProducts = []string{}
				}
			}
		}
	} else {
		withProducts = dedupedArSlice
		for i := range withProducts {
			if withProducts[i].MentionedProducts == nil {
				withProducts[i].MentionedProducts = []string{}
			}
		}
	}

	// Merge MentionedProducts back into dedupedNorm
	for i, ar := range withProducts {
		if i < len(dedupedNorm) {
			dedupedNorm[i].MentionedProducts = ar.MentionedProducts
		}
	}

	// 6b. Partition results into visible and all (for persisting filtered rows).
	var filterFn ResultFilter
	if len(filters) > 0 {
		filterFn = filters[0]
	}
	visibleNorm := make([]normalizedResult, 0, len(dedupedNorm))
	allNorm := make([]normalizedResult, 0, len(dedupedNorm))
	for _, item := range dedupedNorm {
		decision := domain.FilterDecision{
			State:          domain.FilterStateVisible,
			Source:         domain.FilterSourceNone,
			Reasons:        []string{},
			SourceIdentity: domain.SourceIdentity{"domain": item.Domain, "canonical_url": strings.ToLower(item.CanonicalURL)},
		}
		if filterFn != nil {
			fi := FilterInput{
				Title:        item.Title,
				Snippet:      item.Snippet,
				URL:          item.URL,
				Domain:       item.Domain,
				CanonicalURL: item.CanonicalURL,
				PageText:     item.PageText,
			}
			var ferr error
			decision, ferr = filterFn(ctx, projectID, fi)
			if ferr != nil {
				log.Printf("[google-runner] filter error for %q (project=%s): %v; treating as visible", item.URL, projectID, ferr)
				decision = domain.FilterDecision{
					State:          domain.FilterStateVisible,
					Source:         domain.FilterSourceNone,
					Reasons:        []string{},
					Warning:        ferr.Error(),
					SourceIdentity: domain.SourceIdentity{"domain": item.Domain, "canonical_url": strings.ToLower(item.CanonicalURL)},
				}
			}
		}
		item.FilterDecision = decision
		allNorm = append(allNorm, item)
		if decision.State != domain.FilterStateFiltered {
			visibleNorm = append(visibleNorm, item)
		}
	}
	dedupedNorm = visibleNorm

	// 7. Build keyword summaries, domain stats, report
	kwSummaries := buildKeywordSummaries(dedupedNorm, rootKeywords)
	domainStats := buildDomainStats(dedupedNorm)

	finalAR := make([]AnalyzedResult, len(dedupedNorm))
	for i, r := range dedupedNorm {
		finalAR[i] = toAnalyzedResult(r)
	}
	report := BuildGoogleReport(ReportInput{
		AnalyzedResults:  finalAR,
		KeywordSummaries: nil, // use DomainStats path
		DomainStats:      nil,
		RootKeywords:     rootKeywords,
	})

	// Build keyword summaries as KeywordSummary for report builder
	kwForReport := make([]KeywordSummary, len(kwSummaries))
	for i, ks := range kwSummaries {
		kwForReport[i] = KeywordSummary{
			RootKeyword:       ks.RootKeyword,
			TotalResults:      float64(ks.TotalResults),
			AvgRelevanceScore: 0,
		}
		if ks.AvgRelevanceScore != nil {
			kwForReport[i].AvgRelevanceScore = *ks.AvgRelevanceScore
		}
	}
	dsForReport := make([]DomainStat, len(domainStats))
	for i, ds := range domainStats {
		dsForReport[i] = DomainStat{
			Domain:      ds.Domain,
			ResultCount: float64(ds.ResultCount),
		}
	}
	report = BuildGoogleReport(ReportInput{
		AnalyzedResults:  finalAR,
		KeywordSummaries: kwForReport,
		DomainStats:      dsForReport,
		RootKeywords:     rootKeywords,
	})

	// 8. Persist — persist allNorm (includes filtered rows), but report/summary from dedupedNorm (visible).
	updateStep("Persisting google results")

	repoResults := make([]repository.GoogleRunResult, len(allNorm))
	for i, r := range allNorm {
		repoResults[i] = repository.GoogleRunResult{
			RootKeyword:          r.RootKeyword,
			Query:                r.Query,
			Title:                r.Title,
			URL:                  r.URL,
			DisplayURL:           r.DisplayURL,
			Snippet:              r.Snippet,
			Rank:                 r.Rank,
			ResultType:           r.ResultType,
			Domain:               r.Domain,
			PublishedAt:          r.PublishedAt,
			Author:               r.Author,
			PageText:             r.PageText,
			ContentType:          r.ContentType,
			IntentType:           r.IntentType,
			RelevanceFit:         r.RelevanceFit,
			RelevanceScore:       r.RelevanceScore,
			ConfidenceScore:      r.ConfidenceScore,
			OpportunityTypes:     r.OpportunityTypes,
			FitReasons:           r.FitReasons,
			Disqualifiers:        r.Disqualifiers,
			Summary:              r.Summary,
			ActionRecommendation: r.ActionRecommendation,
			OutreachCandidate:    r.OutreachCandidate,
			CanonicalURL:         r.CanonicalURL,
			ContentHash:          r.ContentHash,
			MentionedProducts:    r.MentionedProducts,
			Sources:              r.Sources,
			FilterDecision:       r.FilterDecision,
		}
	}

	execSummary := map[string]interface{}{
		"top_insights":                      report.TopInsights,
		"strongest_recurring_pain_themes":   report.StrongestRecurringPainThemes,
		"interesting_phrases_for_messaging": report.InterestingPhrases,
		"mentioned_products":                report.MentionedProducts,
	}

	repoReport := repository.GoogleRunReport{
		ExecutiveSummaryJSON: execSummary,
		KeywordSummaryJSON:   report.BestKeywordsForSEO,
		OpportunitiesJSON:    report.RecommendedOpportunities,
		RisksJSON:            report.ReportRisks,
		NextActionsJSON:      report.RecommendedNextActions,
	}

	if err := repo.ReplaceGoogleRunData(ctx, runID, projectID, repoResults, kwSummaries, domainStats, repoReport); err != nil {
		return RunResult{RunID: runID}, fmt.Errorf("google runner: persist: %w", err)
	}

	postsFound := int64(len(dedupedNorm))
	if err := repo.CompleteScoutRun(ctx, runID, postsChecked, postsFound, nil); err != nil {
		return RunResult{RunID: runID}, err
	}

	return RunResult{RunID: runID, PostsChecked: postsChecked, PostsFound: postsFound}, nil
}
