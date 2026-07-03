// Package querynorm normalizes a query_url into a stable comparison key so that
// equivalent effective fetch targets can be deduped at run time. It is a leaf
// package (no internal deps) so both services and pipeline can import it without
// creating an import cycle.
package querynorm

import (
	"net/url"
	"strings"
)

// Key returns a normalized comparison key for a platform's query_url.
func Key(platform, queryURL string) string {
	if platform == "reddit" {
		return redditKey(queryURL)
	}
	return platform + "|" + generic(queryURL)
}

func generic(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

// redditKey extracts the search term (q) and any subreddit restriction so that
// encoding / param-order / .json-suffix variants of the same Reddit search
// collapse to one key. Falls back to the generic key for unparseable input.
func redditKey(queryURL string) string {
	u, err := url.Parse(strings.TrimSpace(queryURL))
	if err != nil || u.Host == "" {
		return "reddit|" + generic(queryURL)
	}
	qs := u.Query()
	q := generic(qs.Get("q"))
	sub := ""
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "r" {
		sub = strings.ToLower(parts[1])
	}
	if sub == "" && qs.Get("restrict_sr") == "" {
		return "reddit|site|" + q
	}
	return "reddit|sub:" + sub + "|" + q
}
