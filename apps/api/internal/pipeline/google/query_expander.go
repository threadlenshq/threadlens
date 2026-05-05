package google

import (
	"strings"
)

// QUERY_TEMPLATES mirrors QUERY_TEMPLATES from query-expander.js.
var QUERY_TEMPLATES = []string{
	"{k}",
	"how to {k}",
	"why is it hard to {k}",
	"best tool {k}",
	"site:reddit.com {k}",
	"site:reddit.com {k} problem",
	"site:reddit.com {k} frustrated",
	"site:news.ycombinator.com {k}",
	"site:news.ycombinator.com {k} problem",
	"site:stackexchange.com {k}",
	"site:github.com/discussions {k}",
}

// normalizeKeyword trims and collapses whitespace.
func normalizeKeyword(keyword string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(keyword)), " ")
}

// ExpandQueries expands a root keyword into a set of deterministic search
// queries, mirroring expandQueries() from query-expander.js.
func ExpandQueries(rootKeyword string) []string {
	keyword := normalizeKeyword(rootKeyword)
	if keyword == "" {
		return []string{}
	}

	seen := make(map[string]bool)
	var ordered []string
	for _, tmpl := range QUERY_TEMPLATES {
		q := strings.ReplaceAll(tmpl, "{k}", keyword)
		if !seen[q] {
			seen[q] = true
			ordered = append(ordered, q)
		}
	}
	return ordered
}
