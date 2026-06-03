package pipeline

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

const (
	dmTargetLimit     = 3
	dmCandidateLimit  = 5
	dmSignalMaxLength = 180
)

// DMTargetRepository is the persistence surface needed by DMTargetGenerator.
type DMTargetRepository interface {
	CountDMTargets(ctx context.Context, postID string) (int, error)
	InsertDMTargets(ctx context.Context, postID string, targets []domain.DMTargetInsert) (int64, error)
}

// RedditDMContextFetcher fetches post body and top comments for Reddit targets.
type RedditDMContextFetcher interface {
	FetchRedditContext(ctx context.Context, postURL string) (RedditContext, error)
}

// RedditDMContextFetcherFunc adapts FetchRedditContext for injection.
type RedditDMContextFetcherFunc func(ctx context.Context, postURL string) (RedditContext, error)

func (f RedditDMContextFetcherFunc) FetchRedditContext(ctx context.Context, postURL string) (RedditContext, error) {
	return f(ctx, postURL)
}

// BlueskyReply is a top-level reply candidate returned by FetchBlueskyReplies.
type BlueskyReply struct {
	AuthorHandle string
	Text         string
	LikeCount    int
	IndexedAt    string
}

// BlueskyReplyFetcher fetches top-level replies for Bluesky targets.
type BlueskyReplyFetcher interface {
	FetchBlueskyReplies(ctx context.Context, postURI string) ([]BlueskyReply, error)
}

// BlueskyReplyFetcherFunc adapts FetchBlueskyReplies for injection.
type BlueskyReplyFetcherFunc func(ctx context.Context, postURI string) ([]BlueskyReply, error)

func (f BlueskyReplyFetcherFunc) FetchBlueskyReplies(ctx context.Context, postURI string) ([]BlueskyReply, error) {
	return f(ctx, postURI)
}

// DMTargetGenerator builds deterministic outreach targets after social scout storage.
type DMTargetGenerator struct {
	repo    DMTargetRepository
	reddit  RedditDMContextFetcher
	bluesky BlueskyReplyFetcher
}

func NewDMTargetGenerator(repo DMTargetRepository, reddit RedditDMContextFetcher, bluesky BlueskyReplyFetcher) *DMTargetGenerator {
	return &DMTargetGenerator{repo: repo, reddit: reddit, bluesky: bluesky}
}

type dmCandidate struct {
	username       string
	text           string
	engagement     int
	createdAt      string
	sourcePriority int
	intentScore    float64
}

// Generate creates DM targets for eligible posts and returns non-fatal warning lines.
func (g *DMTargetGenerator) Generate(ctx context.Context, project domain.Project, platform string, posts []domain.Post) []string {
	if g == nil || g.repo == nil || project.Mode != "marketing" || (platform != "reddit" && platform != "bluesky") {
		return nil
	}

	var warnings []string
	for _, post := range posts {
		if ctx.Err() != nil {
			warnings = append(warnings, "DM targets: generation stopped because scout context was cancelled")
			break
		}
		if !dmPostEligible(post) {
			continue
		}

		existing, err := g.repo.CountDMTargets(ctx, post.ID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("DM targets: %s existing-target check failed: %s", post.ID, err.Error()))
			continue
		}
		if existing > 0 {
			continue
		}

		candidates, candidateWarnings := g.buildCandidates(ctx, platform, post)
		warnings = append(warnings, candidateWarnings...)
		targets := targetsFromCandidates(post, candidates)
		if len(targets) == 0 {
			continue
		}

		if _, err := g.repo.InsertDMTargets(ctx, post.ID, targets); err != nil {
			warnings = append(warnings, fmt.Sprintf("DM targets: %s insert failed: %s", post.ID, err.Error()))
			continue
		}
	}
	return warnings
}

func dmPostEligible(post domain.Post) bool {
	if post.Platform != "reddit" && post.Platform != "bluesky" {
		return false
	}
	if post.FinalScore < 5 {
		return false
	}
	return post.FilterState == "" || post.FilterState == domain.FilterStateVisible
}

func (g *DMTargetGenerator) buildCandidates(ctx context.Context, platform string, post domain.Post) ([]dmCandidate, []string) {
	candidates := []dmCandidate{
		{
			username:       post.Author,
			text:           strings.TrimSpace(post.Title + "\n" + post.Body),
			engagement:     postEngagement(post),
			createdAt:      stringPtrValue(post.CreatedAt),
			sourcePriority: 0,
		},
	}
	var warnings []string

	switch platform {
	case "reddit":
		if g.reddit == nil {
			return candidates, warnings
		}
		contextResult, err := g.reddit.FetchRedditContext(ctx, post.URL)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("DM targets: %s reddit context fetch failed: %s", post.ID, err.Error()))
			return candidates, warnings
		}
		for i, comment := range contextResult.TopComments {
			if i >= dmCandidateLimit {
				break
			}
			candidates = append(candidates, dmCandidate{username: comment.Author, text: comment.Body, engagement: comment.Score, sourcePriority: 1})
		}
	case "bluesky":
		if g.bluesky == nil {
			return candidates, warnings
		}
		replies, err := g.bluesky.FetchBlueskyReplies(ctx, post.ID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("DM targets: %s bluesky replies fetch failed: %s", post.ID, err.Error()))
			return candidates, warnings
		}
		for i, reply := range replies {
			if i >= dmCandidateLimit {
				break
			}
			candidates = append(candidates, dmCandidate{username: reply.AuthorHandle, text: reply.Text, engagement: reply.LikeCount, createdAt: reply.IndexedAt, sourcePriority: 1})
		}
	}
	return candidates, warnings
}

