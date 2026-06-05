package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline/google"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

const pipelineTimeout = 15 * time.Minute
const failRunTimeout = 5 * time.Second

// Result is the outcome of a scout run.
type Result struct {
	RunID        int64
	PostsChecked int64
	PostsFound   int64
}

// Runner manages active scout runs and their cancellation contexts.
type Runner struct {
	Repo *repository.Repository
	AI   *ai.Service
	mu   sync.Mutex
	runs map[int64]context.CancelFunc

	filterClassifier *FilterClassifier
	dmTargets        *DMTargetGenerator

	// Overridable fetchers for testing.
	fetchReddit  func(ctx context.Context, queryURLs []string, onProgress func(int, int)) ([]FetchedPost, error)
	fetchBluesky func(ctx context.Context, queries []string, onProgress func(int, int)) ([]FetchedPost, error)
	scorePosts   func(ctx context.Context, posts []ScoringPost, angles []string, rubric *string, desc *string, onProgress func(int, int)) (ScoreResult, error)
}

// NewRunner creates a new Runner.
func NewRunner(repo *repository.Repository, ai *ai.Service) *Runner {
	r := &Runner{
		Repo: repo,
		AI:   ai,
		runs: make(map[int64]context.CancelFunc),
	}
	r.filterClassifier = NewFilterClassifier(repo, nil)
	r.fetchReddit = func(ctx context.Context, queryURLs []string, onProgress func(int, int)) ([]FetchedPost, error) {
		return FetchRedditPosts(ctx, queryURLs, onProgress)
	}
	r.fetchBluesky = func(ctx context.Context, queries []string, onProgress func(int, int)) ([]FetchedPost, error) {
		return FetchBlueskyPosts(ctx, queries, onProgress)
	}
	r.scorePosts = func(ctx context.Context, posts []ScoringPost, angles []string, rubric *string, desc *string, onProgress func(int, int)) (ScoreResult, error) {
		return ScorePosts(ctx, repo, ai, posts, angles, 15, rubric, desc, onProgress)
	}
	r.dmTargets = NewDMTargetGenerator(
		repo,
		RedditDMContextFetcherFunc(FetchRedditContext),
		BlueskyReplyFetcherFunc(FetchBlueskyReplies),
	)
	return r
}

func (r *Runner) failRun(runID int64, message string) {
	failCtx, cancel := context.WithTimeout(context.Background(), failRunTimeout)
	defer cancel()
	if err := r.Repo.FailScoutRun(failCtx, runID, message); err != nil {
		log.Printf("runner: failed to mark run %d as failed: %v", runID, err)
	}
}

// ctxErrMessage returns a human-readable failure reason for a cancelled/timed-out
// context. It distinguishes between a pipeline timeout (DeadlineExceeded) and an
// explicit user-initiated cancel (Canceled) so the two cases are not both surfaced
// as "Cancelled" in the UI.
func ctxErrMessage(ctx context.Context) string {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Sprintf("Pipeline timed out after %s", pipelineTimeout)
	}
	return "Cancelled"
}

// googleResultFilter returns a google.ResultFilter that delegates to the
// Runner's filter classifier with the correct finding type and platform.
func (r *Runner) googleResultFilter() google.ResultFilter {
	classifier := r.filterClassifier
	return func(ctx context.Context, projectID string, input google.FilterInput) (domain.FilterDecision, error) {
		fi := FilterInput{
			FindingType: domain.FindingTypeGoogleResult,
			Platform:    "google",
			Title:       input.Title,
			Body:        input.Snippet + "\n" + input.PageText,
			URL:         input.URL,
			Domain:      input.Domain,
			SourceIdentity: domain.SourceIdentity{
				"domain":        input.Domain,
				"canonical_url": strings.ToLower(input.CanonicalURL),
			},
		}
		return classifier.Classify(ctx, projectID, fi)
	}
}

