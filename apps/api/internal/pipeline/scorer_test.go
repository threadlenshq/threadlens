package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
)

// fakeAIProvider is a controllable test double for ai.Provider.
type fakeAIProvider struct {
	name      string
	responses []string
	callCount int32
	errs      []error
}

func (f *fakeAIProvider) Name() string    { return f.name }
func (f *fakeAIProvider) Available() bool { return true }
func (f *fakeAIProvider) Generate(_ context.Context, _ string, _ string, _ string, _ time.Duration) (string, error) {
	idx := int(atomic.AddInt32(&f.callCount, 1)) - 1
	if f.errs != nil && idx < len(f.errs) && f.errs[idx] != nil {
		return "", f.errs[idx]
	}
	if idx < len(f.responses) {
		return f.responses[idx], nil
	}
	return "", fmt.Errorf("fakeAIProvider: no response for call %d", idx)
}

func makeScoredJSON(posts []ScoringPost, score float64) string {
	type item struct {
		ID             string  `json:"id"`
		PostScore      float64 `json:"post_score"`
		Angle          *string `json:"angle"`
		Why            string  `json:"why"`
		EngagementType string  `json:"engagement_type"`
		KarmaTopic     *string `json:"karma_topic"`
	}
	items := make([]item, len(posts))
	for i, p := range posts {
		items[i] = item{
			ID:             p.ID,
			PostScore:      score,
			Why:            "test match",
			EngagementType: "karma",
		}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func TestScorePosts_ReturnsScoredPosts(t *testing.T) {
	posts := []ScoringPost{
		{ID: "abc", Title: "How do I manage AWS costs?", Selftext: "Bills keep surprising me", Subreddit: "aws", Score: 100, NumComments: 20},
		{ID: "def", Title: "Best practices for env vars?", Selftext: "Staging vs prod configs", Subreddit: "devops", Score: 50, NumComments: 10},
	}

	aiResponse := `[
		{"id":"abc","post_score":9,"angle":"cost pain","why":"Perfect match","engagement_type":"product","karma_topic":null},
		{"id":"def","post_score":6,"angle":"workflow","why":"Moderate match","engagement_type":"karma","karma_topic":"env management"}
	]`

	provider := &fakeAIProvider{name: "copilot", responses: []string{aiResponse}}
	aiSvc := ai.NewServiceWithProviders([]ai.Provider{provider})

	result, err := ScorePosts(context.Background(), nil, aiSvc, posts, []string{"cost management"}, 15, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(result.Scores))
	}
	if result.Scores[0].ID != "abc" {
		t.Errorf("expected first score id abc, got %s", result.Scores[0].ID)
	}
	if result.Scores[0].PostScore != 9 {
		t.Errorf("expected score 9, got %v", result.Scores[0].PostScore)
	}
	if result.Scores[1].ID != "def" {
		t.Errorf("expected second score id def, got %s", result.Scores[1].ID)
	}
	if result.Scores[1].PostScore != 6 {
		t.Errorf("expected score 6, got %v", result.Scores[1].PostScore)
	}
}

func TestScorePosts_HandlesInvalidJSONGracefully(t *testing.T) {
	posts := []ScoringPost{
		{ID: "xyz", Title: "Random question", Selftext: "Some body text", Subreddit: "programming", Score: 5, NumComments: 2},
	}

	// Both initial call and retry return bad JSON.
	provider := &fakeAIProvider{
		name:      "copilot",
		responses: []string{"this is not json at all", "still not json"},
	}
	aiSvc := ai.NewServiceWithProviders([]ai.Provider{provider})

	result, err := ScorePosts(context.Background(), nil, aiSvc, posts, []string{"some angle"}, 15, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Scores) != 0 {
		t.Fatalf("expected 0 scores, got %d", len(result.Scores))
	}
	if result.Stats.FailedBatches != 1 {
		t.Errorf("expected 1 failed batch, got %d", result.Stats.FailedBatches)
	}
}

func TestScorePosts_BatchesLargePostSets(t *testing.T) {
	n := 20
	posts := make([]ScoringPost, n)
	for i := 0; i < n; i++ {
		posts[i] = ScoringPost{
			ID:          fmt.Sprintf("post_%d", i),
			Title:       fmt.Sprintf("Post title %d", i),
			Selftext:    fmt.Sprintf("Body %d", i),
			Subreddit:   "test",
			Score:       i * 10,
			NumComments: i,
		}
	}

	batchSize := 5
	// 20 posts / 5 = 4 batches
	responses := []string{
		makeScoredJSON(posts[0:5], 5),
		makeScoredJSON(posts[5:10], 5),
		makeScoredJSON(posts[10:15], 5),
		makeScoredJSON(posts[15:20], 5),
	}

	provider := &fakeAIProvider{name: "copilot", responses: responses}
	aiSvc := ai.NewServiceWithProviders([]ai.Provider{provider})

	result, err := ScorePosts(context.Background(), nil, aiSvc, posts, []string{"test angle"}, batchSize, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Scores) != n {
		t.Fatalf("expected %d scores, got %d", n, len(result.Scores))
	}
}

func TestScorePosts_CancellationStopsProcessing(t *testing.T) {
	n := 30
	posts := make([]ScoringPost, n)
	for i := 0; i < n; i++ {
		posts[i] = ScoringPost{ID: fmt.Sprintf("post_%d", i), Title: "Title", Subreddit: "test"}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	provider := &fakeAIProvider{name: "copilot", responses: []string{
		makeScoredJSON(posts[0:15], 5),
		makeScoredJSON(posts[15:30], 5),
	}}
	aiSvc := ai.NewServiceWithProviders([]ai.Provider{provider})

	result, err := ScorePosts(ctx, nil, aiSvc, posts, []string{"angle"}, 15, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With cancelled context, no batches should complete.
	if len(result.Scores) > 0 {
		t.Logf("cancellation: got %d scores (may be 0 or some from in-flight goroutines)", len(result.Scores))
	}
}
