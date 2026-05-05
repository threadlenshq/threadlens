package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// AnalysisOptions mirrors the Express analyzer options.
type AnalysisOptions struct {
	MinScore *float64
	DateFrom string
	DateTo   string
}

// postSummary is the per-post shape sent to the AI.
type postSummary struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	Platform   string  `json:"platform"`
	Subreddit  *string `json:"subreddit,omitempty"`
	PainScore  float64 `json:"pain_score"`
	SignalType *string `json:"signal_type,omitempty"`
	Angle      *string `json:"angle,omitempty"`
	Why        string  `json:"why"`
}

// analyzerResult is the JSON shape returned by the AI for report_clustering.
type analyzerResult struct {
	Title      string            `json:"title"`
	Clusters   []json.RawMessage `json:"clusters"`
	Assessment string            `json:"assessment"`
}

// buildSystemPrompt mirrors analyzer.js buildSystemPrompt().
func buildSystemPrompt(hasStarredPosts bool) string {
	step2 := "2. Group posts into 3-7 clusters by theme"
	step3Quotes := "   - Extract 2-4 key quotes that best represent the pain"
	step4 := "4. Provide an overall assessment ranking clusters by opportunity (pain intensity x frequency x willingness-to-pay signals)"

	if hasStarredPosts {
		step2 = "2. Group posts into 3-7 clusters by theme. Priority posts should anchor cluster themes - build clusters around them first, then assign other posts"
		step3Quotes = "   - Extract 2-4 key quotes that best represent the pain. Prefer quotes from priority posts when they represent the pain well"
		step4 = "4. Provide an overall assessment ranking clusters by opportunity (pain intensity x frequency x willingness-to-pay signals). Clusters containing priority posts should receive a ranking boost in opportunity assessment"
	}

	return `You are an expert growth hacker and product researcher analyzing social media posts to identify pain points and suggest niche product opportunities.

You will receive a collection of social media posts that have been scouted from Reddit and Bluesky. Each post includes a title, body, pain score, and signal type (frustration, workaround, or seeking_solution).

Your task:
1. Read all posts carefully and identify recurring pain point themes
` + step2 + `
3. For each cluster:
   - Give it a clear, specific name describing the pain point
   - Count posts and calculate average pain score
   - Count signal breakdown (frustration vs seeking_solution vs workaround)
` + step3Quotes + `
   - List the post IDs that belong to this cluster
   - Suggest a NICHE product angle following Minimalist Entrepreneur principles:
     * Target a specific community with a specific problem
     * Achievable scope for a solo founder
     * NOT broad ideas like "build a habit tracker"
     * Instead: "A daily accountability check-in bot for r/selfimprovement users who quit after day 3"
` + step4 + `
5. Generate a descriptive title for this research report

Respond with ONLY valid JSON in this exact format:
{
  "title": "Report title",
  "clusters": [
    {
      "name": "Pain point name",
      "post_count": 5,
      "avg_pain_score": 7.2,
      "signals": { "frustration": 3, "seeking_solution": 2, "workaround": 0 },
      "key_quotes": ["quote1", "quote2"],
      "post_ids": ["id1", "id2"],
      "product_angle": {
        "idea": "Specific niche product idea",
        "target_niche": "Specific community + specific problem",
        "why": "Why this is an opportunity"
      }
    }
  ],
  "assessment": "Overall assessment text"
}`
}

// dbPost is a minimal post struct for analyzer queries.
type dbPost struct {
	ID         string
	Title      string
	Body       string
	Platform   string
	Subreddit  sql.NullString
	FinalScore float64
	SignalType sql.NullString
	Angle      sql.NullString
	Why        string
	Status     string
}

