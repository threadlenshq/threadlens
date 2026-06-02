package repository

import (
	"context"
	"testing"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/testhelpers"
)

func newFilteringRepo(t *testing.T) *Repository {
	t.Helper()
	return New(testhelpers.OpenTestDB(t))
}

func seedFilteringProject(t *testing.T, repo *Repository, id string) {
	t.Helper()
	_, err := repo.DB.Exec(
		`INSERT INTO projects (id, name, mode, created_at, updated_at) VALUES (?, ?, 'research', datetime('now'), datetime('now'))`,
		id, id,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilteringRepositoryTrustRecordsAreUnique(t *testing.T) {
	repo := newFilteringRepo(t)
	ctx := context.Background()
	seedFilteringProject(t, repo, "p1")

	input := domain.TrustRecord{
		ProjectID:  "p1",
		Platform:   "reddit",
		TrustType:  domain.TrustTypeSource,
		SourceKind: "reddit_author",
		SourceKey:  "alice",
		Reason:     "false positive",
	}

	first, err := repo.CreateTrustRecord(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.CreateTrustRecord(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected existing trust record to be returned, got IDs %d and %d", first.ID, second.ID)
	}
}

func TestFilteringRepositoryRestorePostPreservesScoreAndStatus(t *testing.T) {
	repo := newFilteringRepo(t)
	ctx := context.Background()
	seedFilteringProject(t, repo, "p1")

	_, err := repo.DB.Exec(`INSERT INTO posts (
		id, project_id, platform, title, body, author, url,
		post_score, final_score, engagement_type, status,
		filter_state, filter_reason, filter_reasons_json, filter_explanation, filter_source,
		filtered_at, source_identity_json, found_at, scouted_at
	) VALUES (
		'post1', 'p1', 'reddit', 'T', 'B', 'alice', 'https://reddit.com/r/x',
		7, 7, 'karma', 'starred',
		'filtered', 'spam', '["spam"]', 'x', 'rules',
		datetime('now'), '{"reddit_author":"alice"}', datetime('now'), datetime('now')
	)`)
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.RestoreFindingVisibility(ctx, "p1", domain.FindingTypePost, "post1", "owner restored"); err != nil {
		t.Fatal(err)
	}

	post, err := repo.GetPost(ctx, "p1", "post1")
	if err != nil {
		t.Fatal(err)
	}
	if post.FilterState != domain.FilterStateVisible {
		t.Fatalf("expected filter_state=visible, got %s", post.FilterState)
	}
	if post.Status != "starred" {
		t.Fatalf("expected status=starred, got %s", post.Status)
	}
	if post.FinalScore != 7 {
		t.Fatalf("expected final_score=7, got %v", post.FinalScore)
	}
	if post.RecoveryNote == nil || *post.RecoveryNote != "owner restored" {
		t.Fatalf("expected recovery_note='owner restored', got %v", post.RecoveryNote)
	}
}

func TestFilteringRepositoryFilterJobLifecycle(t *testing.T) {
	repo := newFilteringRepo(t)
	ctx := context.Background()
	seedFilteringProject(t, repo, "p1")

	job, err := repo.CreateFilterJob(ctx, "p1", domain.FilterJobScopeSelectedVisiblePosts,
		[]domain.FilterJobTarget{{FindingType: domain.FindingTypePost, ID: "post1"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != domain.FilterJobStatusRunning {
		t.Fatalf("expected status=running, got %s", job.Status)
	}
	if len(job.Targets) != 1 || job.Targets[0].ID != "post1" {
		t.Fatalf("unexpected targets: %v", job.Targets)
	}

	result := domain.FilterJobResult{
		Filtered: 1,
		Errors:   map[string]string{},
	}
	job, err = repo.CompleteFilterJob(ctx, "p1", job.ID, result)
	if err != nil {
		t.Fatal(err)
	}
	if job.Result == nil || job.Result.Filtered != 1 {
		t.Fatalf("expected result.Filtered=1, got %v", job.Result)
	}
	if job.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
	if job.Status != domain.FilterJobStatusCompleted {
		t.Fatalf("expected status=completed, got %s", job.Status)
	}
}

func TestFilteringRepositoryListFilteredFindings(t *testing.T) {
	repo := newFilteringRepo(t)
	ctx := context.Background()
	seedFilteringProject(t, repo, "p1")

	_, err := repo.DB.Exec(`INSERT INTO posts (
		id, project_id, platform, title, body, author, url,
		post_score, final_score, engagement_type, status,
		filter_state, filter_reason, filter_reasons_json, filter_explanation, filter_source, filter_signature,
		filtered_at, source_identity_json, found_at, scouted_at
	) VALUES (
		'fp1', 'p1', 'reddit', 'Spam Post', 'spam', 'spammer', 'https://reddit.com/r/x',
		1, 1, 'karma', 'new',
		'filtered', 'spam', '["spam"]', 'matched spam rules', 'rules', 'filter:abc123',
		datetime('now'), '{"reddit_author":"spammer"}', datetime('now'), datetime('now')
	)`)
	if err != nil {
		t.Fatal(err)
	}

	result, err := repo.ListFilteredFindings(ctx, "p1", FilteredFindingFilters{}, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 filtered finding, got %d", len(result.Items))
	}
	f := result.Items[0]
	if f.FindingType != domain.FindingTypePost {
		t.Fatalf("expected finding_type=post, got %s", f.FindingType)
	}
	if f.FilterState != domain.FilterStateFiltered {
		t.Fatalf("expected filter_state=filtered, got %s", f.FilterState)
	}
}

func TestFilteringRepositoryApplyPostFilterDecision(t *testing.T) {
	repo := newFilteringRepo(t)
	ctx := context.Background()
	seedFilteringProject(t, repo, "p1")

	_, err := repo.DB.Exec(`INSERT INTO posts (
		id, project_id, platform, title, body, author, url,
		post_score, final_score, engagement_type, status,
		source_identity_json, found_at, scouted_at
	) VALUES (
		'pp1', 'p1', 'reddit', 'Some Post', 'body', 'author', 'https://reddit.com/r/x',
		5, 5, 'karma', 'new',
		'{}', datetime('now'), datetime('now')
	)`)
	if err != nil {
		t.Fatal(err)
	}

	conf := 0.9
	decision := domain.FilterDecision{
		State:       domain.FilterStateFiltered,
		Reason:      domain.FilterReasonSpam,
		Reasons:     []string{domain.FilterReasonSpam},
		Explanation: "test",
		Confidence:  &conf,
		Source:      domain.FilterSourceRules,
		Signature:   "filter:testsig",
	}
	if err := repo.ApplyPostFilterDecision(ctx, "p1", "pp1", decision, nil); err != nil {
		t.Fatal(err)
	}

	post, err := repo.GetPost(ctx, "p1", "pp1")
	if err != nil {
		t.Fatal(err)
	}
	if post.FilterState != domain.FilterStateFiltered {
		t.Fatalf("expected filter_state=filtered, got %s", post.FilterState)
	}
}

func TestFilteringRepositoryFailFilterJob(t *testing.T) {
	repo := newFilteringRepo(t)
	ctx := context.Background()
	seedFilteringProject(t, repo, "p1")

	job, err := repo.CreateFilterJob(ctx, "p1", domain.FilterJobScopeSelectedFiltered,
		[]domain.FilterJobTarget{{FindingType: domain.FindingTypePost, ID: "x"}},
	)
	if err != nil {
		t.Fatal(err)
	}

	job, err = repo.FailFilterJob(ctx, "p1", job.ID, "something went wrong")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != domain.FilterJobStatusFailed {
		t.Fatalf("expected status=failed, got %s", job.Status)
	}
	if job.Error == nil || *job.Error != "something went wrong" {
		t.Fatalf("expected error message, got %v", job.Error)
	}
}