// Run executes a scout pipeline synchronously. If existingRunID is non-nil, the run
// row must already exist; otherwise a new one is created.
func (r *Runner) Run(ctx context.Context, projectID string, platform string, existingRunID *int64) (Result, error) {
	var runID int64
	if existingRunID != nil {
		runID = *existingRunID
	} else {
		id, err := r.Repo.CreateScoutRun(ctx, projectID, platform)
		if err != nil {
			return Result{}, err
		}
		runID = id
	}

	runCtx, cancel := context.WithTimeout(ctx, pipelineTimeout)
	r.mu.Lock()
	r.runs[runID] = cancel
	r.mu.Unlock()

	defer func() {
		cancel()
		r.mu.Lock()
		delete(r.runs, runID)
		r.mu.Unlock()
	}()

	var res Result
	var err error
	if platform == "google" {
		project, projErr := r.Repo.GetProject(runCtx, projectID)
		if projErr != nil {
			r.failRun(runID, projErr.Error())
			return Result{RunID: runID}, fmt.Errorf("project not found: %s", projectID)
		}
		provider := google.NewParallelSearchProvider()
		gr, gerr := google.RunGoogleScoutPipeline(runCtx, r.Repo, r.AI, project, projectID, runID, provider, r.googleResultFilter())
		res = Result{RunID: gr.RunID, PostsChecked: gr.PostsChecked, PostsFound: gr.PostsFound}
		err = gerr
	} else {
		res, err = r.runSocial(runCtx, projectID, platform, runID)
	}
	if err != nil {
		r.failRun(runID, err.Error())
		return Result{RunID: runID}, err
	}
	return res, nil
}

// StartAsync creates a scout_run row (or reuses existingRunID), registers it in the
// active-run map, and starts the pipeline in a goroutine.
func (r *Runner) StartAsync(projectID string, platform string, runID int64) {
	bgCtx := context.Background()
	bgCtx, cancel := context.WithTimeout(bgCtx, pipelineTimeout)

	r.mu.Lock()
	r.runs[runID] = cancel
	r.mu.Unlock()

	go func() {
		defer func() {
			cancel()
			r.mu.Lock()
			delete(r.runs, runID)
			r.mu.Unlock()
		}()

		var err error
		if platform == "google" {
			project, projErr := r.Repo.GetProject(bgCtx, projectID)
			if projErr != nil {
				r.failRun(runID, projErr.Error())
				return
			}
			provider := google.NewParallelSearchProvider()
			_, err = google.RunGoogleScoutPipeline(bgCtx, r.Repo, r.AI, project, projectID, runID, provider, r.googleResultFilter())
		} else {
			_, err = r.runSocial(bgCtx, projectID, platform, runID)
		}
		if err != nil {
			r.failRun(runID, err.Error())
		}
	}()
}

// Cancel cancels the context for a tracked run. Returns true if the run was tracked.
func (r *Runner) Cancel(runID int64) bool {
	r.mu.Lock()
	cancel, ok := r.runs[runID]
	r.mu.Unlock()
	if ok {
		cancel()
	}
	return ok
}

