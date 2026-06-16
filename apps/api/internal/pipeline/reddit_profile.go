package pipeline

import (
	"net/url"
	"strings"
)

// RedditProfile holds deterministic profile signals fetched from Reddit's
// about.json and submitted.json endpoints for a single user.
type RedditProfile struct {
	Username         string  `json:"username"`
	AccountAgeDays   int     `json:"account_age_days"`
	CommentKarma     int     `json:"comment_karma"`
	PostKarma        int     `json:"post_karma"`
	HasVerifiedEmail bool    `json:"has_verified_email"`
	IsGold           bool    `json:"is_gold"`
	IsNSFW           bool    `json:"is_nsfw"`
	SelfPromoRatio   float64 `json:"self_promo_ratio"`
	SubredditCount   int     `json:"subreddit_count"`
	ProfileScore     float64 `json:"profile_score"`
}

// selfPromoDomains is the blocklist of domains that indicate self-promotion
// when they appear in a post's domain field or URL.
var selfPromoDomains = []string{
	"youtube.com",
	"youtu.be",
	"twitch.tv",
	"discord.gg",
	"discord.com",
	"patreon.com",
	"ko-fi.com",
	"buymeacoffee.com",
	"gumroad.com",
	"producthunt.com",
	"medium.com",
	"substack.com",
	"twitter.com",
	"x.com",
}

// selfPromoKeywords are keyword phrases in a post title that indicate
// self-promotion. All comparisons are case-insensitive.
var selfPromoKeywords = []string{
	"check out my",
	"my plugin",
	"my tool",
	"download my",
	"i made a",
	"i built a",
	"i created a",
	"my new ",
	"my latest",
	"my app",
	"i'm working on",
	"i am working on",
	"launched my",
	"shipped my",
}

// DetectSelfPromotion returns true if the post title or domain matches any
// self-promotion heuristic. Both conditions are checked; the caller increments
// a counter once per post regardless of how many conditions match.
func DetectSelfPromotion(title, domain string) bool {
	lowerTitle := strings.ToLower(title)
	lowerDomain := strings.ToLower(domain)

	// Extract hostname from domain if it looks like a full URL.
	host := lowerDomain
	if strings.HasPrefix(lowerDomain, "http://") || strings.HasPrefix(lowerDomain, "https://") {
		if u, err := url.Parse(lowerDomain); err == nil && u.Hostname() != "" {
			host = u.Hostname()
		}
	}

	// Skip self-posts (domain like "self.subreddit") and empty domains.
	if host == "" || strings.HasPrefix(host, "self.") {
		// Still check title match even for self-posts.
		for _, kw := range selfPromoKeywords {
			if strings.Contains(lowerTitle, kw) {
				return true
			}
		}
		return false
	}

	// Domain match.
	for _, d := range selfPromoDomains {
		if strings.Contains(host, d) {
			return true
		}
	}

	// Title match.
	for _, kw := range selfPromoKeywords {
		if strings.Contains(lowerTitle, kw) {
			return true
		}
	}

	return false
}

// ScoreProfile applies the deterministic rule-based scorer to a RedditProfile.
// Returns 0 for a nil profile. The result is clamped to -5..+2.
func ScoreProfile(profile *RedditProfile) float64 {
	if profile == nil {
		return 0
	}
	score := 0.0

	// Account age rules.
	if profile.AccountAgeDays < 30 {
		score -= 2
	} else if profile.AccountAgeDays < 90 {
		score -= 1
	}

	// Comment karma rules.
	if profile.CommentKarma < 0 {
		score -= 2
	}

	// Zero-karma rule.
	if profile.PostKarma == 0 && profile.CommentKarma == 0 {
		score -= 1
	}

	// Self-promo ratio rules.
	if profile.SelfPromoRatio > 0.5 {
		score -= 3
	} else if profile.SelfPromoRatio > 0.25 {
		score -= 1
	}

	// Positive signals.
	if profile.HasVerifiedEmail {
		score += 1
	}
	if profile.IsGold {
		score += 1
	}

	// Single-sub promo account.
	if profile.SubredditCount == 1 && profile.SelfPromoRatio > 0 {
		score -= 1
	}

	// Clamp to -5..+2.
	if score < -5 {
		score = -5
	}
	if score > 2 {
		score = 2
	}
	return score
}
