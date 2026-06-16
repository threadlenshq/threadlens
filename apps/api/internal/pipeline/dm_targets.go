package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

const (
	dmTargetLimit     = 3
	dmCandidateLimit  = 5
	dmSignalMaxLength = 180
)

type DMCandidateOutcome string

const (
	DMCandidateExclude  DMCandidateOutcome = "exclude"
	DMCandidatePenalize DMCandidateOutcome = "penalize"
	DMCandidateAllow    DMCandidateOutcome = "allow"

	dmStrongPenalty   = 2.0
	dmModeratePenalty = 1.0
)

// DMCandidateProfile contains only candidate signals already available in the
// open-core scout flow or bounded comment and reply fetchers. Missing optional
// metadata is neutral and must not create a bot, spam, or quality penalty.
type DMCandidateProfile struct {
	Platform         string
	Username         string
	DisplayName      string
	Bio              string
	Text             string
	Engagement       int
	CreatedAt        string
	ProfileCreatedAt *time.Time
	SourcePriority   int
}

type DMCandidateFilterResult struct {
	Outcome DMCandidateOutcome
	Penalty float64
	Reason  string
}

type DMCandidateFilter struct{}

func (DMCandidateFilter) Evaluate(profile DMCandidateProfile) DMCandidateFilterResult {
	username := normalizeDMUsername(profile.Username)
	text := strings.TrimSpace(profile.Text)
	if username == "" {
		return DMCandidateFilterResult{Outcome: DMCandidateExclude, Reason: "empty username"}
	}
	if username == "[deleted]" || username == "deleted" || username == "[removed]" || username == "removed" {
		return DMCandidateFilterResult{Outcome: DMCandidateExclude, Reason: "deleted or removed identity"}
	}
	if text == "" {
		return DMCandidateFilterResult{Outcome: DMCandidateExclude, Reason: "empty candidate text"}
	}
	if usernameLooksAutomated(username) {
		return DMCandidateFilterResult{Outcome: DMCandidateExclude, Reason: "automation username"}
	}
	combined := strings.ToLower(strings.Join([]string{text, profile.DisplayName, profile.Bio}, "\n"))
	if containsAny(combined, []string{"i am a bot", "automated bot", "auto-generated", "rss feed", "mirror bot", "this account posts automatically"}) {
		return DMCandidateFilterResult{Outcome: DMCandidateExclude, Reason: "automation self-identification"}
	}
	if looksLikePromotionalSpam(combined) {
		return DMCandidateFilterResult{Outcome: DMCandidateExclude, Reason: "promotional spam"}
	}

	penalty := 0.0
	reasons := []string{}
	if profile.ProfileCreatedAt != nil && time.Since(*profile.ProfileCreatedAt) >= 0 && time.Since(*profile.ProfileCreatedAt) <= 48*time.Hour && !containsIntentLanguage(text) {
		penalty += dmStrongPenalty
		reasons = append(reasons, "very new account without intent language")
	}
	profileText := strings.ToLower(strings.TrimSpace(profile.DisplayName + "\n" + profile.Bio))
	if profileText != "" && containsAny(profileText, []string{"automated", "promo", "promotion", "airdrop", "giveaway"}) {
		penalty += dmModeratePenalty
		reasons = append(reasons, "low-confidence profile signal")
	}
	if mostlyRepeatedLinksHashtagsMentions(text) {
		penalty += dmModeratePenalty
		reasons = append(reasons, "boilerplate link or tag-heavy text")
	}
	if penalty > 0 {
		return DMCandidateFilterResult{Outcome: DMCandidatePenalize, Penalty: penalty, Reason: strings.Join(reasons, "; ")}
	}
	return DMCandidateFilterResult{Outcome: DMCandidateAllow}
}

var nonAlnumRE = regexp.MustCompile(`[^a-z0-9]+`)

func usernameLooksAutomated(username string) bool {
	lower := strings.ToLower(username)
	compact := nonAlnumRE.ReplaceAllString(lower, "")
	punctNormalized := nonAlnumRE.ReplaceAllString(lower, "_")
	if strings.Contains(lower, "automoderator") || strings.Contains(compact, "automoderator") {
		return true
	}
	for _, token := range []string{"autoposter", "auto_poster", "rssfeed", "rss_feed", "newsbot", "supportbot", "replybot"} {
		if strings.Contains(lower, token) || strings.Contains(compact, strings.ReplaceAll(token, "_", "")) {
			return true
		}
	}
	return strings.HasSuffix(lower, "bot") || strings.HasSuffix(punctNormalized, "_bot") || strings.Contains(punctNormalized, "bot_")
}

func looksLikePromotionalSpam(text string) bool {
	return containsAny(text, []string{"referral link farming", "promo code", "dm me for paid promotion", "paid promotion"}) ||
		(containsAny(text, []string{"crypto", "airdrop", "giveaway"}) && containsAny(text, []string{"claim now", "referral", "free tokens", "limited time"}))
}

