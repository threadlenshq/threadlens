package repository_test

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func seedPostProject(t *testing.T, repo *repository.Repository, id string) {
	t.Helper()
	_, err := repo.DB.Exec(
		`INSERT INTO projects (id, name, mode, created_at, updated_at) VALUES (?, ?, 'research', datetime('now'), datetime('now'))`,
		id, id,
	)
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
}

func TestInsertSocialPosts_FilterMetadataRoundTrips(t *testing.T) {
	db := testhelpers.OpenTestDB(t)
	repo := repository.New(db)
	ctx := context.Background()
	seedPostProject(t, repo, "proj1")

	reason := "spam"
	conf := 0.92
	jobID := int64(99)
	filteredAt := "2026-06-01T12:00:00Z"

	posts := []domain.Post{
		{
			ID:                "r-insert-test",
			ProjectID:         "proj1",
			Platform:          "reddit",
			Title:             "Test insert",
			Body:              "body",
			Author:            "user",
			URL:               "https://reddit.com/r/x/test",
			PostScore:         5,
			FinalScore:        5,
			EngagementType:    "karma",
			Status:            "new",
			FilterState:       domain.FilterStateFiltered,
			FilterReason:      &reason,
			FilterReasons:     []string{"spam", "promotional_launch_language"},
			FilterExplanation: "test explanation",
			FilterConfidence:  &conf,
			FilterSource:      domain.FilterSourceRules,
			FilterSignature:   "filter:abc123",
			FilterJobID:       &jobID,
			FilteredAt:        &filteredAt,
			SourceIdentity:    domain.SourceIdentity{"reddit_author": "user", "subreddit": "testsubreddit"},
		},
	}

	n, err := repo.InsertSocialPosts(ctx, posts)
	if err != nil {
		t.Fatalf("InsertSocialPosts: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 inserted, got %d", n)
	}

	// GetPost should return the filtered post even though list queries exclude it
	post, err := repo.GetPost(ctx, "proj1", "r-insert-test")
	if err != nil {
		t.Fatalf("GetPost: %v", err)
	}
	if post.FilterState != domain.FilterStateFiltered {
		t.Errorf("FilterState = %q, want %q", post.FilterState, domain.FilterStateFiltered)
	}
	if post.FilterReason == nil || *post.FilterReason != reason {
		t.Errorf("FilterReason = %v, want %q", post.FilterReason, reason)
	}
	if len(post.FilterReasons) != 2 || post.FilterReasons[0] != "spam" {
		t.Errorf("FilterReasons = %v", post.FilterReasons)
	}
	if post.FilterExplanation != "test explanation" {
		t.Errorf("FilterExplanation = %q", post.FilterExplanation)
	}
	if post.FilterConfidence == nil || *post.FilterConfidence != 0.92 {
		t.Errorf("FilterConfidence = %v", post.FilterConfidence)
	}
	if post.FilterSource != domain.FilterSourceRules {
		t.Errorf("FilterSource = %q", post.FilterSource)
	}
	if post.FilterSignature != "filter:abc123" {
		t.Errorf("FilterSignature = %q", post.FilterSignature)
	}
	if post.FilterJobID == nil || *post.FilterJobID != 99 {
		t.Errorf("FilterJobID = %v", post.FilterJobID)
	}
	if post.FilteredAt == nil {
		t.Error("FilteredAt is nil")
	}
	if post.SourceIdentity["reddit_author"] != "user" {
		t.Errorf("SourceIdentity = %v", post.SourceIdentity)
	}

	// Default listing should NOT include this filtered post
	listed, err := repo.ListPosts(ctx, "proj1", repository.PostFilters{})
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected filtered post excluded from default list, got %d posts", len(listed))
	}
}
