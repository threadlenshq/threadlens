package google

import (
	"crypto/sha1"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// canonicalizePath trims trailing slashes and lowercases.
func canonicalizePath(pathname string) string {
	trimmed := regexp.MustCompile(`/+$`).ReplaceAllString(pathname, "")
	if trimmed == "" {
		trimmed = "/"
	}
	return strings.ToLower(trimmed)
}

var trackingParamRe = regexp.MustCompile(`(?i)^(utm_|gclid$|fbclid$|mc_[ce]id$|ref$|ref_src$|source$)`)

// removeTrackingParams strips common tracking query params.
func removeTrackingParams(params string) string {
	// parse manually to avoid net/url.Values reordering
	if params == "" {
		return ""
	}
	var kept [][2]string
	for _, part := range strings.Split(params, "&") {
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		key := kv[0]
		if trackingParamRe.MatchString(key) {
			continue
		}
		val := ""
		if len(kv) == 2 {
			val = kv[1]
		}
		kept = append(kept, [2]string{key, val})
	}
	parts := make([]string, len(kept))
	for i, kv := range kept {
		if kv[1] != "" {
			parts[i] = kv[0] + "=" + kv[1]
		} else {
			parts[i] = kv[0]
		}
	}
	return strings.Join(parts, "&")
}

// CanonicalizeURL normalizes a URL for deduplication.
// Mirrors canonicalizeUrl() from dedupe.js.
func CanonicalizeURL(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}
	// We use a simplified parser that handles most cases.
	// net/url.Parse handles the heavy lifting.
	parsed, err := parseURL(raw)
	if err != nil {
		return strings.ToLower(regexp.MustCompile(`/+$`).ReplaceAllString(raw, ""))
	}

	hostname := regexp.MustCompile(`^www\.`).ReplaceAllString(strings.ToLower(parsed.host), "")
	pathname := canonicalizePath(parsed.path)

	// Sort query params
	cleanedParams := removeTrackingParams(parsed.query)
	if cleanedParams != "" {
		var pairs [][2]string
		for _, part := range strings.Split(cleanedParams, "&") {
			if part == "" {
				continue
			}
			kv := strings.SplitN(part, "=", 2)
			val := ""
			if len(kv) == 2 {
				val = kv[1]
			}
			pairs = append(pairs, [2]string{kv[0], val})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i][0] < pairs[j][0] })
		parts := make([]string, len(pairs))
		for i, kv := range pairs {
			if kv[1] != "" {
				parts[i] = kv[0] + "=" + kv[1]
			} else {
				parts[i] = kv[0]
			}
		}
		cleanedParams = strings.Join(parts, "&")
	}

	if cleanedParams != "" {
		return fmt.Sprintf("%s%s?%s", hostname, pathname, cleanedParams)
	}
	return hostname + pathname
}

// parsedURL is a minimal URL struct.
type parsedURL struct {
	host  string
	path  string
	query string
}

func parseURL(raw string) (parsedURL, error) {
	// Find scheme
	schemeEnd := strings.Index(raw, "://")
	if schemeEnd < 0 {
		return parsedURL{}, fmt.Errorf("no scheme")
	}
	rest := raw[schemeEnd+3:]
	// Strip fragment
	if idx := strings.Index(rest, "#"); idx >= 0 {
		rest = rest[:idx]
	}
	// Split host from path
	slashIdx := strings.Index(rest, "/")
	var host, pathAndQuery string
	if slashIdx < 0 {
		host = rest
		pathAndQuery = "/"
	} else {
		host = rest[:slashIdx]
		pathAndQuery = rest[slashIdx:]
	}
	// Strip port
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		// simple check: is it a port?
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
	// Split path from query
	var path, query string
	if qIdx := strings.Index(pathAndQuery, "?"); qIdx >= 0 {
		path = pathAndQuery[:qIdx]
		query = pathAndQuery[qIdx+1:]
	} else {
		path = pathAndQuery
	}
	return parsedURL{host: host, path: path, query: query}, nil
}

func normalizeTextForHash(value string) string {
	return strings.TrimSpace(strings.ToLower(regexp.MustCompile(`\s+`).ReplaceAllString(value, " ")))
}

func getCanonicalGroupKey(r *AnalyzedResult) string {
	if key := CanonicalizeURL(r.CanonicalURL); key != "" {
		return key
	}
	if r.URL != "" {
		return CanonicalizeURL(r.URL)
	}
	return CanonicalizeURL(r.DisplayURL)
}

func getContentHashKey(r *AnalyzedResult) string {
	if r.ContentHash != "" {
		return normalizeTextForHash(r.ContentHash)
	}
	title := normalizeTextForHash(r.Title)
	snippet := normalizeTextForHash(r.Snippet)
	if title == "" && snippet == "" {
		return ""
	}
	h := sha1.Sum([]byte(title + "::" + snippet))
	return fmt.Sprintf("%x", h)
}

