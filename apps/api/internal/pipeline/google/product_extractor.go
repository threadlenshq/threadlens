package google

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
)

const productExtractorSystemPrompt = "You extract named software products, SaaS tools, and companies mentioned in web search results. " +
	"Return ONLY a JSON array of canonical product/company names (e.g. [\"Notion\",\"Linear\",\"Trello\"]). " +
	"Rules: exclude generic terms (app, tool, platform, system, software, service, solution), " +
	"exclude content platforms (reddit, github, youtube, medium, stackoverflow, twitter, linkedin, substack, producthunt, indiehackers, hackernews, quora, facebook, dev.to), " +
	"exclude the word \"google\", " +
	"deduplicate case-insensitively, " +
	"return at most 10 items, " +
	"return [] if nothing qualifies."

func buildProductExtractorUserMessage(r AnalyzedResult) string {
	title := strings.TrimSpace(r.Title)
	snippet := strings.TrimSpace(r.Snippet)
	pagePreview := r.Summary
	if len(pagePreview) > 800 {
		pagePreview = pagePreview[:800]
	}
	pagePreview = strings.TrimSpace(pagePreview)

	var parts []string
	if title != "" {
		parts = append(parts, "Title: "+title)
	}
	if snippet != "" {
		parts = append(parts, "Snippet: "+snippet)
	}
	if pagePreview != "" {
		parts = append(parts, "Page excerpt: "+pagePreview)
	}
	return strings.Join(parts, "\n")
}

func normalizeProducts(raw []interface{}) []string {
	seen := make(map[string]string) // lowercase -> canonical
	var order []string
	for _, item := range raw {
		name := strings.TrimSpace(strings.TrimSpace(func() string {
			switch v := item.(type) {
			case string:
				return v
			default:
				return ""
			}
		}()))
		if name == "" || len(name) < 2 {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; !exists {
			seen[key] = name
			order = append(order, key)
		}
	}
	result := make([]string, 0, len(order))
	for _, k := range order {
		result = append(result, seen[k])
	}
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

// ExtractMentionedProducts calls the AI service to extract product names from a result.
// Mirrors extractMentionedProducts() from product-extractor.js.
func ExtractMentionedProducts(ctx context.Context, r AnalyzedResult, svc *ai.Service) ([]string, error) {
	userMessage := buildProductExtractorUserMessage(r)
	if userMessage == "" {
		return []string{}, nil
	}

	raw, _, err := svc.GenerateForTask(ctx, "google_analysis", productExtractorSystemPrompt, userMessage)
	if err != nil {
		return []string{}, nil
	}

	// Extract JSON array from response
	match := regexp.MustCompile(`\[[\s\S]*\]`).FindString(raw)
	if match == "" {
		return []string{}, nil
	}

	var parsed []interface{}
	if err := json.Unmarshal([]byte(match), &parsed); err != nil {
		return []string{}, nil
	}

	return normalizeProducts(parsed), nil
}

// ExtractProductsFromResults runs product extraction on qualifying results concurrently.
// Mirrors extractProductsFromResults() from product-extractor.js.
func ExtractProductsFromResults(ctx context.Context, results []AnalyzedResult, svc *ai.Service) ([]AnalyzedResult, error) {
	const relevanceThreshold = 3.0
	const maxResults = 30
	const concurrency = 5

	// Filter candidates
	candidates := make([]AnalyzedResult, 0)
	for _, r := range results {
		if r.RelevanceScore >= relevanceThreshold {
			candidates = append(candidates, r)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].RelevanceScore > candidates[j].RelevanceScore
	})
	if len(candidates) > maxResults {
		candidates = candidates[:maxResults]
	}

	if len(candidates) == 0 {
		out := make([]AnalyzedResult, len(results))
		for i, r := range results {
			if r.MentionedProducts == nil {
				r.MentionedProducts = []string{}
			}
			out[i] = r
		}
		return out, nil
	}

	// Run batch with limited concurrency
	type extraction struct {
		key      string
		products []string
	}
	extractionResults := make([]extraction, len(candidates))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, candidate := range candidates {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, r AnalyzedResult) {
			defer wg.Done()
			defer func() { <-sem }()
			products, _ := ExtractMentionedProducts(ctx, r, svc)
			key := r.URL
			if key == "" {
				key = r.CanonicalURL
			}
			extractionResults[idx] = extraction{key: key, products: products}
		}(i, candidate)
	}
	wg.Wait()

	mentionedByURL := make(map[string][]string)
	for _, e := range extractionResults {
		if e.key != "" {
			mentionedByURL[e.key] = e.products
		}
	}

	out := make([]AnalyzedResult, len(results))
	for i, r := range results {
		key := r.URL
		if key == "" {
			key = r.CanonicalURL
		}
		if products, ok := mentionedByURL[key]; ok {
			r.MentionedProducts = products
		} else if r.MentionedProducts == nil {
			r.MentionedProducts = []string{}
		}
		out[i] = r
	}
	return out, nil
}
