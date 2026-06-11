package google

import (
	"strconv"
	"testing"
)

func TestBuildGoogleReportAllSections(t *testing.T) {
	report := BuildGoogleReport(ReportInput{})

	if report.TopInsights == nil {
		t.Error("expected top_insights to be non-nil")
	}
	if report.BestKeywordsForSEO == nil {
		t.Error("expected best_keywords_for_seo to be non-nil")
	}
	if report.StrongestRecurringPainThemes == nil {
		t.Error("expected strongest_recurring_pain_themes to be non-nil")
	}
	if report.BestContentGapOpportunities == nil {
		t.Error("expected best_content_gap_opportunities to be non-nil")
	}
	if report.PotentialCompetitorTargets == nil {
		t.Error("expected potential_competitor_comparison_targets to be non-nil")
	}
	if report.PotentialOutreachCandidates == nil {
		t.Error("expected potential_outreach_candidates to be non-nil")
	}
	if report.LowValueKeywordsToDrop == nil {
		t.Error("expected low_value_keywords_to_drop to be non-nil")
	}
	if report.InterestingPhrases == nil {
		t.Error("expected interesting_phrases_for_messaging to be non-nil")
	}
}

func TestBuildGoogleReportRankingBehavior(t *testing.T) {
	report := BuildGoogleReport(ReportInput{
		AnalyzedResults: []AnalyzedResult{
			{
				Title: "High-value fit", Summary: "Best direct fit summary",
				RelevanceScore: 9.6, RelevanceFit: "direct_fit",
				OpportunityTypes: []string{"content_gap"}, OutreachCandidate: 1,
				URL: "https://alpha.example.com/high-fit",
			},
			{
				Title: "Second fit", Summary: "Secondary summary",
				RelevanceScore: 7.1, RelevanceFit: "direct_fit",
				OpportunityTypes: []string{"competitor_weakness"}, OutreachCandidate: 1,
				URL: "https://beta.example.com/second-fit",
			},
		},
		KeywordSummaries: []KeywordSummary{
			{Keyword: "customer onboarding checklist", AvgRelevance: 8.9, RepeatCount: 4},
			{Query: "legacy keyword", OpportunityScore: 2, HitCount: 0},
		},
		DomainStats: []DomainStat{
			{Domain: "alpha.example.com", ResultCount: 6},
			{Domain: "beta.example.com", Count: 2},
		},
	})

	if len(report.TopInsights) == 0 || report.TopInsights[0] != "Best direct fit summary" {
		t.Errorf("expected TopInsights[0]='Best direct fit summary', got %v", report.TopInsights)
	}
	if len(report.BestKeywordsForSEO) == 0 || report.BestKeywordsForSEO[0].Keyword != "customer onboarding checklist" {
		t.Errorf("unexpected best_keywords_for_seo[0]: %+v", report.BestKeywordsForSEO)
	}
	if report.BestKeywordsForSEO[0].OpportunityScore != 8.9 {
		t.Errorf("expected opportunityScore=8.9, got %v", report.BestKeywordsForSEO[0].OpportunityScore)
	}
	if len(report.PotentialCompetitorTargets) == 0 || report.PotentialCompetitorTargets[0] != "alpha.example.com" {
		t.Errorf("unexpected potential_competitor_comparison_targets: %v", report.PotentialCompetitorTargets)
	}
	if len(report.PotentialOutreachCandidates) == 0 || report.PotentialOutreachCandidates[0].Title != "High-value fit" {
		t.Errorf("unexpected outreach candidates: %v", report.PotentialOutreachCandidates)
	}

	hasLegacy := false
	for _, k := range report.LowValueKeywordsToDrop {
		if k == "legacy keyword" {
			hasLegacy = true
		}
	}
	if !hasLegacy {
		t.Errorf("expected 'legacy keyword' in low_value_keywords_to_drop, got %v", report.LowValueKeywordsToDrop)
	}
}