func mergeUniqueStrings(base, incoming []string) []string {
	merged := make([]string, len(base))
	copy(merged, base)
	for _, v := range incoming {
		if v == "" {
			continue
		}
		found := false
		for _, b := range merged {
			if b == v {
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, v)
		}
	}
	return merged
}

// DedupeSearchResults deduplicates AnalyzedResults by canonical URL and content hash.
// Mirrors dedupeSearchResults() from dedupe.js.
func DedupeSearchResults(results []AnalyzedResult) []AnalyzedResult {
	type group struct {
		result        AnalyzedResult
		appearances   int
		duplicateURLs []string
	}

	groups := make(map[string]*group)
	canonicalIndex := make(map[string]string) // canonicalKey -> groupKey
	hashIndex := make(map[string]string)      // contentHashKey -> groupKey
	var groupOrder []string
	fallbackCounter := 0

	for i := range results {
		r := &results[i]
		canonicalKey := getCanonicalGroupKey(r)
		contentHashKey := getContentHashKey(r)

		var key string
		if canonicalKey != "" {
			if k, ok := canonicalIndex[canonicalKey]; ok {
				key = k
			}
		}
		if key == "" && contentHashKey != "" {
			if k, ok := hashIndex[contentHashKey]; ok {
				key = k
			}
		}
		if key == "" {
			if canonicalKey != "" {
				key = "url:" + canonicalKey
			} else if contentHashKey != "" {
				key = "hash:" + contentHashKey
			} else {
				key = fmt.Sprintf("fallback:%d", fallbackCounter)
				fallbackCounter++
			}
		}

		originalURL := strings.TrimSpace(r.URL)

		existing, exists := groups[key]
		if !exists {
			dupURLs := []string{}
			if originalURL != "" {
				dupURLs = []string{originalURL}
			}
			g := &group{
				result:        *r,
				appearances:   1,
				duplicateURLs: dupURLs,
			}
			g.result.CanonicalURL = canonicalKey
			groups[key] = g
			groupOrder = append(groupOrder, key)
		} else {
			existing.appearances++
			if originalURL != "" {
				found := false
				for _, u := range existing.duplicateURLs {
					if u == originalURL {
						found = true
						break
					}
				}
				if !found {
					existing.duplicateURLs = append(existing.duplicateURLs, originalURL)
				}
			}

			// Merge sources
			existingSources := existing.result.Sources
			if existingSources == nil {
				existingSources = []string{}
			}
			incomingSources := r.Sources
			if incomingSources == nil {
				incomingSources = []string{}
			}
			mergedSources := mergeUniqueStrings(existingSources, incomingSources)
			if len(mergedSources) > 0 {
				existing.result.Sources = mergedSources
			}

			if existing.result.CanonicalURL == "" && canonicalKey != "" {
				existing.result.CanonicalURL = canonicalKey
			}

			// Compare rank (lower is better)
			existingRank := math.Inf(1)
			if existing.result.Rank != nil {
				existingRank = *existing.result.Rank
			}
			candidateRank := math.Inf(1)
			if r.Rank != nil {
				candidateRank = *r.Rank
			}

			if candidateRank < existingRank {
				appearances := existing.appearances
				dupURLs := existing.duplicateURLs
				canonURL := existing.result.CanonicalURL
				if canonURL == "" {
					canonURL = canonicalKey
				}
				newResult := *r
				newResult.CanonicalURL = canonURL
				newResult.Sources = mergedSources
				existing.result = newResult
				existing.appearances = appearances
				existing.duplicateURLs = dupURLs
			}
		}

		if canonicalKey != "" {
			canonicalIndex[canonicalKey] = key
		}
		if contentHashKey != "" {
			hashIndex[contentHashKey] = key
		}
	}

	// Build output in group-insertion order, then sort by rank
	out := make([]AnalyzedResult, 0, len(groupOrder))
	for _, key := range groupOrder {
		g := groups[key]
		r := g.result
		r.Appearances = g.appearances
		r.DuplicateURLs = g.duplicateURLs
		out = append(out, r)
	}

	sort.SliceStable(out, func(i, j int) bool {
		ri := math.Inf(1)
		if out[i].Rank != nil {
			ri = *out[i].Rank
		}
		rj := math.Inf(1)
		if out[j].Rank != nil {
			rj = *out[j].Rank
		}
		if ri != rj {
			return ri < rj
		}
		ai := out[i].CanonicalURL
		if ai == "" {
			ai = out[i].URL
		}
		if ai == "" {
			ai = out[i].Title
		}
		bj := out[j].CanonicalURL
		if bj == "" {
			bj = out[j].URL
		}
		if bj == "" {
			bj = out[j].Title
		}
		return ai < bj
	})

	return out
}
