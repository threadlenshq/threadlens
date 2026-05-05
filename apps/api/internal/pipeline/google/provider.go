package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	parallelSearchEndpoint = "https://api.parallel.ai/v1/search"
	defaultObjective       = "Find web pages relevant to the given search queries."
)

// SearchOptions carries per-request options for a search batch.
type SearchOptions struct {
	Objective string
	Num       int
}

// SearchProvider abstracts over search API backends.
type SearchProvider interface {
	SearchBatch(ctx context.Context, queries []string, options SearchOptions) ([]SearchResult, error)
}

// ParallelSearchProvider is the default provider backed by parallel.ai.
type ParallelSearchProvider struct {
	APIKey string
	Client *http.Client
}

// NewParallelSearchProvider creates a ParallelSearchProvider using PARALLEL_API_KEY from env.
func NewParallelSearchProvider() *ParallelSearchProvider {
	return &ParallelSearchProvider{
		APIKey: os.Getenv("PARALLEL_API_KEY"),
		Client: &http.Client{},
	}
}

func safeStr(v string) string {
	return strings.TrimSpace(v)
}

func hostnameFromURL(rawURL string) string {
	s := safeStr(rawURL)
	if s == "" {
		return ""
	}
	// Simple extraction: strip scheme, take host part before first /
	schemeEnd := strings.Index(s, "://")
	if schemeEnd < 0 {
		return ""
	}
	rest := s[schemeEnd+3:]
	slashIdx := strings.Index(rest, "/")
	var host string
	if slashIdx < 0 {
		host = rest
	} else {
		host = rest[:slashIdx]
	}
	// Strip port
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		portPart := host[idx+1:]
		allDigits := len(portPart) > 0
		for _, c := range portPart {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			host = host[:idx]
		}
	}
	host = strings.ToLower(host)
	host = strings.TrimPrefix(host, "www.")
	return host
}

func joinExcerpts(excerpts []interface{}) string {
	var parts []string
	for _, e := range excerpts {
		if s, ok := e.(string); ok {
			s = safeStr(s)
			if s != "" {
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, " ... ")
}

func normalizeParallelItem(item map[string]interface{}, index int) SearchResult {
	rawURL := safeStr(getString(item, "url"))
	rank := float64(index + 1)

	var snippet string
	if excerpts, ok := item["excerpts"].([]interface{}); ok {
		snippet = joinExcerpts(excerpts)
	}

	var publishedAt *string
	if pd, ok := item["publish_date"].(string); ok && pd != "" {
		publishedAt = &pd
	}

	return SearchResult{
		Title:       safeStr(getString(item, "title")),
		URL:         rawURL,
		DisplayURL:  hostnameFromURL(rawURL),
		Snippet:     snippet,
		Rank:        &rank,
		ResultType:  "organic",
		PublishedAt: publishedAt,
	}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func formatParallelError(status int, details string) string {
	text := safeStr(details)
	suffix := ""
	if text != "" {
		suffix = " " + text
	}
	switch status {
	case 401, 403:
		return fmt.Sprintf("Parallel auth failed (%d): check PARALLEL_API_KEY.%s", status, suffix)
	case 402:
		return fmt.Sprintf("Parallel credits exhausted (402): top up at platform.parallel.ai.%s", suffix)
	case 429:
		return fmt.Sprintf("Parallel rate limit hit (429).%s", suffix)
	default:
		if text == "" {
			text = "Unknown error"
		}
		return fmt.Sprintf("Parallel search failed (%d): %s", status, text)
	}
}

// SearchBatch sends all queries in a single request to the Parallel.ai search API
// and returns normalised SearchResult items, mirroring ParallelSearchProvider.searchBatch().
func (p *ParallelSearchProvider) SearchBatch(ctx context.Context, queries []string, options SearchOptions) ([]SearchResult, error) {
	// Filter empty queries
	var cleaned []string
	for _, q := range queries {
		if s := safeStr(q); s != "" {
			cleaned = append(cleaned, s)
		}
	}
	if len(cleaned) == 0 {
		return []SearchResult{}, nil
	}
	if p.APIKey == "" {
		return nil, fmt.Errorf("Missing Parallel configuration (set PARALLEL_API_KEY)")
	}

	objective := options.Objective
	if safeStr(objective) == "" {
		objective = defaultObjective
	}

	body := map[string]interface{}{
		"objective":      objective,
		"search_queries": cleaned,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("parallel search: marshal request: %w", err)
	}

	client := p.Client
	if client == nil {
		client = &http.Client{}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parallelSearchEndpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("parallel search: build request: %w", err)
	}
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("parallel search: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		details, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s", formatParallelError(resp.StatusCode, string(details)))
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("parallel search: decode response: %w", err)
	}

	var results []SearchResult
	if items, ok := payload["results"].([]interface{}); ok {
		for i, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				results = append(results, normalizeParallelItem(m, i))
			}
		}
	}
	if results == nil {
		results = []SearchResult{}
	}
	return results, nil
}
