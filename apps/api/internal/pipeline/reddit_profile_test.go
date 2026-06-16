package pipeline

import (
	"testing"
)

func TestScoreProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile *RedditProfile
		want    float64
	}{
		{
			name:    "nil profile returns 0",
			profile: nil,
			want:    0,
		},
		{
			name: "new account with negative comment karma",
			profile: &RedditProfile{
				AccountAgeDays: 5,
				CommentKarma:   -10,
				PostKarma:      0,
			},
			want: -4, // -2 (age<30) + -2 (neg karma) = -4; zero-karma rule doesn't fire because CommentKarma is -10
		},
		{
			name: "old account with negative comment karma",
			profile: &RedditProfile{
				AccountAgeDays: 1500,
				CommentKarma:   -10,
				PostKarma:      100,
			},
			want: -2, // -2 (neg karma only)
		},
		{
			name: "old account with verified email and gold",
			profile: &RedditProfile{
				AccountAgeDays:   1500,
				CommentKarma:     100,
				PostKarma:        100,
				HasVerifiedEmail: true,
				IsGold:           true,
			},
			want: 2, // +1 (verified) + +1 (gold) = +2
		},
		{
			name: "heavy self-promoter single subreddit",
			profile: &RedditProfile{
				AccountAgeDays: 500,
				CommentKarma:   50,
				PostKarma:      50,
				SelfPromoRatio: 0.6,
				SubredditCount: 1,
			},
			want: -4, // -3 (promo>0.5) + -1 (single-sub promo) = -4
		},
		{
			name: "boundary SelfPromoRatio 0.5 triggers both penalties",
			profile: &RedditProfile{
				AccountAgeDays: 500,
				CommentKarma:   50,
				PostKarma:      50,
				SelfPromoRatio: 0.5,
				SubredditCount: 1,
			},
			want: -2, // -1 (promo>0.25) + -1 (single-sub with ratio>0) = -2
		},
		{
			name: "boundary AccountAgeDays 30",
			profile: &RedditProfile{
				AccountAgeDays: 30,
				CommentKarma:   100,
				PostKarma:      100,
			},
			want: -1, // 30 is NOT < 30, but IS < 90 → -1
		},
		{
			name: "boundary AccountAgeDays 90",
			profile: &RedditProfile{
				AccountAgeDays: 90,
				CommentKarma:   100,
				PostKarma:      100,
			},
			want: 0, // 90 is NOT < 90 → 0
		},
		{
			name: "neutral profile no rules fire",
			profile: &RedditProfile{
				AccountAgeDays: 1000,
				CommentKarma:   100,
				PostKarma:      10,
				SelfPromoRatio: 0.0,
				SubredditCount: 5,
			},
			want: 0,
		},
		{
			name: "clamp at -5 floor",
			profile: &RedditProfile{
				AccountAgeDays: 5,
				CommentKarma:   -10,
				PostKarma:      0,
				SelfPromoRatio: 0.6,
				SubredditCount: 1,
			},
			want: -5, // -2 + -2 + -1 + -3 + -1 = -9, clamped to -5
		},
		{
			name: "clamp at +2 ceiling",
			profile: &RedditProfile{
				AccountAgeDays:   1000,
				CommentKarma:     100,
				PostKarma:        100,
				HasVerifiedEmail: true,
				IsGold:           true,
			},
			want: 2, // +1 + +1 = +2
		},
		{
			name: "moderate self-promo ratio between 0.25 and 0.5",
			profile: &RedditProfile{
				AccountAgeDays: 500,
				CommentKarma:   50,
				PostKarma:      50,
				SelfPromoRatio: 0.3,
				SubredditCount: 5,
			},
			want: -1, // -1 (promo>0.25 and <=0.5)
		},
		{
			name: "zero karma both post and comment",
			profile: &RedditProfile{
				AccountAgeDays: 500,
				CommentKarma:   0,
				PostKarma:      0,
			},
			want: -1, // -1 (both zero)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ScoreProfile(tc.profile)
			if got != tc.want {
				t.Errorf("ScoreProfile() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetectSelfPromotion(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		domain string
		want   bool
	}{
		{
			name:   "domain match youtube",
			title:  "My new post",
			domain: "youtube.com",
			want:   true,
		},
		{
			name:   "domain match via full URL twitter",
			title:  "Great tool",
			domain: "https://twitter.com/foo",
			want:   true,
		},
		{
			name:   "title keyword match check out my",
			title:  "check out my plugin",
			domain: "self.dev",
			want:   true,
		},
		{
			name:   "title keyword match with empty domain",
			title:  "I made a tool for X",
			domain: "",
			want:   true,
		},
		{
			name:   "normal post no match",
			title:  "Looking for recommendations",
			domain: "self.dev",
			want:   false,
		},
		{
			name:   "self-post with neutral title",
			title:  "Need help with a problem",
			domain: "self.dev",
			want:   false,
		},
		{
			name:   "domain and title both match returns true once",
			title:  "check out my video",
			domain: "youtube.com",
			want:   true,
		},
		{
			name:   "patreon domain match",
			title:  "Support my work",
			domain: "patreon.com",
			want:   true,
		},
		{
			name:   "i built a keyword match",
			title:  "I built a new CLI tool",
			domain: "github.com",
			want:   true,
		},
		{
			name:   "launched my keyword match",
			title:  "Launched my SaaS today",
			domain: "example.com",
			want:   true,
		},
		{
			name:   "substack with @ path match",
			title:  "Weekly newsletter",
			domain: "substack.com/@user",
			want:   true,
		},
		{
			name:   "case insensitive title match",
			title:  "CHECK OUT MY new project",
			domain: "self.test",
			want:   true,
		},
		{
			name:   "empty title and empty domain",
			title:  "",
			domain: "",
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectSelfPromotion(tc.title, tc.domain)
			if got != tc.want {
				t.Errorf("DetectSelfPromotion(%q, %q) = %v, want %v", tc.title, tc.domain, got, tc.want)
			}
		})
	}
}

func TestDetectSelfPromotionIdempotent(t *testing.T) {
	first := DetectSelfPromotion("check out my video", "youtube.com")
	second := DetectSelfPromotion("check out my video", "youtube.com")
	if first != second {
		t.Errorf("DetectSelfPromotion is not idempotent: first=%v second=%v", first, second)
	}
}