func targetsFromCandidates(post domain.Post, candidates []dmCandidate) []domain.DMTargetInsert {
	bestByUser := map[string]dmCandidate{}
	for _, candidate := range candidates {
		candidate.username = cleanUsername(candidate.username)
		candidate.text = strings.TrimSpace(candidate.text)
		if !validDMCandidate(candidate) {
			continue
		}
		base := post.FinalScore
		if candidate.sourcePriority > 0 {
			base = post.FinalScore - 1
		}
		candidate.intentScore = scoreDMCandidate(base, candidate.text, candidate.engagement, candidate.createdAt)
		key := normalizeDMUsername(candidate.username)
		if existing, ok := bestByUser[key]; !ok || candidate.intentScore > existing.intentScore {
			bestByUser[key] = candidate
		}
	}

	ranked := make([]dmCandidate, 0, len(bestByUser))
	for _, candidate := range bestByUser {
		ranked = append(ranked, candidate)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].intentScore != ranked[j].intentScore {
			return ranked[i].intentScore > ranked[j].intentScore
		}
		if ranked[i].sourcePriority != ranked[j].sourcePriority {
			return ranked[i].sourcePriority < ranked[j].sourcePriority
		}
		return strings.ToLower(ranked[i].username) < strings.ToLower(ranked[j].username)
	})

	limit := dmTargetLimit
	if len(ranked) < limit {
		limit = len(ranked)
	}
	targets := make([]domain.DMTargetInsert, 0, limit)
	for _, candidate := range ranked[:limit] {
		targets = append(targets, domain.DMTargetInsert{
			Username:    candidate.username,
			IntentScore: candidate.intentScore,
			Signal:      truncateDMText(candidate.text, dmSignalMaxLength),
			Context:     fmt.Sprintf("%s showed relevant pain or buying intent around this %s discussion.", candidate.username, post.Platform),
			Approach:    "Open with the specific pain they described and ask whether solving that workflow is still a priority.",
			DMStatus:    "new",
		})
	}
	return targets
}

func validDMCandidate(candidate dmCandidate) bool {
	username := normalizeDMUsername(candidate.username)
	if username == "" || username == "[deleted]" || username == "deleted" || username == "[removed]" || username == "removed" {
		return false
	}
	if strings.Contains(username, "automoderator") || strings.HasSuffix(username, "bot") {
		return false
	}
	return strings.TrimSpace(candidate.text) != ""
}

func scoreDMCandidate(base float64, text string, engagement int, createdAt string) float64 {
	score := clampDMScore(base)
	lower := strings.ToLower(text)
	if containsAny(lower, []string{"i need", "i'm struggling", "i am struggling", "i hate", "i can't", "i cannot", "my problem"}) {
		score += 0.5
	}
	if containsAny(lower, []string{"can someone", "does anyone", "recommend", "any suggestions", "looking for"}) {
		score += 0.75
	}
	if containsAny(lower, []string{"tool", "app", "software", "service", "workflow", "automation"}) {
		score += 0.5
	}
	if containsAny(lower, []string{"frustrated", "annoying", "pain", "broken", "waste of time", "workaround"}) {
		score += 0.5
	}
	if engagement > 0 {
		score += math.Min(0.5, float64(engagement)/20.0)
	}
	if bonus := recencyBonus(createdAt); bonus > 0 {
		score += bonus
	}
	return clampDMScore(score)
}

func recencyBonus(createdAt string) float64 {
	if strings.TrimSpace(createdAt) == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return 0
	}
	age := time.Since(parsed)
	if age < 0 {
		age = 0
	}
	if age <= 7*24*time.Hour {
		return 0.25
	}
	if age <= 30*24*time.Hour {
		return 0.1
	}
	return 0
}

func postEngagement(post domain.Post) int {
	if post.Platform == "reddit" && post.RedditScore != nil {
		return int(*post.RedditScore)
	}
	if post.Platform == "bluesky" && post.LikeCount != nil {
		return int(*post.LikeCount)
	}
	return 0
}

func clampDMScore(score float64) float64 {
	if score < 1 {
		return 1
	}
	if score > 10 {
		return 10
	}
	return math.Round(score*100) / 100
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func cleanUsername(username string) string {
	trimmed := strings.TrimSpace(username)
	trimmed = strings.TrimPrefix(trimmed, "u/")
	trimmed = strings.TrimPrefix(trimmed, "@")
	return trimmed
}

func normalizeDMUsername(username string) string {
	return strings.ToLower(cleanUsername(username))
}

func truncateDMText(text string, maxRunes int) string {
	trimmed := strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return string(runes[:maxRunes])
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
