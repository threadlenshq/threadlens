package services

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
	"github.com/kyle/scout/open-core/apps/api/internal/tenant"
)

func testCtx() context.Context {
	return tenant.WithSubject(context.Background(), entitlements.Subject{
		RuntimeMode: entitlements.RuntimeModeSelfHosted,
	})
}

type testManualScoutSetup struct {
	svc  *ManualScoutService
	db   *sql.DB
	repo *repository.Repository
}

func setupManualScoutService(t *testing.T) *testManualScoutSetup {
	t.Helper()
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	filterClassifier := pipeline.NewFilterClassifier(nil, nil)
	resolver := entitlements.NewLocalResolver(entitlements.RuntimeModeSelfHosted, nil)
	svc := NewManualScoutService(repo, nil, filterClassifier, entitlements.RuntimeModeSelfHosted, resolver)
	return &testManualScoutSetup{svc: svc, db: db, repo: repo}
}

func insertProject(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO projects (id, name, mode) VALUES (?, 'Test Project', 'research')`, id)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
}

func insertSeenPost(t *testing.T, db *sql.DB, projectID, platform, postID string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO seen_posts (id, project_id, platform) VALUES (?, ?, ?)`, postID, projectID, platform)
	if err != nil {
		t.Fatalf("insert seen post: %v", err)
	}
}

func insertPost(t *testing.T, ctx context.Context, repo *repository.Repository, post domain.Post) {
	t.Helper()
	_, err := repo.InsertSocialPosts(ctx, []domain.Post{post})
	if err != nil {
		t.Fatalf("insert post: %v", err)
	}
}

func strPtr(s string) *string { return &s }

// ─── ScoutPost tests ─────────────────────────────────────────────────────────

func TestScoutPost_InvalidPlatform(t *testing.T) {
	setup := setupManualScoutService(t)
	ctx := testCtx()

	// Empty URL
	_, code, msg := setup.svc.ScoutPost(ctx, "proj1", "", "reddit")
	if code != http.StatusBadRequest {
		t.Errorf("empty URL: expected status %d, got %d", http.StatusBadRequest, code)
	}
	if msg != "URL is required" {
		t.Errorf("empty URL: expected message %q, got %q", "URL is required", msg)
	}

	// Unknown platform
	_, code, msg = setup.svc.ScoutPost(ctx, "proj1", "http://example.com", "twitter")
	if code != http.StatusBadRequest {
		t.Errorf("unknown platform: expected status %d, got %d", http.StatusBadRequest, code)
	}
	if msg != `platform must be "reddit" or "bluesky"` {
		t.Errorf("unknown platform: expected message %q, got %q", `platform must be "reddit" or "bluesky"`, msg)
	}
}

func TestScoutPost_FetchError(t *testing.T) {
	setup := setupManualScoutService(t)
	insertProject(t, setup.db, "proj-fetch-err")
	ctx := testCtx()

	setup.svc.fetchReddit = func(ctx context.Context, url string) (*pipeline.FetchedPost, error) {
		return nil, errors.New("reddit API down")
	}

	result, code, msg := setup.svc.ScoutPost(ctx, "proj-fetch-err", "https://reddit.com/r/test/comments/abc/test", "reddit")
	if code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
	if result.Status != "error" {
		t.Errorf("expected status %q, got %q", "error", result.Status)
	}
	if result.Error != "reddit API down" {
		t.Errorf("expected error %q, got %q", "reddit API down", result.Error)
	}
}

func TestScoutPost_AlreadyScouted(t *testing.T) {
	setup := setupManualScoutService(t)
	insertProject(t, setup.db, "proj-seen")
	ctx := testCtx()

	postID := "t3_seen123"
	existingPost := domain.Post{
		ID:        postID,
		ProjectID: "proj-seen",
		Platform:  "reddit",
		Title:     "Already seen post",
		Author:    "user1",
		Status:    "new",
	}
	insertPost(t, ctx, setup.repo, existingPost)
	insertSeenPost(t, setup.db, "proj-seen", "reddit", postID)

	setup.svc.fetchReddit = func(ctx context.Context, url string) (*pipeline.FetchedPost, error) {
		return &pipeline.FetchedPost{
			ID:     postID,
			Title:  "Already seen post",
			Author: "user1",
		}, nil
	}

	result, code, msg := setup.svc.ScoutPost(ctx, "proj-seen", "https://reddit.com/r/test/comments/abc/test", "reddit")
	if code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
	if result.Status != "already_scouted" {
		t.Errorf("expected status %q, got %q", "already_scouted", result.Status)
	}
	if result.Post == nil {
		t.Fatal("expected post to be returned")
	}
	if result.Post.ID != postID {
		t.Errorf("expected post ID %q, got %q", postID, result.Post.ID)
	}
}