// runSocial executes the social (reddit / bluesky) scout pipeline, mirroring
// the Express _runScoutPipeline function exactly.
func (r *Runner) runSocial(ctx context.Context, projectID string, platform string, runID int64) (Result, error) {
	// 1. Verify project exists.
	project, err := r.Repo.GetProject(ctx, projectID)
	if err != nil {
		return Result{RunID: runID}, fmt.Errorf("project not found: %s", projectID)
	}

	// 2. Update step: loading queries.
	_ = r.Repo.UpdateScoutStep(ctx, runID, "Loading queries")

	// 3. Load enabled queries for this project + platform.
	queries, err := r.Repo.EnabledQueries(ctx, projectID, platform)
	if err != nil {
		return Result{RunID: runID}, err
	}

	// 4. No queries → complete with 0 counts.
	if len(queries) == 0 {
		if err := r.Repo.CompleteScoutRun(ctx, runID, 0, 0, nil); err != nil {
			return Result{RunID: runID}, err
		}
		return Result{RunID: runID, PostsChecked: 0, PostsFound: 0}, nil
	}

	// 5. Fetch posts.
	_ = r.Repo.UpdateScoutStep(ctx, runID, fmt.Sprintf("Fetching %s posts", platform))
	onFetchProgress := func(done, total int) {
		_ = r.Repo.UpdateScoutStep(ctx, runID,
			fmt.Sprintf("Fetching %s posts (%d/%d queries)", platform, done, total))
	}

	queryURLs := make([]string, len(queries))
	for i, q := range queries {
		queryURLs[i] = q.QueryURL
	}

	var fetchedPosts []FetchedPost
	switch platform {
	case "reddit":
		fetchedPosts, err = r.fetchReddit(ctx, queryURLs, onFetchProgress)
	case "bluesky":
		fetchedPosts, err = r.fetchBluesky(ctx, queryURLs, onFetchProgress)
	default:
		return Result{RunID: runID}, fmt.Errorf("unsupported platform: %s", platform)
	}
	if err != nil {
		return Result{RunID: runID}, err
	}

	postsChecked := int64(len(fetchedPosts))

	// 6. Filter against seen_posts.
	_ = r.Repo.UpdateScoutStep(ctx, runID, fmt.Sprintf("Filtering %d posts", len(fetchedPosts)))
	seenIDs, err := r.Repo.SeenIDs(ctx, projectID, platform)
	if err != nil {
		return Result{RunID: runID}, err
	}

	getPostID := func(p FetchedPost) string {
		if platform == "reddit" {
			return p.ID // p.ID holds the "name" (t3_xxx) for Reddit
		}
		return p.ID // for Bluesky, ID holds the URI
	}

	newPosts := make([]FetchedPost, 0, len(fetchedPosts))
	for _, p := range fetchedPosts {
		if !seenIDs[getPostID(p)] {
			newPosts = append(newPosts, p)
		}
	}

	// 7. Partition into visible and filtered candidates.
	visibleCandidates := make([]FetchedPost, 0, len(newPosts))
	filteredPosts := make([]domain.Post, 0)
	nowStr := time.Now().UTC().Format(time.RFC3339)
	for _, p := range newPosts {
		decision, err := r.filterClassifier.Classify(ctx, projectID, NormalizeFetchedPostForFiltering(platform, projectID, p))
		if err != nil {
			return Result{RunID: runID}, err
		}
		if decision.State == domain.FilterStateFiltered {
			fp := domain.Post{
				ID:                getPostID(p),
				ProjectID:         projectID,
				Platform:          platform,
				Status:            "new",
				EngagementType:    "karma",
				FilterState:       decision.State,
				FilterSource:      decision.Source,
				FilterSignature:   decision.Signature,
				FilterExplanation: decision.Explanation,
				SourceIdentity:    decision.SourceIdentity,
				FilteredAt:        &nowStr,
			}
			if decision.Reason != "" {
				fp.FilterReason = &decision.Reason
			}
			fp.FilterReasons = decision.Reasons
			if decision.Confidence != nil {
				fp.FilterConfidence = decision.Confidence
			}
			if platform == "reddit" {
				fp.Title = p.Title
				fp.Body = p.Selftext
				fp.Author = p.Author
				fp.URL = "https://www.reddit.com" + p.Permalink
				subreddit := p.Subreddit
				if subreddit != "" {
					fp.Subreddit = &subreddit
				}
				score := int64(p.Score)
				fp.RedditScore = &score
				numComments := int64(p.NumComments)
				fp.NumComments = &numComments
				if p.CreatedUTC != 0 {
					t := time.Unix(int64(p.CreatedUTC), 0).UTC().Format(time.RFC3339)
					fp.CreatedAt = &t
				}
			} else {
				title := p.Text
				if len(title) > 100 {
					title = title[:100]
				}
				fp.Title = title
				fp.Body = p.Text
				fp.Author = p.AuthorHandle
				fp.URL = p.PostURL
				uri := p.ID
				fp.BlueskyURI = &uri
				cid := p.CID
				if cid != "" {
					fp.BlueskyCID = &cid
				}
				likeCount := int64(p.LikeCount)
				fp.LikeCount = &likeCount
				replyCount := int64(p.ReplyCount)
				fp.ReplyCount = &replyCount
				repostCount := int64(p.RepostCount)
				fp.RepostCount = &repostCount
				if p.IndexedAt != "" {
					fp.CreatedAt = &p.IndexedAt
				}
			}
			filteredPosts = append(filteredPosts, fp)
		} else {
			visibleCandidates = append(visibleCandidates, p)
		}
	}

	// 8. Dedup (Reddit only) — applies only to visible candidates.
	if platform == "reddit" {
		visibleCandidates = DeduplicatePosts(visibleCandidates)
	}

	// 8b. Persist filtered posts before scoring visible ones, and mark them seen
	// so they are not re-classified on subsequent runs.
	if len(filteredPosts) > 0 {
		if _, err := r.Repo.InsertSocialPosts(ctx, filteredPosts); err != nil {
			return Result{RunID: runID}, err
		}
		filteredIDs := make([]string, len(filteredPosts))
		for i, fp := range filteredPosts {
			filteredIDs[i] = fp.ID
		}
		if err := r.Repo.MarkSeen(ctx, projectID, platform, filteredIDs); err != nil {
			return Result{RunID: runID}, err
		}
	}

	// Use visibleCandidates as the working set from here on.
	filtered := visibleCandidates

	// 9. No new posts → complete with postsChecked, postsFound=0.
	if len(filtered) == 0 {
		if err := r.Repo.CompleteScoutRun(ctx, runID, postsChecked, 0, nil); err != nil {
			return Result{RunID: runID}, err
		}
		return Result{RunID: runID, PostsChecked: postsChecked, PostsFound: 0}, nil
	}

	// 10. Build pain angles from enabled queries.
	angleSet := make(map[string]bool)
	for _, q := range queries {
		if q.Angle != "" {
			angleSet[q.Angle] = true
		}
	}
	painAngles := make([]string, 0, len(angleSet))
	for a := range angleSet {
		painAngles = append(painAngles, a)
	}

	// 11. Build scoring rubric: custom > research default.
	var scoringRubric *string
	if project.ScoringPrompt != nil && *project.ScoringPrompt != "" {
		scoringRubric = project.ScoringPrompt
	} else if project.Mode == "research" {
		rubric := defaultScoringRubric
		scoringRubric = &rubric
	}

	// 12. Normalize posts for scorer.
	totalToScore := len(filtered)
	_ = r.Repo.UpdateScoutStep(ctx, runID, fmt.Sprintf("Scoring 0/%d posts", totalToScore))
	scoringPosts := make([]ScoringPost, len(filtered))
	for i, p := range filtered {
		if platform == "reddit" {
			scoringPosts[i] = ScoringPost{
				ID:          p.ID,
				Title:       p.Title,
				Selftext:    p.Selftext,
				Subreddit:   p.Subreddit,
				Score:       p.Score,
				NumComments: p.NumComments,
			}
		} else {
			title := p.Text
			if len(title) > 100 {
				title = title[:100]
			}
			scoringPosts[i] = ScoringPost{
				ID:          p.ID,
				Title:       title,
				Selftext:    p.Text,
				Score:       p.LikeCount,
				NumComments: p.ReplyCount,
			}
		}
	}

	scoreResult, err := r.scorePosts(ctx, scoringPosts, painAngles, scoringRubric, project.Description, func(scored, total int) {
		_ = r.Repo.UpdateScoutStep(ctx, runID, fmt.Sprintf("Scoring %d/%d posts", scored, total))
	})
	if err != nil {
		return Result{RunID: runID}, err
	}

	// 12b. Mark seen only if scoring at least partially succeeded.
	if scoreResult.Stats.TotalScored > 0 || scoreResult.Stats.FailedBatches == 0 {
		allIDs := make([]string, len(fetchedPosts))
		for i, p := range fetchedPosts {
			allIDs[i] = getPostID(p)
		}
		if err := r.Repo.MarkSeen(ctx, projectID, platform, allIDs); err != nil {
			return Result{RunID: runID}, err
		}
	}

	// 13. Build score map.
	scoreMap := make(map[string]ScoredPost)
	for _, s := range scoreResult.Scores {
		if s.ID != "" {
			scoreMap[s.ID] = s
		}
	}

	// 14. Check cancellation before storage.
	if ctx.Err() != nil {
		r.failRun(runID, ctxErrMessage(ctx))
		return Result{RunID: runID, PostsChecked: postsChecked, PostsFound: 0}, nil
	}

	_ = r.Repo.UpdateScoutStep(ctx, runID, "Storing results")

	// 15. Build posts to insert (score >= 2).
	postsToInsert := make([]domain.Post, 0, len(filtered))
	for _, p := range filtered {
		postID := getPostID(p)
		scored, ok := scoreMap[postID]
		if !ok || scored.PostScore < 2 {
			continue
		}
		post := domain.Post{
			ID:             postID,
			ProjectID:      projectID,
			Platform:       platform,
			PostScore:      scored.PostScore,
			FinalScore:     scored.PostScore,
			Angle:          scored.Angle,
			Why:            scored.Why,
			Status:         "new",
			EngagementType: scored.EngagementType,
			KarmaTopic:     scored.KarmaTopic,
		}
		if platform == "reddit" {
			post.Title = p.Title
			post.Body = p.Selftext
			post.Author = p.Author
			post.URL = "https://www.reddit.com" + p.Permalink
			subreddit := p.Subreddit
			if subreddit != "" {
				post.Subreddit = &subreddit
			}
			score := int64(p.Score)
			post.RedditScore = &score
			numComments := int64(p.NumComments)
			post.NumComments = &numComments
			if p.CreatedUTC != 0 {
				t := time.Unix(int64(p.CreatedUTC), 0).UTC().Format(time.RFC3339)
				post.CreatedAt = &t
			}
		} else {
			title := p.Text
			if len(title) > 100 {
				title = title[:100]
			}
			post.Title = title
			post.Body = p.Text
			post.Author = p.AuthorHandle
			post.URL = p.PostURL
			likeCount := int64(p.LikeCount)
			post.LikeCount = &likeCount
			replyCount := int64(p.ReplyCount)
			post.ReplyCount = &replyCount
			repostCount := int64(p.RepostCount)
			post.RepostCount = &repostCount
			uri := p.ID
			post.BlueskyURI = &uri
			cid := p.CID
			if cid != "" {
				post.BlueskyCID = &cid
			}
			if p.IndexedAt != "" {
				post.CreatedAt = &p.IndexedAt
			}
		}
		postsToInsert = append(postsToInsert, post)
	}

	inserted, err := r.Repo.InsertSocialPosts(ctx, postsToInsert)
	if err != nil {
		return Result{RunID: runID}, err
	}
	postsFound := inserted

	var dmWarnings []string
	if r.dmTargets != nil {
		dmWarnings = r.dmTargets.Generate(ctx, project, platform, postsToInsert)
	}

	// 16. Build warnings text.
	var warnings []string
	if scoreResult.Stats.FailedBatches > 0 {
		warnings = append(warnings,
			fmt.Sprintf("Scoring: %d/%d batches failed", scoreResult.Stats.FailedBatches, scoreResult.Stats.TotalBatches))
		// Deduplicate errors by stripping "Batch N: " prefix.
		errorCounts := make(map[string]int)
		for _, e := range scoreResult.Stats.Errors {
			idx := strings.Index(e, ": ")
			msg := e
			if idx >= 0 {
				msg = e[idx+2:]
			}
			errorCounts[msg]++
		}
		for msg, count := range errorCounts {
			if count > 1 {
				warnings = append(warnings, fmt.Sprintf("  - %s (x%d)", msg, count))
			} else {
				warnings = append(warnings, fmt.Sprintf("  - %s", msg))
			}
		}
	}
	unmatchedCount := int64(len(filtered)) - int64(len(scoreMap))
	if unmatchedCount > 0 {
		warnings = append(warnings,
			fmt.Sprintf("%d/%d posts had no matching score returned", unmatchedCount, len(filtered)))
	}
	warnings = append(warnings, dmWarnings...)
	var warningsText *string
	if len(warnings) > 0 {
		s := strings.Join(warnings, "\n")
		warningsText = &s
	}

	// 17. Check cancellation again before completing.
	if ctx.Err() != nil {
		r.failRun(runID, ctxErrMessage(ctx))
		return Result{RunID: runID, PostsChecked: postsChecked, PostsFound: 0}, nil
	}

	if err := r.Repo.CompleteScoutRun(ctx, runID, postsChecked, postsFound, warningsText); err != nil {
		return Result{RunID: runID}, err
	}

	return Result{RunID: runID, PostsChecked: postsChecked, PostsFound: postsFound}, nil
}
