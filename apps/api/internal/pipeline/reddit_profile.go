package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
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

	// Check title keywords once, regardless of domain type.
	hasKeyword := false
	for _, kw := range selfPromoKeywords {
		if strings.Contains(lowerTitle, kw) {
			hasKeyword = true
			break
		}
	}

	if host == "" || strings.HasPrefix(host, "self.") {
		return hasKeyword
	}

	for _, d := range selfPromoDomains {
		if strings.Contains(host, d) {
			return true
		}
	}

	return hasKeyword
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

// redditThrottle enforces the shared minimum interval between Reddit fetches.
// It mirrors the throttle pattern in FetchRedditContext.
func redditThrottle(ctx context.Context) error {
	lastRedditFetchMu.Lock()
	wait := redditFetchMinInterval - time.Since(lastRedditFetch)
	lastRedditFetch = time.Now()
	lastRedditFetchMu.Unlock()
	if wait > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil
}

// FetchRedditProfile fetches a Reddit user's about.json and submitted.json
// endpoints and returns a populated RedditProfile. Returns nil, nil for empty
// usernames. On about.json failure, returns nil, err. On submitted.json
// failure, returns a partial profile with karma/age data but zero self-promo
// signals.
func FetchRedditProfile(ctx context.Context, username string) (*RedditProfile, error) {
	cleaned := cleanUsername(username)
	if cleaned == "" {
		return nil, nil
	}

	// Throttle before about.json.
	if err := redditThrottle(ctx); err != nil {
		return nil, err
	}

	// Fetch about.json.
	encoded := url.PathEscape(cleaned)
	aboutURL := "https://old.reddit.com/user/" + encoded + "/about.json"
	aboutBody, err := redditFetchWithRetry(ctx, aboutURL)
	if err != nil {
		log.Printf("[reddit-profile] %s about.json failed: %v", cleaned, err)
		return nil, err
	}

	// Parse about.json response.
	var aboutResp struct {
		Data struct {
			Name             string  `json:"name"`
			CreatedUTC       float64 `json:"created_utc"`
			CommentKarma     int     `json:"comment_karma"`
			LinkKarma        int     `json:"link_karma"`
			HasVerifiedEmail bool    `json:"has_verified_email"`
			IsGold           bool    `json:"is_gold"`
			Subreddit        struct {
				Over18 bool `json:"over_18"`
			} `json:"subreddit"`
		} `json:"data"`
	}
	if err := json.Unmarshal(aboutBody, &aboutResp); err != nil {
		log.Printf("[reddit-profile] %s about.json parse error: %v", cleaned, err)
		return nil, fmt.Errorf("about.json parse: %w", err)
	}

	ageDays := int(time.Since(time.Unix(int64(aboutResp.Data.CreatedUTC), 0)).Hours() / 24)
	if ageDays < 0 {
		ageDays = 0
	}
	profile := &RedditProfile{
		Username:         aboutResp.Data.Name,
		AccountAgeDays:   ageDays,
		CommentKarma:     aboutResp.Data.CommentKarma,
		PostKarma:        aboutResp.Data.LinkKarma,
		HasVerifiedEmail: aboutResp.Data.HasVerifiedEmail,
		IsGold:           aboutResp.Data.IsGold,
		IsNSFW:           aboutResp.Data.Subreddit.Over18,
	}

	// Throttle before submitted.json.
	if err := redditThrottle(ctx); err != nil {
		// Return partial profile without submitted data.
		profile.ProfileScore = ScoreProfile(profile)
		return profile, nil
	}

	// Fetch submitted.json.
	submittedURL := "https://old.reddit.com/user/" + encoded + "/submitted.json?limit=25"
	submittedBody, err := redditFetchWithRetry(ctx, submittedURL)
	if err != nil {
		log.Printf("[reddit-profile] %s submitted.json failed (partial profile): %v", cleaned, err)
		profile.ProfileScore = ScoreProfile(profile)
		return profile, nil
	}

	// Parse submitted.json response.
	var submittedResp struct {
		Data struct {
			Children []struct {
				Data struct {
					Title     string `json:"title"`
					Subreddit string `json:"subreddit"`
					URL       string `json:"url"`
					Domain    string `json:"domain"`
					IsSelf    bool   `json:"is_self"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(submittedBody, &submittedResp); err != nil {
		log.Printf("[reddit-profile] %s submitted.json parse error (partial profile): %v", cleaned, err)
		profile.ProfileScore = ScoreProfile(profile)
		return profile, nil
	}

	// Compute self-promo ratio and subreddit count.
	posts := submittedResp.Data.Children
	selfPromoCount := 0
	subreddits := map[string]bool{}
	for _, child := range posts {
		d := child.Data
		if d.Subreddit != "" {
			subreddits[strings.ToLower(d.Subreddit)] = true
		}
		if DetectSelfPromotion(d.Title, d.Domain) {
			selfPromoCount++
		}
	}
	if len(posts) > 0 {
		profile.SelfPromoRatio = float64(selfPromoCount) / float64(len(posts))
	}
	profile.SubredditCount = len(subreddits)
	profile.ProfileScore = ScoreProfile(profile)

	return profile, nil
}