func TestScoutPost_Filtered(t *testing.T) {
	setup := setupManualScoutService(t)
	insertProject(t, setup.db, "proj-filter")
	ctx := testCtx()

	postID := "t3_filter123"
	setup.svc.fetchReddit = func(ctx context.Context, url string) (*pipeline.FetchedPost, error) {
		return &pipeline.FetchedPost{
			ID:          postID,
			Title:       "I built a new app — check it out at example.com",
			Selftext:    "Launching my product, feedback welcome",
			Author:      "builder",
			Permalink:   "/r/entrepreneur/comments/filter123/i_built_a_new_app/",
			Subreddit:   "entrepreneur",
			Score:       50,
			NumComments: 10,
		}, nil
	}

	result, code, msg := setup.svc.ScoutPost(ctx, "proj-filter", "https://reddit.com/r/entrepreneur/comments/filter123/i_built_a_new_app/", "reddit")
	if code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
	if result.Status != "filtered" {
		t.Errorf("expected status %q, got %q", "filtered", result.Status)
	}
	if !result.Filtered {
		t.Error("expected Filtered to be true")
	}
	if result.PostID != postID {
		t.Errorf("expected PostID %q, got %q", postID, result.PostID)
	}
	if result.Reason == "" {
		t.Error("expected Reason to be set")
	}

	// Verify post was inserted and marked seen
	seenIDs, err := setup.repo.SeenIDs(ctx, "proj-filter", "reddit")
	if err != nil {
		t.Fatalf("SeenIDs: %v", err)
	}
	if !seenIDs[postID] {
		t.Error("expected post to be marked as seen")
	}
}

func TestScoutPost_ScorerReturnsHighScore(t *testing.T) {
	setup := setupManualScoutService(t)
	insertProject(t, setup.db, "proj-high")
	ctx := testCtx()

	postID := "t3_high123"
	setup.svc.fetchReddit = func(ctx context.Context, url string) (*pipeline.FetchedPost, error) {
		return &pipeline.FetchedPost{
			ID:          postID,
			Title:       "How do I solve this problem?",
			Selftext:    "I need some advice.",
			Author:      "helpseeker",
			Permalink:   "/r/startups/comments/high123/how_do_i_solve_this/",
			Subreddit:   "startups",
			Score:       20,
			NumComments: 5,
		}, nil
	}

	setup.svc.scorePosts = func(ctx context.Context, repo *repository.Repository, aiSvc *ai.Service, posts []pipeline.ScoringPost, painAngles []string, batchSize int, scoringRubric *string, description string, onProgress func(int, int)) (*pipeline.ScoreResult, error) {
		return &pipeline.ScoreResult{
			Scores: []pipeline.ScoredPost{
				{ID: postID, PostScore: 3.5, EngagementType: "product"},
			},
		}, nil
	}

	result, code, msg := setup.svc.ScoutPost(ctx, "proj-high", "https://reddit.com/r/startups/comments/high123/how_do_i_solve_this/", "reddit")
	if code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
	if result.Status != "saved" {
		t.Errorf("expected status %q, got %q", "saved", result.Status)
	}
	if result.Score != 3.5 {
		t.Errorf("expected score %f, got %f", 3.5, result.Score)
	}
	if result.PostID != postID {
		t.Errorf("expected PostID %q, got %q", postID, result.PostID)
	}

	// Verify post was inserted and marked seen
	seenIDs, err := setup.repo.SeenIDs(ctx, "proj-high", "reddit")
	if err != nil {
		t.Fatalf("SeenIDs: %v", err)
	}
	if !seenIDs[postID] {
		t.Error("expected post to be marked as seen")
	}
}

func TestScoutPost_ScorerReturnsLowScore(t *testing.T) {
	setup := setupManualScoutService(t)
	insertProject(t, setup.db, "proj-low")
	ctx := testCtx()

	postID := "t3_low123"
	setup.svc.fetchReddit = func(ctx context.Context, url string) (*pipeline.FetchedPost, error) {
		return &pipeline.FetchedPost{
			ID:          postID,
			Title:       "How do I solve this problem?",
			Selftext:    "I need some advice.",
			Author:      "helpseeker",
			Permalink:   "/r/startups/comments/low123/how_do_i_solve_this/",
			Subreddit:   "startups",
			Score:       20,
			NumComments: 5,
		}, nil
	}

	setup.svc.scorePosts = func(ctx context.Context, repo *repository.Repository, aiSvc *ai.Service, posts []pipeline.ScoringPost, painAngles []string, batchSize int, scoringRubric *string, description string, onProgress func(int, int)) (*pipeline.ScoreResult, error) {
		return &pipeline.ScoreResult{
			Scores: []pipeline.ScoredPost{
				{ID: postID, PostScore: 1.0, EngagementType: "karma"},
			},
		}, nil
	}

	result, code, msg := setup.svc.ScoutPost(ctx, "proj-low", "https://reddit.com/r/startups/comments/low123/how_do_i_solve_this/", "reddit")
	if code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
	if result.Status != "needs_decision" {
		t.Errorf("expected status %q, got %q", "needs_decision", result.Status)
	}
	if result.Score != 1.0 {
		t.Errorf("expected score %f, got %f", 1.0, result.Score)
	}
	if result.PostID != postID {
		t.Errorf("expected PostID %q, got %q", postID, result.PostID)
	}
	if result.Post == nil {
		t.Fatal("expected Post to be returned")
	}
}

