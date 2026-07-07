package services

import (
	"fmt"
	"net/url"
	"strings"
)

// AdaptQueryURL mechanically converts a general-query text into the query_url
// shape each platform expects. No AI rewriting: the user's text is used verbatim,
// only structurally adapted (Reddit gets a site-wide search URL; the others take
// the raw string/keyword).
func AdaptQueryURL(platform, queryText string) (string, error) {
	text := strings.TrimSpace(queryText)
	if text == "" {
		return "", fmt.Errorf("query text is required")
	}
	switch platform {
	case "reddit":
		// Reddit's JSON search endpoint is the /search.json PATH (the .json goes on
		// the path, NOT on a query-param value). Match the codebase's canonical
		// site-wide default from query_ai.go so materialized rows look identical to
		// AI-suggested ones: q + sort=new + t=month + limit=100, in that order.
		return "https://www.reddit.com/search.json?q=" + url.QueryEscape(text) + "&sort=new&t=month&limit=100", nil
	case "google", "bluesky", "hackernews":
		return text, nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}
