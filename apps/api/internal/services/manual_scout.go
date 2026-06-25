package services

import (
	"context"
	"net/http"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/tenant"
)

// ManualScoutResult is the response shape for manual scouting operations.
type ManualScoutResult struct {
	Status      string       `json:"status"`
	Post        *domain.Post `json:"post,omitempty"`
	PostID      string       `json:"post_id,omitempty"`
	Score       float64      `json:"score"`
	Filtered    bool         `json:"filtered"`
	Error       string       `json:"error,omitempty"`
	Reason      string       `json:"reason,omitempty"`
	Reasons     []string     `json:"reasons,omitempty"`
	Explanation string       `json:"explanation,omitempty"`
	Source      string       `json:"source,omitempty"`
	Signature   string       `json:"signature,omitempty"`
	Confidence  *float64     `json:"confidence,omitempty"`
}

// FilterInfo captures filter decision details.
type FilterInfo struct {
	Reason      string
	Reasons     []string
	Explanation string
	Source      string
	Signature   string
	Confidence  *float64
}

// PostData is a round-trip payload for commit with all fields as pointers.
type PostData struct {
	ID                *string                `json:"id"`
	ProjectID         *string                `json:"project_id"`
	Platform          *string                `json:"platform"`
	Title             *string                `json:"title"`
	Body              *string                `json:"body"`
	Author            *string                `json:"author"`
	URL               *string                `json:"url"`
	Subreddit         *string                `json:"subreddit"`
	RedditScore       *int64                 `json:"reddit_score"`
	NumComments       *int64                 `json:"num_comments"`
	LikeCount         *int64                 `json:"like_count"`
	ReplyCount        *int64                 `json:"reply_count"`
	RepostCount       *int64                 `json:"repost_count"`
	BlueskyURI        *string                `json:"bluesky_uri"`
	BlueskyCID        *string                `json:"bluesky_cid"`
	PostScore         *float64               `json:"post_score"`
	CommentScore      *float64               `json:"comment_score"`
	FinalScore        *float64               `json:"final_score"`
	Angle             *string                `json:"angle"`
	Why               *string                `json:"why"`
	EngagementType    *string                `json:"engagement_type"`
	KarmaTopic        *string                `json:"karma_topic"`
	TopCommentSignals *string                `json:"top_comment_signals"`
	Status            *string                `json:"status"`
	DraftComment      *string                `json:"draft_comment"`
	DraftProvider     *string                `json:"draft_provider"`
	SignalType        *string                `json:"signal_type"`
	FilterState       *string                `json:"filter_state"`
	FilterReason      *string                `json:"filter_reason"`
	FilterReasons     *[]string              `json:"filter_reasons"`
	FilterExplanation *string                `json:"filter_explanation"`
	FilterConfidence  *float64               `json:"filter_confidence"`
	FilterSource      *string                `json:"filter_source"`
	FilterSignature   *string                `json:"filter_signature"`
	FilterJobID       *int64                 `json:"filter_job_id"`
	FilteredAt        *string                `json:"filtered_at"`
	RecoveredAt       *string                `json:"recovered_at"`
	RecoveryNote      *string                `json:"recovery_note"`
	SourceIdentity    *domain.SourceIdentity `json:"source_identity"`
	CreatedAt         *string                `json:"created_at"`
	FoundAt           *string                `json:"found_at"`
	ScoutedAt         *string                `json:"scouted_at"`
	DMTargets         *[]domain.DMTarget     `json:"dm_targets"`
}

// ManualScoutService handles manual scouting of individual posts.
type ManualScoutService struct {
	repo             *repository.Repository
	ai               *ai.Service
	filterClassifier *pipeline.FilterClassifier
	mode             entitlements.RuntimeMode
	resolver         entitlements.Resolver

	// Overridable fetchers for testing.
	fetchReddit  func(ctx context.Context, url string) (*pipeline.FetchedPost, error)
	fetchBluesky func(ctx context.Context, url string) (*pipeline.FetchedPost, error)
	scorePosts   func(ctx context.Context, repo *repository.Repository, aiSvc *ai.Service, posts []pipeline.ScoringPost, painAngles []string, batchSize int, scoringRubric *string, description string, onProgress func(int, int)) (*pipeline.ScoreResult, error)
}