// RunAnalysis fetches posts and runs AI clustering. Mirrors Express runAnalysis().
// On success it updates research_reports to completed; on failure it marks failed.
// It also triggers council generation in the background after a successful completion.
func RunAnalysis(ctx context.Context, db *sql.DB, aiSvc *ai.Service, repo *repository.Repository, projectID string, reportID int64, opts AnalysisOptions) error {
	// Build dynamic SQL to filter posts.
	query := "SELECT id, title, body, platform, subreddit, final_score, signal_type, angle, why, status FROM posts WHERE project_id = ? AND status != 'excluded'"
	args := []any{projectID}

	if opts.MinScore != nil {
		query += " AND final_score >= ?"
		args = append(args, *opts.MinScore)
	}
	if opts.DateFrom != "" {
		query += " AND found_at >= ?"
		args = append(args, opts.DateFrom)
	}
	if opts.DateTo != "" {
		query += " AND found_at <= ?"
		args = append(args, opts.DateTo)
	}
	query += " ORDER BY final_score DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return repo.MarkReportFailed(ctx, reportID, err.Error())
	}
	defer rows.Close()

	var posts []dbPost
	for rows.Next() {
		var p dbPost
		if err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.Platform, &p.Subreddit, &p.FinalScore, &p.SignalType, &p.Angle, &p.Why, &p.Status); err != nil {
			return repo.MarkReportFailed(ctx, reportID, err.Error())
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return repo.MarkReportFailed(ctx, reportID, err.Error())
	}

	if len(posts) == 0 {
		return repo.MarkReportFailed(ctx, reportID, "No posts to analyze")
	}

	// Link posts to the report.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return repo.MarkReportFailed(ctx, reportID, err.Error())
	}
	stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO report_posts (report_id, post_id) VALUES (?, ?)")
	if err != nil {
		_ = tx.Rollback()
		return repo.MarkReportFailed(ctx, reportID, err.Error())
	}
	for _, p := range posts {
		if _, err := stmt.ExecContext(ctx, reportID, p.ID); err != nil {
			_ = tx.Rollback()
			return repo.MarkReportFailed(ctx, reportID, err.Error())
		}
	}
	_ = stmt.Close()
	if err := tx.Commit(); err != nil {
		return repo.MarkReportFailed(ctx, reportID, err.Error())
	}

	// Build post summaries for AI.
	summaries := make([]postSummary, 0, len(posts))
	for _, p := range posts {
		body := p.Body
		if len(body) > 500 {
			body = body[:500]
		}
		ps := postSummary{
			ID:        p.ID,
			Title:     p.Title,
			Body:      body,
			Platform:  p.Platform,
			PainScore: p.FinalScore,
			Why:       p.Why,
		}
		if p.Subreddit.Valid {
			ps.Subreddit = &p.Subreddit.String
		}
		if p.SignalType.Valid {
			ps.SignalType = &p.SignalType.String
		}
		if p.Angle.Valid {
			ps.Angle = &p.Angle.String
		}
		summaries = append(summaries, ps)
	}

	// Separate starred from other posts (mirrors Express logic).
	var starred, other []postSummary
	starredIDs := map[string]bool{}
	for _, p := range posts {
		if p.Status == "starred" {
			starredIDs[p.ID] = true
		}
	}
	for _, s := range summaries {
		if starredIDs[s.ID] {
			starred = append(starred, s)
		} else {
			other = append(other, s)
		}
	}
	hasStarred := len(starred) > 0 && len(starred) < len(summaries)

	var userMessage string
	if hasStarred {
		starredJSON, _ := json.MarshalIndent(starred, "", "  ")
		otherJSON, _ := json.MarshalIndent(other, "", "  ")
		userMessage = fmt.Sprintf("Analyze these %d scouted posts.\n\n## PRIORITY POSTS (%d posts)\nThe researcher flagged these as particularly important.\n\n%s\n\n## OTHER POSTS (%d posts)\n\n%s",
			len(posts), len(starred), starredJSON, len(other), otherJSON)
	} else {
		allJSON, _ := json.MarshalIndent(summaries, "", "  ")
		userMessage = fmt.Sprintf("Analyze these %d scouted posts:\n\n%s", len(posts), allJSON)
	}

	sysPrompt := buildSystemPrompt(hasStarred)
	raw, modelID, err := aiSvc.GenerateForTask(ctx, "report_clustering", sysPrompt, userMessage)
	if err != nil {
		return repo.MarkReportFailed(ctx, reportID, err.Error())
	}

	// Strip markdown fences.
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	var result analyzerResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return repo.MarkReportFailed(ctx, reportID, "Failed to parse AI response: "+err.Error())
	}

	title := result.Title
	if title == "" {
		title = "Research Report"
	}
	clustersJSON, _ := json.Marshal(result.Clusters)

	if err := repo.CompleteReport(ctx, reportID, title, int64(len(posts)), clustersJSON, result.Assessment, modelID); err != nil {
		return err
	}

	// Kick off council in background – do not block the caller.
	go func() {
		bgCtx := context.Background()
		councilID, err := StartCouncil(bgCtx, db, repo, projectID, reportID)
		if err != nil {
			fmt.Printf("[council] failed to start for report %d: %v\n", reportID, err)
			return
		}
		if err := RunCouncil(bgCtx, db, aiSvc, projectID, reportID, councilID); err != nil {
			fmt.Printf("[council] background run failed for report %d: %v\n", reportID, err)
		}
	}()

	return nil
}