func containsIntentLanguage(text string) bool {
	lower := strings.ToLower(text)
	return containsAny(lower, []string{"i need", "i'm struggling", "i am struggling", "i want", "i'm looking", "i am looking", "can someone", "does anyone", "recommend", "any suggestions", "tool", "workflow", "problem"})
}

func mostlyRepeatedLinksHashtagsMentions(text string) bool {
	fields := strings.Fields(text)
	if len(fields) < 4 {
		return false
	}
	special := 0
	substantive := 0
	seenLinks := map[string]int{}
	for _, field := range fields {
		lower := strings.ToLower(strings.Trim(field, ".,;:!?()[]{}\"'"))
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
			special++
			seenLinks[lower]++
			continue
		}
		if strings.HasPrefix(lower, "#") || strings.HasPrefix(lower, "@") {
			special++
			continue
		}
		if len(lower) >= 3 {
			substantive++
		}
	}
	for _, count := range seenLinks {
		if count >= 2 && substantive > 0 {
			return true
		}
	}
	return substantive > 0 && float64(special)/float64(special+substantive) >= 0.6
}

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

// RedditProfileFetcher fetches Reddit user profile signals for DM target enrichment.
type RedditProfileFetcher interface {
	FetchRedditProfile(ctx context.Context, username string) (*RedditProfile, error)
}

// RedditProfileFetcherFunc adapts FetchRedditProfile for injection.
type RedditProfileFetcherFunc func(ctx context.Context, username string) (*RedditProfile, error)

func (f RedditProfileFetcherFunc) FetchRedditProfile(ctx context.Context, username string) (*RedditProfile, error) {
	return f(ctx, username)
}

// DMTargetGenerator builds deterministic outreach targets after social scout storage.
type DMTargetGenerator struct {
	repo            DMTargetRepository
	reddit          RedditDMContextFetcher
	bluesky         BlueskyReplyFetcher
	profileFetcher  RedditProfileFetcher
	profileCache    map[string]*RedditProfile
	profileCacheMu  sync.Mutex
}

func NewDMTargetGenerator(repo DMTargetRepository, reddit RedditDMContextFetcher, bluesky BlueskyReplyFetcher, profile RedditProfileFetcher) *DMTargetGenerator {
	return &DMTargetGenerator{repo: repo, reddit: reddit, bluesky: bluesky, profileFetcher: profile}
}

type dmCandidate struct {
	profile      DMCandidateProfile
	filter       DMCandidateFilterResult
	intentScore  float64
	profileData  *RedditProfile
	profileScore float64
}

// fetchProfileWithCache retrieves a RedditProfile for username, using an
// in-memory per-run cache to deduplicate fetches across candidates and posts.
// Returns nil if the fetcher is nil, username is empty, or the fetch fails.
func (g *DMTargetGenerator) fetchProfileWithCache(ctx context.Context, username string) *RedditProfile {
	if g == nil || g.profileFetcher == nil {
		return nil
	}
	key := normalizeDMUsername(username)
	if key == "" {
		return nil
	}
	g.profileCacheMu.Lock()
	if cached, ok := g.profileCache[key]; ok {
		g.profileCacheMu.Unlock()
		return cached // may be nil if a previous fetch already failed
	}
	g.profileCacheMu.Unlock()
	profile, err := g.profileFetcher.FetchRedditProfile(ctx, username)
	if err != nil {
		log.Printf("[reddit-profile] %s fetch failed: %v", username, err)
		g.profileCacheMu.Lock()
		g.profileCache[key] = nil
		g.profileCacheMu.Unlock()
		return nil
	}
	g.profileCacheMu.Lock()
	g.profileCache[key] = profile
	g.profileCacheMu.Unlock()
	return profile
}