// NewManualScoutService creates a new ManualScoutService.
func NewManualScoutService(repo *repository.Repository, aiSvc *ai.Service, filterClassifier *pipeline.FilterClassifier, mode entitlements.RuntimeMode, resolver entitlements.Resolver) *ManualScoutService {
	s := &ManualScoutService{
		repo:             repo,
		ai:               aiSvc,
		filterClassifier: filterClassifier,
		mode:             mode,
		resolver:         resolver,
	}
	s.fetchReddit = func(ctx context.Context, url string) (*pipeline.FetchedPost, error) {
		return pipeline.FetchSingleRedditPost(ctx, url)
	}
	s.fetchBluesky = func(ctx context.Context, url string) (*pipeline.FetchedPost, error) {
		return pipeline.FetchSingleBlueskyPost(ctx, url)
	}
	s.scorePosts = func(ctx context.Context, repo *repository.Repository, aiSvc *ai.Service, posts []pipeline.ScoringPost, painAngles []string, batchSize int, scoringRubric *string, description string, onProgress func(int, int)) (*pipeline.ScoreResult, error) {
		var descPtr *string
		if description != "" {
			descPtr = &description
		}
		res, err := pipeline.ScorePosts(ctx, repo, aiSvc, posts, painAngles, batchSize, scoringRubric, descPtr, onProgress)
		if err != nil {
			return nil, err
		}
		return &res, nil
	}
	return s
}

