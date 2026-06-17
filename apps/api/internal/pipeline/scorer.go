package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// defaultScoringRubric mirrors DEFAULT_SCORING_RUBRIC from prompts/defaults.js.
const defaultScoringRubric = `You are a scoring engine that evaluates social media posts for relevance to a product's pain angles.

Score each post from 1-10 using this rubric:
- 9-10: Perfect match - the post describes exactly the pain our product solves, high intent to engage
- 7-8: Strong match - clearly relevant pain or problem, good engagement opportunity
- 5-6: Moderate match - related topic but not a direct pain point, some engagement value
- 3-4: Weak match - tangentially related, likely worth a downgrade or skip
- 1-2: Not relevant - unrelated to our pain angles, do not engage

Return a JSON array only, with no additional text. Each item must have:
{
  "id": "post id",
  "post_score": number 1-10,
  "angle": "which pain angle this matches (or null)",
  "why": "one sentence explanation",
  "engagement_type": "product" or "karma",
  "karma_topic": "topic for karma reply if engagement_type is karma, else null"
}`

// ScoringPost is the input post shape for ScorePosts.
type ScoringPost struct {
	ID          string
	Title       string
	Selftext    string
	Subreddit   string
	Score       int
	NumComments int
}

// ScoredPost is a single AI scoring result.
type ScoredPost struct {
	ID             string  `json:"id"`
	PostScore      float64 `json:"post_score"`
	Angle          *string `json:"angle"`
	Why            string  `json:"why"`
	EngagementType string  `json:"engagement_type"`
	KarmaTopic     *string `json:"karma_topic"`
}

// ScoreStats reports scoring batch outcomes.
type ScoreStats struct {
	TotalBatches  int
	FailedBatches int
	TotalScored   int
	TotalPosts    int
	Errors        []string
}

// ScoreResult is the return value of ScorePosts.
type ScoreResult struct {
	Scores []ScoredPost
	Stats  ScoreStats
}

// codeFencePattern strips markdown code fences from model output.
var codeFencePattern = regexp.MustCompile(`(?s)^\x60\x60\x60(?:json)?\s*\n?(.+?)\n?\x60\x60\x60$`)

