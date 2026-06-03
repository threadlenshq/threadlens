package pipeline

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

// ---------------------------------------------------------------------------
// Interfaces
// ---------------------------------------------------------------------------

// DMTargetRepo is the repository contract needed by DMTargetGenerator.
type DMTargetRepo interface {
	ListEligibleDMPosts(ctx context.Context, projectID string) ([]domain.Post, error)
	InsertDMTarget(ctx context.Context, postID string, t domain.DMTargetInsert) (domain.DMTarget, error)
	ListExistingDMTargets(ctx context.Context, postID string) ([]domain.DMTarget, error)
}

// RedditDMContextFetcher fetches comment authors for a Reddit post URL.
type RedditDMContextFetcher interface {
	FetchCommentAuthors(ctx context.Context, postURL string) ([]string, error)
}

// RedditDMContextFetcherFunc is a function adapter for RedditDMContextFetcher.
type RedditDMContextFetcherFunc func(ctx context.Context, postURL string) ([]string, error)

func (f RedditDMContextFetcherFunc) FetchCommentAuthors(ctx context.Context, postURL string) ([]string, error) {
	return f(ctx, postURL)
}

// BlueskyReply represents a single reply on Bluesky (reserved for future use).
type BlueskyReply struct {
	Author string
}

// BlueskyReplyFetcher fetches reply authors for a Bluesky URI.
type BlueskyReplyFetcher interface {
	FetchReplyAuthors(ctx context.Context, uri string) ([]string, error)
}

// BlueskyReplyFetcherFunc is a function adapter for BlueskyReplyFetcher.
type BlueskyReplyFetcherFunc func(ctx context.Context, uri string) ([]string, error)

func (f BlueskyReplyFetcherFunc) FetchReplyAuthors(ctx context.Context, uri string) ([]string, error) {
	return f(ctx, uri)
}

// ---------------------------------------------------------------------------
// Generator
// ---------------------------------------------------------------------------

const dmTargetsPerPost = 3

// DMTargetGenerator generates DM targets for eligible posts in a marketing project.
type DMTargetGenerator struct {
	repo           DMTargetRepo
	redditFetcher  RedditDMContextFetcher
	blueskyFetcher BlueskyReplyFetcher
}

// NewDMTargetGenerator creates a new DMTargetGenerator.
// All three arguments must be non-nil.
func NewDMTargetGenerator(
	repo DMTargetRepo,
	redditFetcher RedditDMContextFetcher,
	blueskyFetcher BlueskyReplyFetcher,
) *DMTargetGenerator {
	if repo == nil {
		panic("dm target generator: repo must not be nil")
	}
	if redditFetcher == nil {
		panic("dm target generator: redditFetcher must not be nil")
	}
	if blueskyFetcher == nil {
		panic("dm target generator: blueskyFetcher must not be nil")
	}
	return &DMTargetGenerator{
		repo:           repo,
		redditFetcher:  redditFetcher,
		blueskyFetcher: blueskyFetcher,
	}
}

// Generate runs the DM target generation for the given project.
// It returns non-fatal warnings for per-post failures and only returns an
// error for fatal top-level failures.
func (g *DMTargetGenerator) Generate(ctx context.Context, project domain.Project) ([]string, error) {
	if project.Mode != "marketing" {
		return nil, nil
	}

	posts, err := g.repo.ListEligibleDMPosts(ctx, project.ID)
	if err != nil {
		return nil, fmt.Errorf("list eligible dm posts: %w", err)
	}

	var warnings []string

	for _, post := range posts {
		// Skip posts that already have DM targets in the repo.
		existing, err := g.repo.ListExistingDMTargets(ctx, post.ID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("post %s: list existing targets: %v", post.ID, err))
			continue
		}
		if len(existing) > 0 {
			continue
		}

		// Collect candidates: author + commenters/reply authors.
		candidates, fetchWarning := g.collectCandidates(ctx, post)
		if fetchWarning != "" {
			warnings = append(warnings, fetchWarning)
		}

		// Rank deterministically and take top N.
		ranked := rankCandidates(candidates, post)
		if len(ranked) > dmTargetsPerPost {
			ranked = ranked[:dmTargetsPerPost]
		}

		// Insert each target; continue on failure.
		for _, c := range ranked {
			insert := domain.DMTargetInsert{
				Username:    c.username,
				IntentScore: c.intentScore,
				Signal:      c.signal,
				DMStatus:    "pending",
			}
			_, ierr := g.repo.InsertDMTarget(ctx, post.ID, insert)
			if ierr != nil {
				warnings = append(warnings, fmt.Sprintf("post %s: insert target %s: %v", post.ID, c.username, ierr))
				continue
			}
		}
	}

	return warnings, nil
}

// ---------------------------------------------------------------------------
// Candidate ranking
// ---------------------------------------------------------------------------

type candidate struct {
	username    string
	intentScore float64
	signal      string
}

// collectCandidates builds the candidate pool: post author + commenters/reply
// authors. Returns the candidates and an optional warning string.
func (g *DMTargetGenerator) collectCandidates(ctx context.Context, post domain.Post) ([]candidate, string) {
	var others []string
	var warning string

	switch post.Platform {
	case "reddit":
		authors, err := g.redditFetcher.FetchCommentAuthors(ctx, post.URL)
		if err != nil {
			warning = fmt.Sprintf("post %s: fetch comment authors: %v", post.ID, err)
		} else {
			others = authors
		}
	case "bluesky":
		if post.BlueskyURI == nil || *post.BlueskyURI == "" {
			warning = fmt.Sprintf("post %s: missing bluesky URI", post.ID)
			break
		}
		authors, err := g.blueskyFetcher.FetchReplyAuthors(ctx, *post.BlueskyURI)
		if err != nil {
			warning = fmt.Sprintf("post %s: fetch reply authors: %v", post.ID, err)
		} else {
			others = authors
		}
	}

	// Deduplicate: author first, then others in order, skipping duplicates.
	seen := map[string]bool{}
	var candidates []candidate

	addCandidate := func(username, signal string) {
		if username == "" || seen[username] {
			return
		}
		seen[username] = true
		score := scoreCandidate(username, signal)
		candidates = append(candidates, candidate{username: username, intentScore: score, signal: signal})
	}

	addCandidate(post.Author, "author")
	for _, u := range others {
		addCandidate(u, "commenter")
	}

	return candidates, warning
}

// directRequestPhrases are lower-cased substrings that signal purchase intent.
var directRequestPhrases = []string{
	"recommend", "looking for", "need a", "want a", "any suggestions",
	"anyone know", "help me find", "where can i",
}

// scoreCandidate computes a deterministic intent score for a candidate.
// Base: 1.0 for any candidate; bonuses for the author role and direct-request
// language signals that may appear in future text fields.
func scoreCandidate(username, signal string) float64 {
	score := 1.0
	if signal == "author" {
		score += 0.5
	}
	// Small bonus if the username itself contains direct-request language
	// (very rare, but keeps the scoring function deterministic and testable).
	lower := strings.ToLower(username)
	for _, phrase := range directRequestPhrases {
		if strings.Contains(lower, phrase) {
			score += 0.1
			break
		}
	}
	return score
}

// rankCandidates sorts candidates deterministically: descending intentScore,
// then alphabetically by username for tie-breaking.
func rankCandidates(candidates []candidate, _ domain.Post) []candidate {
	sorted := make([]candidate, len(candidates))
	copy(sorted, candidates)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].intentScore != sorted[j].intentScore {
			return sorted[i].intentScore > sorted[j].intentScore
		}
		return sorted[i].username < sorted[j].username
	})
	return sorted
}