// ScoutPost fetches, filters, and scores a single post for a project.
func (s *ManualScoutService) ScoutPost(ctx context.Context, projectID string, url string, platform string) (ManualScoutResult, int, string) {
	if url == "" {
		return ManualScoutResult{}, http.StatusBadRequest, "URL is required"
	}
	if platform != "reddit" && platform != "bluesky" {
		return ManualScoutResult{}, http.StatusBadRequest, `platform must be "reddit" or "bluesky"`
	}

	decision, err := s.resolver.Check(ctx, entitlements.CheckRequest{
		Subject:    tenant.SubjectFromContext(ctx, s.mode),
		Capability: entitlements.CapabilityForScoutPlatform(platform),
		ProjectID:  projectID,
		Action:     "scout.manual",
	})
	if err != nil {
		return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
	}
	if err := entitlements.EnsureAllowed(decision); err != nil {
		return ManualScoutResult{}, entitlements.StatusCode(err), err.Error()
	}

	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Project not found"
		}
		return ManualScoutResult{}, code, msg
	}

	var fetched *pipeline.FetchedPost
	if platform == "reddit" {
		fetched, err = s.fetchReddit(ctx, url)
	} else {
		fetched, err = s.fetchBluesky(ctx, url)
	}
	if err != nil {
		return ManualScoutResult{Status: "error", Error: err.Error()}, http.StatusOK, ""
	}

	postID := fetched.ID

	seenIDs, err := s.repo.SeenIDs(ctx, projectID, platform)
	if err != nil {
		return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
	}
	if seenIDs[postID] {
		existing, err := s.repo.GetPost(ctx, projectID, postID)
		if err != nil {
			return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
		}
		return ManualScoutResult{Status: "already_scouted", Post: &existing}, http.StatusOK, ""
	}

	filterDecision, err := s.filterClassifier.Classify(ctx, projectID, pipeline.NormalizeFetchedPostForFiltering(platform, projectID, *fetched))
	if err != nil {
		return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
	}

	if filterDecision.State == domain.FilterStateFiltered {
		post := buildFilteredPost(platform, projectID, postID, *fetched, filterDecision)
		if _, err := s.repo.InsertSocialPosts(ctx, []domain.Post{post}); err != nil {
			return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
		}
		if err := s.repo.MarkSeen(ctx, projectID, platform, []string{postID}); err != nil {
			return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
		}
		return ManualScoutResult{
			Status:      "filtered",
			PostID:      postID,
			Filtered:    true,
			Reason:      filterDecision.Reason,
			Reasons:     filterDecision.Reasons,
			Explanation: filterDecision.Explanation,
			Source:      filterDecision.Source,
			Signature:   filterDecision.Signature,
			Confidence:  filterDecision.Confidence,
		}, http.StatusOK, ""
	}

	queries, err := s.repo.EnabledQueries(ctx, projectID, platform)
	if err != nil {
		return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
	}
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

	scoringPosts := []pipeline.ScoringPost{buildScoringPost(platform, *fetched)}

	var scoringRubric *string
	if project.ScoringPrompt != nil && *project.ScoringPrompt != "" {
		scoringRubric = project.ScoringPrompt
	}

	description := ""
	if project.Description != nil {
		description = *project.Description
	}

	scoreResult, err := s.scorePosts(ctx, s.repo, s.ai, scoringPosts, painAngles, 1, scoringRubric, description, nil)
	if err != nil {
		post := buildScoredPost(platform, projectID, postID, *fetched, pipeline.ScoredPost{})
		return ManualScoutResult{Status: "needs_decision", Post: &post, PostID: postID, Score: 0}, http.StatusOK, ""
	}

	if len(scoreResult.Scores) == 0 {
		post := buildScoredPost(platform, projectID, postID, *fetched, pipeline.ScoredPost{})
		return ManualScoutResult{Status: "needs_decision", Post: &post, PostID: postID, Score: 0}, http.StatusOK, ""
	}

	scored := scoreResult.Scores[0]
	if scored.PostScore >= 2 {
		post := buildScoredPost(platform, projectID, postID, *fetched, scored)
		if _, err := s.repo.InsertSocialPosts(ctx, []domain.Post{post}); err != nil {
			return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
		}
		if err := s.repo.MarkSeen(ctx, projectID, platform, []string{postID}); err != nil {
			return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
		}
		return ManualScoutResult{Status: "saved", PostID: postID, Score: scored.PostScore}, http.StatusOK, ""
	}

	post := buildScoredPost(platform, projectID, postID, *fetched, scored)
	return ManualScoutResult{Status: "needs_decision", Post: &post, PostID: postID, Score: scored.PostScore}, http.StatusOK, ""
}

// CommitDecision persists a manual keep/exclude decision for a post.
func (s *ManualScoutService) CommitDecision(ctx context.Context, projectID string, decision string, postData PostData) (ManualScoutResult, int, string) {
	if decision != "keep" && decision != "exclude" {
		return ManualScoutResult{}, http.StatusBadRequest, `decision must be "keep" or "exclude"`
	}

	if postData.ID == nil || *postData.ID == "" {
		return ManualScoutResult{}, http.StatusBadRequest, "post ID is required"
	}
	postID := *postData.ID

	if postData.Platform == nil || *postData.Platform == "" {
		return ManualScoutResult{}, http.StatusBadRequest, "platform is required"
	}
	platform := *postData.Platform

	entDecision, err := s.resolver.Check(ctx, entitlements.CheckRequest{
		Subject:    tenant.SubjectFromContext(ctx, s.mode),
		Capability: entitlements.CapabilityForScoutPlatform(platform),
		ProjectID:  projectID,
		Action:     "scout.manual",
	})
	if err != nil {
		return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
	}
	if err := entitlements.EnsureAllowed(entDecision); err != nil {
		return ManualScoutResult{}, entitlements.StatusCode(err), err.Error()
	}

	_, err = s.repo.GetProject(ctx, projectID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Project not found"
		}
		return ManualScoutResult{}, code, msg
	}

	if decision == "keep" {
		post := buildPostFromPostData(postData)
		post.ProjectID = projectID
		if _, err := s.repo.InsertSocialPosts(ctx, []domain.Post{post}); err != nil {
			return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
		}
		if err := s.repo.MarkSeen(ctx, projectID, platform, []string{postID}); err != nil {
			return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
		}
		return ManualScoutResult{Status: "saved", PostID: postID}, http.StatusOK, ""
	}

	if err := s.repo.MarkSeen(ctx, projectID, platform, []string{postID}); err != nil {
		return ManualScoutResult{}, http.StatusInternalServerError, "Internal server error"
	}
	return ManualScoutResult{Status: "excluded", PostID: postID}, http.StatusOK, ""
}