func TestBuildGoogleReportContentGapRanking(t *testing.T) {
	report := BuildGoogleReport(ReportInput{
		AnalyzedResults: []AnalyzedResult{
			{Title: "Low score first", Summary: "Low score summary", RelevanceScore: 2.1, RelevanceFit: "direct_fit", OpportunityTypes: []string{"content_gap"}},
			{Title: "Top score appears later", Summary: "Top score summary", RelevanceScore: 9.8, RelevanceFit: "direct_fit", OpportunityTypes: []string{"content_gap"}},
			{Title: "Middle score", Summary: "Middle score summary", RelevanceScore: 6.4, RelevanceFit: "direct_fit", OpportunityTypes: []string{"content_gap"}},
		},
	})

	expected := []string{"Top score summary", "Middle score summary", "Low score summary"}
	if len(report.BestContentGapOpportunities) != 3 {
		t.Fatalf("expected 3 best_content_gap_opportunities, got %d", len(report.BestContentGapOpportunities))
	}
	for i, s := range expected {
		if report.BestContentGapOpportunities[i] != s {
			t.Errorf("[%d] expected %q, got %q", i, s, report.BestContentGapOpportunities[i])
		}
	}
}

func TestBuildGoogleReportPainThemes(t *testing.T) {
	report := BuildGoogleReport(ReportInput{
		AnalyzedResults: []AnalyzedResult{
			{Summary: "Developers get stuck when they return to a cold codebase and lose momentum"},
			{Summary: "Abandon your side project? Here is how to restart without frustration"},
			{Summary: "Why coders struggle to resume coding project work after a break"},
			{Summary: "A relaxing workflow for picking up context switch momentum"},
		},
	})

	values := make(map[string]bool)
	for _, pt := range report.StrongestRecurringPainThemes {
		values[pt.Value] = true
	}

	// Should NOT contain fit reason labels
	if values["matches root keyword context"] || values["clear pain signal language"] {
		t.Errorf("pain themes should not contain fit reason labels")
	}

	// Should contain actual pain terms
	if !values["stuck"] && !values["momentum"] {
		t.Errorf("expected stuck or momentum in pain themes, got %v", report.StrongestRecurringPainThemes)
	}

	if len(report.StrongestRecurringPainThemes) > 0 {
		pt := report.StrongestRecurringPainThemes[0]
		if pt.Value == "" {
			t.Error("expected non-empty value")
		}
		if pt.Category != "problem" && pt.Category != "workflow" {
			t.Errorf("unexpected category: %q", pt.Category)
		}
		if pt.Count <= 0 {
			t.Errorf("expected count > 0")
		}
	}
}

func TestBuildGoogleReportStripsScaffoldingFromPhrases(t *testing.T) {
	report := BuildGoogleReport(ReportInput{
		AnalyzedResults: []AnalyzedResult{
			{Summary: "Section Title: Developer Momentum > Content: working with unfamiliar codebases is painful"},
			{Summary: "Section Title: The Handoff > Content: unfamiliar patterns trip newcomers on unfamiliar projects"},
			{Summary: "Section Title: Re-entry > Content: unfamiliar context makes the work painful again"},
		},
	})

	phrases := make(map[string]bool)
	for _, p := range report.InterestingPhrases {
		phrases[p.Phrase] = true
	}
	if phrases["section"] || phrases["title"] || phrases["content"] {
		t.Errorf("phrases should not contain scaffolding words, got %v", report.InterestingPhrases)
	}
	if !phrases["unfamiliar"] {
		t.Errorf("expected 'unfamiliar' in phrases, got %v", report.InterestingPhrases)
	}
}

func TestBuildGoogleReportExcludesRootKeywordTokens(t *testing.T) {
	report := BuildGoogleReport(ReportInput{
		AnalyzedResults: []AnalyzedResult{
			{Summary: "resume coding project tips for returning developers handling unfamiliar repositories"},
			{Summary: "resume coding project checklist for developers working on unfamiliar repositories"},
		},
		RootKeywords: []string{"resume coding project"},
	})

	phrases := make(map[string]bool)
	for _, p := range report.InterestingPhrases {
		phrases[p.Phrase] = true
	}
	if phrases["resume"] || phrases["coding"] || phrases["project"] {
		t.Errorf("phrases should not contain root keyword tokens, got %v", report.InterestingPhrases)
	}
	if !phrases["unfamiliar"] {
		t.Errorf("expected 'unfamiliar' in phrases, got %v", report.InterestingPhrases)
	}
}