// extractJSONArray finds the first complete JSON array in text that may have
// surrounding descriptive content (e.g. "JSON array returned above." from models
// that don't strictly follow instructions). Uses bracket-depth tracking to
// correctly handle nested brackets inside strings.
func extractJSONArray(response string) string {
	cleaned := strings.TrimSpace(response)

	// Strip markdown code fences if the entire response is wrapped in them.
	if m := codeFencePattern.FindStringSubmatch(cleaned); len(m) > 1 {
		cleaned = strings.TrimSpace(m[1])
	}

	// Find the first '[' that could be the start of a JSON array.
	start := strings.Index(cleaned, "[")
	if start < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(cleaned); i++ {
		c := cleaned[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '[' {
			depth++
		} else if c == ']' {
			depth--
			if depth == 0 {
				return cleaned[start : i+1]
			}
		}
	}
	return ""
}

// ScorePosts scores a slice of posts using the AI service and returns all scored results.
// It mirrors scorePosts() from apps/api/server/pipeline/scorer.js.
//
// batchSize 0 or negative uses the default of 15.
// scoringRubric nil uses the built-in default rubric.
// onProgress is called after each concurrency chunk; it may be nil.
func ScorePosts(
	ctx context.Context,
	repo *repository.Repository,
	aiSvc *ai.Service,
	posts []ScoringPost,
	painAngles []string,
	batchSize int,
	scoringRubric *string,
	projectDescription *string,
	onProgress func(scored int, total int),
) (ScoreResult, error) {
	if batchSize <= 0 {
		batchSize = 15
	}

	rubric := defaultScoringRubric
	if scoringRubric != nil && *scoringRubric != "" {
		rubric = *scoringRubric
	}

	anglesText := strings.Join(painAngles, "\n- ")
	descriptionLine := ""
	if projectDescription != nil && *projectDescription != "" {
		descriptionLine = "\n\nProject description: " + *projectDescription
	}
	systemPrompt := rubric + descriptionLine + "\n\nPain angles to match against:\n- " + anglesText

	// Split posts into batches.
	var batches [][]ScoringPost
	for i := 0; i < len(posts); i += batchSize {
		end := i + batchSize
		if end > len(posts) {
			end = len(posts)
		}
		batches = append(batches, posts[i:end])
	}

	allScores := make([]ScoredPost, 0, len(posts))
	var errors []string

	type batchResult struct {
		idx    int
		scores []ScoredPost
		err    error
	}

	const concurrency = 10
	const batchTimeout = 3 * time.Minute

	scoreBatch := func(batchCtx context.Context, batch []ScoringPost) ([]ScoredPost, error) {
		type postData struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Body        string `json:"body"`
			Subreddit   string `json:"subreddit"`
			Score       int    `json:"score"`
			NumComments int    `json:"num_comments"`
		}
		pd := make([]postData, len(batch))
		for i, p := range batch {
			body := p.Selftext
			if len(body) > 500 {
				body = body[:500]
			}
			pd[i] = postData{
				ID:          p.ID,
				Title:       p.Title,
				Body:        body,
				Subreddit:   p.Subreddit,
				Score:       p.Score,
				NumComments: p.NumComments,
			}
		}

		raw, err := json.Marshal(pd)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		userMessage := "Score these posts:\n" + string(raw)

		response, _, err := aiSvc.GenerateForTask(batchCtx, "post_scoring", systemPrompt, userMessage)
		if err != nil {
			return nil, fmt.Errorf("ai: %w", err)
		}

		// Extract a JSON array from the response, which may include
		// markdown fences or descriptive text (e.g. models that add
		// "JSON array returned above" instead of returning clean JSON).
		var parsed []ScoredPost
		jsonFragment := extractJSONArray(response)
		parseErr := json.Unmarshal([]byte(jsonFragment), &parsed)
		if parseErr != nil && jsonFragment == "" {
			// No array found at all — try the raw response as a last resort.
			parseErr = json.Unmarshal([]byte(response), &parsed)
		}
		if parseErr != nil {
			snippet := response
			if len(snippet) > 200 {
				snippet = snippet[:200]
			}
			return nil, fmt.Errorf("parse error - %s", snippet)
		}
		if len(parsed) == 0 {
			return nil, fmt.Errorf("parse error - empty JSON array returned")
		}
		return parsed, nil
	}

	type failedBatch struct {
		batch    []ScoringPost
		batchIdx int
	}
	var failedBatches []failedBatch

	// Process in chunks of concurrency.
	for start := 0; start < len(batches); start += concurrency {
		if ctx.Err() != nil {
			break
		}

		end := start + concurrency
		if end > len(batches) {
			end = len(batches)
		}
		chunk := batches[start:end]

		results := make(chan batchResult, len(chunk))
		for i, batch := range chunk {
			go func(idx int, b []ScoringPost) {
				batchCtx, batchCancel := context.WithTimeout(ctx, batchTimeout)
				defer batchCancel()
				scores, err := scoreBatch(batchCtx, b)
				results <- batchResult{idx: start + idx, scores: scores, err: err}
			}(i, batch)
		}

		for range chunk {
			r := <-results
			if r.err != nil {
				log.Printf("scorer: batch %d/%d failed, will retry - %s", r.idx+1, len(batches), r.err.Error())
				failedBatches = append(failedBatches, failedBatch{batch: batches[r.idx], batchIdx: r.idx})
			} else {
				allScores = append(allScores, r.scores...)
			}
		}

		if onProgress != nil {
			onProgress(len(allScores), len(posts))
		}
	}

	// Retry failed batches once, sequentially.
	// Use a context independent of the pipeline deadline so that deadline
	// expiry during the first pass doesn't cascade-cancel all retries.
	// Each retry gets a generous per-batch timeout of its own.
	const retryBatchTimeout = 3 * time.Minute
	for _, fb := range failedBatches {
		retryCtx, retryCancel := context.WithTimeout(context.Background(), retryBatchTimeout)
		scores, err := scoreBatch(retryCtx, fb.batch)
		retryCancel()
		if err != nil {
			log.Printf("scorer: batch %d/%d failed after retry - %s", fb.batchIdx+1, len(batches), err.Error())
			errors = append(errors, fmt.Sprintf("Batch %d: %s", fb.batchIdx+1, err.Error()))
		} else {
			log.Printf("scorer: batch %d/%d succeeded on retry", fb.batchIdx+1, len(batches))
			allScores = append(allScores, scores...)
		}
	}

	return ScoreResult{
		Scores: allScores,
		Stats: ScoreStats{
			TotalBatches:  len(batches),
			FailedBatches: len(errors),
			TotalScored:   len(allScores),
			TotalPosts:    len(posts),
			Errors:        errors,
		},
	}, nil
}