func buildFilteredPost(platform, projectID, postID string, fetched pipeline.FetchedPost, decision domain.FilterDecision) domain.Post {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	post := domain.Post{
		ID:                postID,
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
		post.FilterReason = &decision.Reason
	}
	post.FilterReasons = decision.Reasons
	if decision.Confidence != nil {
		post.FilterConfidence = decision.Confidence
	}
	if platform == "reddit" {
		post.Title = fetched.Title
		post.Body = fetched.Selftext
		post.Author = fetched.Author
		post.URL = "https://www.reddit.com" + fetched.Permalink
		if fetched.Subreddit != "" {
			post.Subreddit = &fetched.Subreddit
		}
		score := int64(fetched.Score)
		post.RedditScore = &score
		numComments := int64(fetched.NumComments)
		post.NumComments = &numComments
		if fetched.CreatedUTC != 0 {
			t := time.Unix(int64(fetched.CreatedUTC), 0).UTC().Format(time.RFC3339)
			post.CreatedAt = &t
		}
	} else {
		// Bluesky posts use full text as title; truncate for display consistency.
		title := fetched.Text
		if len(title) > 100 {
			title = title[:100]
		}
		post.Title = title
		post.Body = fetched.Text
		post.Author = fetched.AuthorHandle
		post.URL = fetched.PostURL
		uri := fetched.ID
		post.BlueskyURI = &uri
		cid := fetched.CID
		if cid != "" {
			post.BlueskyCID = &cid
		}
		likeCount := int64(fetched.LikeCount)
		post.LikeCount = &likeCount
		replyCount := int64(fetched.ReplyCount)
		post.ReplyCount = &replyCount
		repostCount := int64(fetched.RepostCount)
		post.RepostCount = &repostCount
		if fetched.IndexedAt != "" {
			post.CreatedAt = &fetched.IndexedAt
		}
	}
	return post
}

func buildScoringPost(platform string, fetched pipeline.FetchedPost) pipeline.ScoringPost {
	if platform == "reddit" {
		return pipeline.ScoringPost{
			ID:          fetched.ID,
			Title:       fetched.Title,
			Selftext:    fetched.Selftext,
			Subreddit:   fetched.Subreddit,
			Score:       fetched.Score,
			NumComments: fetched.NumComments,
		}
	}
	// Bluesky posts use full text as title; truncate for display consistency.
	title := fetched.Text
	if len(title) > 100 {
		title = title[:100]
	}
	return pipeline.ScoringPost{
		ID:          fetched.ID,
		Title:       title,
		Selftext:    fetched.Text,
		Score:       fetched.LikeCount,
		NumComments: fetched.ReplyCount,
	}
}