func TestBuildGoogleReportNextActions(t *testing.T) {
	report := BuildGoogleReport(ReportInput{
		AnalyzedResults: []AnalyzedResult{
			{
				Title: "Great outreach target", Summary: "Great outreach target summary",
				RelevanceScore: 9.5, RelevanceFit: "direct_fit", OutreachCandidate: 1,
				URL: "https://example.com/outreach", OpportunityTypes: []string{},
			},
			{
				Title: "Content gap target", Summary: "Content gap target summary",
				RelevanceScore: 9.2, RelevanceFit: "direct_fit", OutreachCandidate: 0,
				URL: "https://example.com/gap", OpportunityTypes: []string{"content_gap"},
			},
		},
	})

	var engageAction *NextAction
	var gapAction *NextAction
	for i := range report.RecommendedNextActions {
		a := &report.RecommendedNextActions[i]
		if a.Action == "Engage author" {
			engageAction = a
		}
		if a.Action == "Create content filling gap" {
			gapAction = a
		}
	}

	if engageAction == nil {
		t.Fatal("expected 'Engage author' action")
	}
	if engageAction.Target != "Great outreach target" {
		t.Errorf("expected target='Great outreach target', got %q", engageAction.Target)
	}
	if engageAction.URL != "https://example.com/outreach" {
		t.Errorf("unexpected URL: %q", engageAction.URL)
	}

	if gapAction == nil {
		t.Fatal("expected 'Create content filling gap' action")
	}
	if gapAction.Target != "Content gap target" {
		t.Errorf("expected target='Content gap target', got %q", gapAction.Target)
	}
}

func TestBuildGoogleReportOpportunitiesSplit(t *testing.T) {
	report := BuildGoogleReport(ReportInput{
		AnalyzedResults: []AnalyzedResult{
			{
				Title: "Gap piece", Summary: "Gap piece summary",
				RelevanceScore: 9, RelevanceFit: "direct_fit",
				URL: "https://example.com/gap", OpportunityTypes: []string{"content_gap"},
			},
		},
		DomainStats: []DomainStat{
			{Domain: "acme-saas.com", ResultCount: 35},
			{Domain: "rival-tool.io", ResultCount: 16},
		},
	})

	var gaps, competitors []Opportunity
	for _, o := range report.RecommendedOpportunities {
		if o.Kind == "content_gap" {
			gaps = append(gaps, o)
		}
		if o.Kind == "competitor" {
			competitors = append(competitors, o)
		}
	}

	if len(gaps) != 1 {
		t.Errorf("expected 1 content_gap, got %d", len(gaps))
	}
	if gaps[0].Label != "Gap piece" || gaps[0].URL != "https://example.com/gap" {
		t.Errorf("unexpected gap: %+v", gaps[0])
	}
	labels := []string{}
	for _, c := range competitors {
		labels = append(labels, c.Label)
	}
	if len(labels) < 2 || labels[0] != "acme-saas.com" || labels[1] != "rival-tool.io" {
		t.Errorf("unexpected competitor labels: %v", labels)
	}
	if competitors[0].Count != 35 {
		t.Errorf("expected count=35, got %d", competitors[0].Count)
	}
}

func TestBuildGoogleReportContentPlatformsAsTopSource(t *testing.T) {
	report := BuildGoogleReport(ReportInput{
		DomainStats: []DomainStat{
			{Domain: "reddit.com", ResultCount: 40},
			{Domain: "medium.com", ResultCount: 20},
			{Domain: "acme-saas.com", ResultCount: 10},
		},
	})

	topSourceLabels := map[string]bool{}
	competitorLabels := map[string]bool{}
	for _, o := range report.RecommendedOpportunities {
		if o.Kind == "top_source" {
			topSourceLabels[o.Label] = true
		}
		if o.Kind == "competitor" {
			competitorLabels[o.Label] = true
		}
	}

	if !topSourceLabels["reddit.com"] {
		t.Errorf("expected reddit.com in top_source labels")
	}
	if !topSourceLabels["medium.com"] {
		t.Errorf("expected medium.com in top_source labels")
	}
	if !competitorLabels["acme-saas.com"] {
		t.Errorf("expected acme-saas.com in competitor labels")
	}
	if competitorLabels["reddit.com"] || competitorLabels["medium.com"] {
		t.Errorf("content platforms should not be competitors")
	}
}