// Generate creates DM targets for eligible posts and returns non-fatal warning lines.
func (g *DMTargetGenerator) Generate(ctx context.Context, project domain.Project, platform string, posts []domain.Post) []string {
	if g == nil || g.repo == nil || project.Mode != "marketing" || (platform != "reddit" && platform != "bluesky") {
		return nil
	}

	g.profileCacheMu.Lock()
	g.profileCache = map[string]*RedditProfile{}
	g.profileCacheMu.Unlock()

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
		targets := g.targetsFromCandidates(ctx, post, candidates)
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
		{profile: DMCandidateProfile{
			Platform:       platform,
			Username:       post.Author,
			Text:           strings.TrimSpace(post.Title + "\n" + post.Body),
			Engagement:     postEngagement(post),
			CreatedAt:      stringPtrValue(post.CreatedAt),
			SourcePriority: 0,
		}},
	}
	var warnings []string

	switch platform {
	case "reddit":
		if g.reddit == nil {
			return candidates, warnings
		}
		contextResult, err := g.reddit.FetchRedditContext(ctx, post.URL)
		if err != nil {
			msg := "reddit context fetch failed"
			if strings.Contains(err.Error(), "429") {
				msg = "reddit context fetch failed (rate limited)"
			}
			warnings = append(warnings, fmt.Sprintf("DM targets: %s %s", post.ID, msg))
			return candidates, warnings
		}
		for i, comment := range contextResult.TopComments {
			if i >= dmCandidateLimit {
				break
			}
			candidates = append(candidates, dmCandidate{profile: DMCandidateProfile{Platform: platform, Username: comment.Author, Text: comment.Body, Engagement: comment.Score, SourcePriority: 1}})
		}
	case "bluesky":
		if g.bluesky == nil {
			return candidates, warnings
		}
		postURI := post.ID
		if post.BlueskyURI != nil && strings.TrimSpace(*post.BlueskyURI) != "" {
			postURI = *post.BlueskyURI
		}
		replies, err := g.bluesky.FetchBlueskyReplies(ctx, postURI)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("DM targets: %s bluesky replies fetch failed: %s", post.ID, err.Error()))
			return candidates, warnings
		}
		for i, reply := range replies {
			if i >= dmCandidateLimit {
				break
			}
			candidates = append(candidates, dmCandidate{profile: DMCandidateProfile{Platform: platform, Username: reply.AuthorHandle, Text: reply.Text, Engagement: reply.LikeCount, CreatedAt: reply.IndexedAt, SourcePriority: 1}})
		}
	}
	return candidates, warnings
}

func (g *DMTargetGenerator) targetsFromCandidates(ctx context.Context, post domain.Post, candidates []dmCandidate) []domain.DMTargetInsert {
	filter := DMCandidateFilter{}
	bestByUser := map[string]dmCandidate{}
	for _, candidate := range candidates {
		candidate.profile.Username = cleanUsername(candidate.profile.Username)
		candidate.profile.Text = strings.TrimSpace(candidate.profile.Text)
		candidate.filter = filter.Evaluate(candidate.profile)
		if candidate.filter.Outcome == DMCandidateExclude {
			continue
		}
		base := post.FinalScore
		if candidate.profile.SourcePriority > 0 {
			base = post.FinalScore - 1
		}
		candidate.intentScore = scoreDMCandidate(base, candidate.profile, candidate.filter.Penalty)

		// Reddit profile enrichment: fetch, score, and multiplicatively adjust.
		if post.Platform == "reddit" && normalizeDMUsername(candidate.profile.Username) != "" {
			profile := g.fetchProfileWithCache(ctx, candidate.profile.Username)
			if profile != nil {
				ps := ScoreProfile(profile)
				candidate.profileScore = ps
				candidate.profileData = profile
				adjusted := (10 + ps) / 10 * candidate.intentScore
				candidate.intentScore = clampDMScore(adjusted)
			}
		}

		key := normalizeDMUsername(candidate.profile.Username)
		if existing, ok := bestByUser[key]; !ok || candidate.intentScore > existing.intentScore || (candidate.intentScore == existing.intentScore && candidate.profile.SourcePriority < existing.profile.SourcePriority) {
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
		if ranked[i].profile.SourcePriority != ranked[j].profile.SourcePriority {
			return ranked[i].profile.SourcePriority < ranked[j].profile.SourcePriority
		}
		return strings.ToLower(ranked[i].profile.Username) < strings.ToLower(ranked[j].profile.Username)
	})

	limit := dmTargetLimit
	if len(ranked) < limit {
		limit = len(ranked)
	}
	targets := make([]domain.DMTargetInsert, 0, limit)
	for _, candidate := range ranked[:limit] {
		insert := domain.DMTargetInsert{
			Username:    candidate.profile.Username,
			IntentScore: candidate.intentScore,
			Signal:      truncateDMText(candidate.profile.Text, dmSignalMaxLength),
			Context:     fmt.Sprintf("%s showed relevant pain or buying intent around this %s discussion.", candidate.profile.Username, post.Platform),
			Approach:    "Open with the specific pain they described and ask whether solving that workflow is still a priority.",
			DMStatus:    "new",
		}
		if candidate.profileData != nil {
			ps := candidate.profileScore
			insert.ProfileScore = &ps
			if signalsJSON, err := json.Marshal(candidate.profileData); err == nil {
				signalsStr := string(signalsJSON)
				insert.ProfileSignals = &signalsStr
			} else {
				log.Printf("[reddit-profile] %s JSON marshal failed: %v", candidate.profile.Username, err)
			}
		}
		targets = append(targets, insert)
	}
	return targets
}

func scoreDMCandidate(base float64, profile DMCandidateProfile, penalty float64) float64 {
	score := clampDMScore(base)
	lower := strings.ToLower(profile.Text)
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
	if profile.Engagement > 0 {
		score += math.Min(0.5, float64(profile.Engagement)/20.0)
	}
	score -= penalty
	return clampDMScore(score)
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