func TestScoutPost_ScorerError(t *testing.T) {
	setup := setupManualScoutService(t)
	insertProject(t, setup.db, "proj-score-err")
	ctx := testCtx()

	postID := "t3_score_err123"
	setup.svc.fetchReddit = func(ctx context.Context, url string) (*pipeline.FetchedPost, error) {
		return &pipeline.FetchedPost{
			ID:          postID,
			Title:       "How do I solve this problem?",
			Selftext:    "I need some advice.",
			Author:      "helpseeker",
			Permalink:   "/r/startups/comments/score_err123/how_do_i_solve_this/",
			Subreddit:   "startups",
			Score:       20,
			NumComments: 5,
		}, nil
	}

	setup.svc.scorePosts = func(ctx context.Context, repo *repository.Repository, aiSvc *ai.Service, posts []pipeline.ScoringPost, painAngles []string, batchSize int, scoringRubric *string, description string, onProgress func(int, int)) (*pipeline.ScoreResult, error) {
		return nil, errors.New("scoring service unavailable")
	}

	result, code, msg := setup.svc.ScoutPost(ctx, "proj-score-err", "https://reddit.com/r/startups/comments/score_err123/how_do_i_solve_this/", "reddit")
	if code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
	if result.Status != "needs_decision" {
		t.Errorf("expected status %q, got %q", "needs_decision", result.Status)
	}
	if result.Score != 0 {
		t.Errorf("expected score 0, got %f", result.Score)
	}
	if result.PostID != postID {
		t.Errorf("expected PostID %q, got %q", postID, result.PostID)
	}
	if result.Post == nil {
		t.Fatal("expected Post to be returned")
	}
	if result.Post.PostScore != 0 {
		t.Errorf("expected Post.PostScore 0, got %f", result.Post.PostScore)
	}
	if result.Post.FinalScore != 0 {
		t.Errorf("expected Post.FinalScore 0, got %f", result.Post.FinalScore)
	}
}

func TestScoutPost_ProjectNotFound(t *testing.T) {
	setup := setupManualScoutService(t)
	ctx := testCtx()

	// Don't insert any project

	_, code, msg := setup.svc.ScoutPost(ctx, "nonexistent-project", "https://reddit.com/r/test/comments/abc/test", "reddit")
	if code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, code)
	}
	if msg != "Project not found" {
		t.Errorf("expected message %q, got %q", "Project not found", msg)
	}
}

// ─── CommitDecision tests ────────────────────────────────────────────────────

func TestCommitDecision_Keep(t *testing.T) {
	setup := setupManualScoutService(t)
	insertProject(t, setup.db, "proj-keep")
	ctx := testCtx()

	postID := "t3_keep123"
	platform := "reddit"
	postData := PostData{
		ID:       &postID,
		Platform: &platform,
		Title:    strPtr("Keep this post"),
		Author:   strPtr("author1"),
	}

	result, code, msg := setup.svc.CommitDecision(ctx, "proj-keep", "keep", postData)
	if code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
	if result.Status != "saved" {
		t.Errorf("expected status %q, got %q", "saved", result.Status)
	}
	if result.PostID != postID {
		t.Errorf("expected PostID %q, got %q", postID, result.PostID)
	}

	// Verify post was inserted and marked seen
	seenIDs, err := setup.repo.SeenIDs(ctx, "proj-keep", "reddit")
	if err != nil {
		t.Fatalf("SeenIDs: %v", err)
	}
	if !seenIDs[postID] {
		t.Error("expected post to be marked as seen")
	}
}

func TestCommitDecision_Exclude(t *testing.T) {
	setup := setupManualScoutService(t)
	insertProject(t, setup.db, "proj-exclude")
	ctx := testCtx()

	postID := "t3_exclude123"
	platform := "reddit"
	postData := PostData{
		ID:       &postID,
		Platform: &platform,
	}

	result, code, msg := setup.svc.CommitDecision(ctx, "proj-exclude", "exclude", postData)
	if code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
	if result.Status != "excluded" {
		t.Errorf("expected status %q, got %q", "excluded", result.Status)
	}
	if result.PostID != postID {
		t.Errorf("expected PostID %q, got %q", postID, result.PostID)
	}

	// Verify post was marked as seen
	seenIDs, err := setup.repo.SeenIDs(ctx, "proj-exclude", "reddit")
	if err != nil {
		t.Fatalf("SeenIDs: %v", err)
	}
	if !seenIDs[postID] {
		t.Error("expected post to be marked as seen")
	}
}

func TestCommitDecision_InvalidDecision(t *testing.T) {
	setup := setupManualScoutService(t)
	insertProject(t, setup.db, "proj-invalid")
	ctx := testCtx()

	postID := "t3_invalid123"
	platform := "reddit"
	postData := PostData{
		ID:       &postID,
		Platform: &platform,
	}

	_, code, msg := setup.svc.CommitDecision(ctx, "proj-invalid", "unknown", postData)
	if code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, code)
	}
	if msg != `decision must be "keep" or "exclude"` {
		t.Errorf("expected message %q, got %q", `decision must be "keep" or "exclude"`, msg)
	}
}