func TestBuildGoogleReportRisks(t *testing.T) {
	thinWeak := make([]AnalyzedResult, 8)
	for i := range thinWeak {
		thinWeak[i] = AnalyzedResult{Summary: "s", RelevanceFit: "weak_fit", OutreachCandidate: 0}
	}
	strong := []AnalyzedResult{
		{Summary: "Long, meaningful summary about a problem, this is quite long you know really, extremely, incredibly, massively.", RelevanceFit: "direct_fit", OutreachCandidate: 1},
		{Summary: "Another long meaningful summary about the workflow, this is quite long you know really, extremely, incredibly, massively.", RelevanceFit: "direct_fit", OutreachCandidate: 1},
	}
	report := BuildGoogleReport(ReportInput{
		AnalyzedResults:  append(thinWeak, strong...),
		KeywordSummaries: []KeywordSummary{{RootKeyword: "dud kw", AvgRelevanceScore: 0, TotalResults: 0}},
	})

	labels := map[string]bool{}
	for _, r := range report.ReportRisks {
		labels[r.Label] = true
	}

	hasWeakFit := false
	for k := range labels {
		if len(k) > 8 && k[len(k)-8:] == "weak fit" || (len(k) > 8 && func() bool {
			for _, s := range []string{"weak fit"} {
				if len(k) >= len(s) {
					for i := 0; i <= len(k)-len(s); i++ {
						if k[i:i+len(s)] == s {
							return true
						}
					}
				}
			}
			return false
		}()) {
			hasWeakFit = true
		}
	}
	if !hasWeakFit {
		t.Errorf("expected a risk label with 'weak fit', got %v", report.ReportRisks)
	}

	hasThinContent := false
	for _, r := range report.ReportRisks {
		if containsStr(r.Label, "thin content") {
			hasThinContent = true
		}
	}
	if !hasThinContent {
		t.Errorf("expected thin content risk, got %v", report.ReportRisks)
	}

	hasDudKw := false
	for _, r := range report.ReportRisks {
		if containsStr(r.Label, "Drop weak keyword: dud kw") {
			hasDudKw = true
		}
	}
	if !hasDudKw {
		t.Errorf("expected 'Drop weak keyword: dud kw' risk, got %v", report.ReportRisks)
	}
}

func TestExtractPosterReddit(t *testing.T) {
	cases := []struct {
		url      string
		expected *string
	}{
		{"https://www.reddit.com/r/sideproject/comments/abc/title/", strptr("r/sideproject")},
		{"https://reddit.com/r/programming/", strptr("r/programming")},
		{"https://old.reddit.com/r/startups/comments/xyz/", strptr("r/startups")},
		{"https://www.reddit.com/user/johndoe/posts/", strptr("u/johndoe")},
		{"https://reddit.com/u/janedoe/", strptr("u/janedoe")},
		{"https://reddit.com/", nil},
		{"https://reddit.com", nil},
	}
	for _, c := range cases {
		got := ExtractPoster(c.url)
		if c.expected == nil {
			if got != nil {
				t.Errorf("url=%q: expected nil, got %q", c.url, *got)
			}
		} else if got == nil || *got != *c.expected {
			gotStr := "<nil>"
			if got != nil {
				gotStr = *got
			}
			t.Errorf("url=%q: expected %q, got %q", c.url, *c.expected, gotStr)
		}
	}
}

func TestExtractPosterGithub(t *testing.T) {
	cases := []struct {
		url      string
		expected *string
	}{
		{"https://github.com/facebook/react", strptr("facebook")},
		{"https://github.com/vercel/next.js/issues/123", strptr("vercel")},
		{"https://github.com/orgs/acme/teams", nil},
		{"https://github.com/explore", nil},
		{"https://docs.github.com/en/actions", nil},
		{"https://gist.github.com/user/abc123", nil},
	}
	for _, c := range cases {
		got := ExtractPoster(c.url)
		if c.expected == nil {
			if got != nil {
				t.Errorf("url=%q: expected nil, got %q", c.url, *got)
			}
		} else if got == nil || *got != *c.expected {
			gotStr := "<nil>"
			if got != nil {
				gotStr = *got
			}
			t.Errorf("url=%q: expected %q, got %q", c.url, *c.expected, gotStr)
		}
	}
}