func buildScoredPost(platform, projectID, postID string, fetched pipeline.FetchedPost, scored pipeline.ScoredPost) domain.Post {
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
		post.Title = fetched.Title
		post.Body = fetched.Selftext
		post.Author = fetched.Author
		post.URL = "https://www.reddit.com" + fetched.Permalink
		if fetched.Subreddit != "" {
			post.Subreddit = &fetched.Subreddit
		}
		score := int64(fetched.Score)
		post.RedditScore = &score
		numComments := int64(fetched.NumComments)
		post.NumComments = &numComments
		if fetched.CreatedUTC != 0 {
			t := time.Unix(int64(fetched.CreatedUTC), 0).UTC().Format(time.RFC3339)
			post.CreatedAt = &t
		}
	} else {
		// Bluesky posts use full text as title; truncate for display consistency.
		title := fetched.Text
		if len(title) > 100 {
			title = title[:100]
		}
		post.Title = title
		post.Body = fetched.Text
		post.Author = fetched.AuthorHandle
		post.URL = fetched.PostURL
		likeCount := int64(fetched.LikeCount)
		post.LikeCount = &likeCount
		replyCount := int64(fetched.ReplyCount)
		post.ReplyCount = &replyCount
		repostCount := int64(fetched.RepostCount)
		post.RepostCount = &repostCount
		uri := fetched.ID
		post.BlueskyURI = &uri
		cid := fetched.CID
		if cid != "" {
			post.BlueskyCID = &cid
		}
		if fetched.IndexedAt != "" {
			post.CreatedAt = &fetched.IndexedAt
		}
	}
	return post
}

func buildPostFromPostData(data PostData) domain.Post {
	p := domain.Post{}
	if data.ID != nil {
		p.ID = *data.ID
	}
	if data.ProjectID != nil {
		p.ProjectID = *data.ProjectID
	}
	if data.Platform != nil {
		p.Platform = *data.Platform
	}
	if data.Title != nil {
		p.Title = *data.Title
	}
	if data.Body != nil {
		p.Body = *data.Body
	}
	if data.Author != nil {
		p.Author = *data.Author
	}
	if data.URL != nil {
		p.URL = *data.URL
	}
	p.Subreddit = data.Subreddit
	p.RedditScore = data.RedditScore
	p.NumComments = data.NumComments
	p.LikeCount = data.LikeCount
	p.ReplyCount = data.ReplyCount
	p.RepostCount = data.RepostCount
	p.BlueskyURI = data.BlueskyURI
	p.BlueskyCID = data.BlueskyCID
	if data.PostScore != nil {
		p.PostScore = *data.PostScore
	}
	p.CommentScore = data.CommentScore
	if data.FinalScore != nil {
		p.FinalScore = *data.FinalScore
	}
	p.Angle = data.Angle
	if data.Why != nil {
		p.Why = *data.Why
	}
	if data.EngagementType != nil {
		p.EngagementType = *data.EngagementType
	}
	p.KarmaTopic = data.KarmaTopic
	p.TopCommentSignals = data.TopCommentSignals
	if data.Status != nil {
		p.Status = *data.Status
	}
	p.DraftComment = data.DraftComment
	p.DraftProvider = data.DraftProvider
	p.SignalType = data.SignalType
	if data.FilterState != nil {
		p.FilterState = *data.FilterState
	}
	p.FilterReason = data.FilterReason
	if data.FilterReasons != nil {
		p.FilterReasons = *data.FilterReasons
	}
	if data.FilterExplanation != nil {
		p.FilterExplanation = *data.FilterExplanation
	}
	p.FilterConfidence = data.FilterConfidence
	if data.FilterSource != nil {
		p.FilterSource = *data.FilterSource
	}
	if data.FilterSignature != nil {
		p.FilterSignature = *data.FilterSignature
	}
	p.FilterJobID = data.FilterJobID
	p.FilteredAt = data.FilteredAt
	p.RecoveredAt = data.RecoveredAt
	p.RecoveryNote = data.RecoveryNote
	if data.SourceIdentity != nil {
		p.SourceIdentity = *data.SourceIdentity
	}
	p.CreatedAt = data.CreatedAt
	if data.FoundAt != nil {
		p.FoundAt = *data.FoundAt
	}
	if data.ScoutedAt != nil {
		p.ScoutedAt = *data.ScoutedAt
	}
	if data.DMTargets != nil {
		p.DMTargets = *data.DMTargets
	}
	return p
}
