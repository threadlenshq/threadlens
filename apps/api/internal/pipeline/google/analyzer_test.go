package google

import (
	"strings"
	"testing"
)

func TestAnalyzeResultHighRelevance(t *testing.T) {
	result := SearchResult{
		Title:    "Getting back into coding after months away is painful",
		Snippet:  "I keep losing context when I return to my side project and spend hours re-onboarding every week.",
		PageText: "Need a lightweight way to resume coding project flow, rebuild momentum, and avoid re-learning my own code.",
		URL:      "https://www.reddit.com/r/learnprogramming/comments/abc123/resume_coding_project/",
	}
	analyzed := AnalyzeResult(result, FetchResult{}, AnalysisContext{})

	if analyzed.RelevanceScore <= 8 {
		t.Errorf("expected relevance_score > 8, got %v", analyzed.RelevanceScore)
	}
	if analyzed.RelevanceFit != "direct_fit" {
		t.Errorf("expected direct_fit, got %q", analyzed.RelevanceFit)
	}
	hasContentGap := false
	for _, ot := range analyzed.OpportunityTypes {
		if ot == "content_gap" {
			hasContentGap = true
		}
	}
	if !hasContentGap {
		t.Errorf("expected content_gap in opportunity_types, got %v", analyzed.OpportunityTypes)
	}
}

func TestAnalyzeResultClassifiesDimensions(t *testing.T) {
	result := SearchResult{
		Title:    "Guide: personal workflow to restart abandoned coding projects",
		Snippet:  "Step-by-step checklist for developers to restart in-progress apps quickly.",
		PageText: "Template and checklist with exact actions to resume work after context switching.",
		URL:      "https://dev.to/someone/restart-coding-projects",
	}
	analyzed := AnalyzeResult(result, FetchResult{}, AnalysisContext{})

	if analyzed.ContentType != "guide" {
		t.Errorf("expected guide, got %q", analyzed.ContentType)
	}
	if analyzed.IntentType != "solution" {
		t.Errorf("expected solution, got %q", analyzed.IntentType)
	}
	if analyzed.ConfidenceScore <= 0 {
		t.Errorf("expected confidence_score > 0, got %v", analyzed.ConfidenceScore)
	}
	if analyzed.ActionRecommendation == "" {
		t.Errorf("expected non-empty action_recommendation")
	}
}

func TestAnalyzeResultThreeArgContract(t *testing.T) {
	result := SearchResult{
		Title:   "A thread about coding focus",
		Snippet: "General discussion without explicit restart keywords.",
		URL:     "https://example.com/thread",
	}
	fetched := FetchResult{
		PageText: "Developers feel stuck and lose momentum after context switch. A checklist can help resume coding project flow.",
	}
	kctx := AnalysisContext{RootKeyword: "resume coding project"}
	analyzed := AnalyzeResult(result, fetched, kctx)

	if analyzed.ScoreBreakdown.Workflow <= 0 {
		t.Errorf("expected workflow > 0")
	}
	if analyzed.ScoreBreakdown.Actionability <= 0 {
		t.Errorf("expected actionability > 0")
	}
	hasRootKeyword := false
	for _, r := range analyzed.KeegoingFitReasons {
		if r == "matches root keyword context" {
			hasRootKeyword = true
		}
	}
	if !hasRootKeyword {
		t.Errorf("expected 'matches root keyword context' in keepgoing_fit_reasons, got %v", analyzed.KeegoingFitReasons)
	}
	if analyzed.RelevanceScore <= 0 {
		t.Errorf("expected relevance_score > 0")
	}
}

func TestAnalyzeResultProductGap(t *testing.T) {
	result := SearchResult{
		Title:   "Painful developer workflow when returning to old projects",
		Snippet: "Hard and frustrating to restart without the right support.",
		URL:     "https://example.com/problem-post",
	}
	fetched := FetchResult{
		PageText: "I need a tool with demo and pricing details to solve this stuck workflow.",
	}
	analyzed := AnalyzeResult(result, fetched, AnalysisContext{})

	hasProductGap := false
	for _, ot := range analyzed.OpportunityTypes {
		if ot == "product_gap" {
			hasProductGap = true
		}
	}
	if !hasProductGap {
		t.Errorf("expected product_gap in opportunity_types, got %v", analyzed.OpportunityTypes)
	}
}

func TestAnalyzeResultThresholdsAndDisqualifiers(t *testing.T) {
	adjacent := AnalyzeResult(SearchResult{
		Title:   "Developers stuck with workflow issues",
		Snippet: "Guide to improve process when context switching.",
		URL:     "https://example.com/workflow-guide",
	}, FetchResult{}, AnalysisContext{})

	weak := AnalyzeResult(SearchResult{
		Title:   "Software engineer salary trends",
		Snippet: "Job hiring discussion and compensation outlook.",
		URL:     "https://technews.example.com/article",
	}, FetchResult{}, AnalysisContext{})

	if adjacent.RelevanceScore < 5 {
		t.Errorf("adjacent: expected >= 5, got %v", adjacent.RelevanceScore)
	}
	if adjacent.RelevanceScore >= 8 {
		t.Errorf("adjacent: expected < 8, got %v", adjacent.RelevanceScore)
	}
	if adjacent.RelevanceFit != "adjacent_fit" {
		t.Errorf("expected adjacent_fit, got %q", adjacent.RelevanceFit)
	}
	if weak.RelevanceFit != "weak_fit" {
		t.Errorf("expected weak_fit, got %q", weak.RelevanceFit)
	}

	hasCareer := false
	for _, d := range weak.Disqualifiers {
		if strings.Contains(d, "career") {
			hasCareer = true
		}
	}
	if !hasCareer {
		t.Errorf("expected 'career-only topic' disqualifier, got %v", weak.Disqualifiers)
	}

	hasNews := false
	for _, d := range weak.Disqualifiers {
		if strings.Contains(d, "news") {
			hasNews = true
		}
	}
	if !hasNews {
		t.Errorf("expected 'news-heavy source' disqualifier, got %v", weak.Disqualifiers)
	}

	if weak.OutreachCandidate != 0 {
		t.Errorf("expected outreach_candidate == 0")
	}
}