func TestExtractPosterMedium(t *testing.T) {
	cases := []struct {
		url      string
		expected string
	}{
		{"https://medium.com/@johndoe/my-article-123", "@johndoe"},
		{"https://medium.com/better-programming/some-article", "better-programming"},
		{"https://dev.to/johndoe/my-post-title-abc", "johndoe"},
	}
	for _, c := range cases {
		got := ExtractPoster(c.url)
		if got == nil || *got != c.expected {
			gotStr := "<nil>"
			if got != nil {
				gotStr = *got
			}
			t.Errorf("url=%q: expected %q, got %q", c.url, c.expected, gotStr)
		}
	}
}

func TestExtractPosterYouTube(t *testing.T) {
	cases := []struct {
		url      string
		expected *string
	}{
		{"https://www.youtube.com/@mychannel/videos", strptr("@mychannel")},
		{"https://www.youtube.com/c/channelname", strptr("channelname")},
		{"https://www.youtube.com/user/oldname", strptr("oldname")},
		{"https://www.youtube.com/watch?v=abc123", nil},
	}
	for _, c := range cases {
		got := ExtractPoster(c.url)
		if c.expected == nil {
			if got != nil {
				t.Errorf("url=%q: expected nil, got %q", c.url, *got)
			}
		} else if got == nil || *got != *c.expected {
			gotStr := "<nil>"
			if got != nil {
				gotStr = *got
			}
			t.Errorf("url=%q: expected %q, got %q", c.url, *c.expected, gotStr)
		}
	}
}

func TestExtractPosterInvalid(t *testing.T) {
	if ExtractPoster("") != nil {
		t.Error("expected nil for empty string")
	}
	if ExtractPoster("not-a-url") != nil {
		t.Error("expected nil for not-a-url")
	}
}

func TestBuildMentionedProductsEmpty(t *testing.T) {
	result := BuildMentionedProducts([]AnalyzedResult{
		{URL: "https://a.com", MentionedProducts: []string{}},
		{URL: "https://b.com"},
	})
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestBuildMentionedProductsAggregates(t *testing.T) {
	results := []AnalyzedResult{
		{URL: "https://a.com", MentionedProducts: []string{"Notion", "Linear"}},
		{URL: "https://b.com", MentionedProducts: []string{"Notion", "Trello"}},
		{URL: "https://c.com", MentionedProducts: []string{"Notion"}},
	}
	products := BuildMentionedProducts(results)

	if len(products) == 0 {
		t.Fatal("expected products")
	}
	if products[0].Name != "Notion" || products[0].Count != 3 || products[0].Kind != "mentioned_product" {
		t.Errorf("unexpected first product: %+v", products[0])
	}
}

func TestBuildMentionedProductsDeduplicatesCaseInsensitive(t *testing.T) {
	results := []AnalyzedResult{
		{URL: "https://a.com", MentionedProducts: []string{"Notion", "notion", "NOTION"}},
		{URL: "https://b.com", MentionedProducts: []string{"Notion"}},
	}
	products := BuildMentionedProducts(results)

	notionCount := 0
	for _, p := range products {
		if p.Name == "Notion" {
			notionCount++
		}
	}
	if notionCount != 1 {
		t.Errorf("expected 1 Notion entry, got %d", notionCount)
	}
	// Count should be 4 (all case variations)
	for _, p := range products {
		if p.Name == "Notion" {
			if p.Count != 4 {
				t.Errorf("expected count=4, got %d", p.Count)
			}
		}
	}
}

func TestBuildMentionedProductsCapsAt10(t *testing.T) {
	names := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"}
	results := make([]AnalyzedResult, len(names))
	for i, name := range names {
		results[i] = AnalyzedResult{URL: "https://result-" + strconv.Itoa(i) + ".com", MentionedProducts: []string{name}}
	}
	products := BuildMentionedProducts(results)
	if len(products) != 10 {
		t.Errorf("expected 10, got %d", len(products))
	}
}

func strptr(s string) *string { return &s }
