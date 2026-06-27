package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// maxPostsPerChunk caps the number of posts sent in a single clustering
// call to keep the payload under the bridge's request size limit and small
// enough that a mid-tier model (claude-cli:sonnet) reliably finishes a chunk
// within report_clustering's 5-minute timeout. A 100-post chunk was observed
// to exceed the deadline on sonnet; 50 leaves comfortable margin.
const maxPostsPerChunk = 50

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

// assessmentFormatGuidance tells the model to emit the overall assessment as
// structured markdown (lead paragraph + numbered list + closing paragraph) so
// the web client's renderAssessment (lib/assessment.js) shows it as a scannable
// list rather than one dense block. \n\n stays literal here on purpose - it is
// instruction text the model should reproduce inside the JSON string value.
const assessmentFormatGuidance = `   - Format the assessment as markdown for readability: open with a 1-2 sentence lead paragraph, then a numbered list with one "N. **Opportunity Name** - why it ranks here" per line, then a brief closing paragraph. Separate the lead, list, and closing paragraphs with blank lines (use \n\n between them in the JSON string), and use **bold** for opportunity names.`

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
` + assessmentFormatGuidance + `
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

// buildChunkSystemPrompt provides a system prompt for clustering a subset of
// posts. It is similar to buildSystemPrompt but tells the AI it is working with
// one chunk of the full dataset.
func buildChunkSystemPrompt(hasStarredPosts bool) string {
	step2 := "2. Group posts into 3-7 clusters by theme"
	step3Quotes := "   - Extract 2-4 key quotes that best represent the pain"
	step4 := "4. Provide an overall assessment ranking clusters by opportunity"

	if hasStarredPosts {
		step2 = "2. Group posts into 3-7 clusters by theme. Priority posts should anchor cluster themes - build clusters around them first, then assign other posts"
		step3Quotes = "   - Extract 2-4 key quotes that best represent the pain. Prefer quotes from priority posts when they represent the pain well"
		step4 = "4. Provide an overall assessment ranking clusters by opportunity. Clusters containing priority posts should receive a ranking boost"
	}

	return `You are an expert growth hacker and product researcher analyzing social media posts. You are working with a SUBSET of the total posts — later a merge step will combine your clusters with clusters from other chunks.

Focus on identifying the most distinct and promising themes WITHIN THESE POSTS ONLY. Do not worry about cross-chunk consistency — the merge step will handle that.

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

Respond with ONLY valid JSON in this exact format:
{
  "title": "Chunk analysis",
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
  "assessment": "Chunk-level assessment"
}`
}

// buildMergeSystemPrompt provides a system prompt for merging cluster results
// from multiple chunks into a unified report.
func buildMergeSystemPrompt() string {
	return `You are an expert growth hacker and product researcher.

You have previously analyzed subsets of a large collection of social media posts. Each subset produced clusters of pain points. Now you need to merge these cluster results into a SINGLE unified report.

Your task:
1. Review all incoming clusters from each chunk
2. MERGE clusters that share the same underlying pain point or theme across different chunks. Combine their post IDs, recalculate post_count (sum), and compute a weighted average pain score.
3. MERGE key_quotes — deduplicate overlapping quotes but keep the best ones from each merged cluster (up to 4 quotes total per merged cluster).
4. MERGE signals by summing counts from merged clusters.
5. For any cluster that is UNIQUE to one chunk (no overlapping theme), keep it as-is.
6. Generate a descriptive report title that captures the overall research findings.
7. Provide an overall assessment ranking all merged clusters by opportunity.
` + assessmentFormatGuidance + `

Respond with ONLY valid JSON in this exact format:
{
  "title": "Report title",
  "clusters": [
    {
      "name": "Pain point name",
      "post_count": 35,
      "avg_pain_score": 7.2,
      "signals": { "frustration": 15, "seeking_solution": 12, "workaround": 8 },
      "key_quotes": ["quote1", "quote2", "quote3"],
      "post_ids": ["id1", "id2", "..."],
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

// chunkSummaries splits a slice of post summaries into chunks of at most
// maxPostsPerChunk each.
func chunkSummaries(summaries []postSummary) [][]postSummary {
	if len(summaries) <= maxPostsPerChunk {
		return [][]postSummary{summaries}
	}
	chunks := make([][]postSummary, 0, (len(summaries)+maxPostsPerChunk-1)/maxPostsPerChunk)
	for i := 0; i < len(summaries); i += maxPostsPerChunk {
		end := i + maxPostsPerChunk
		if end > len(summaries) {
			end = len(summaries)
		}
		chunks = append(chunks, summaries[i:end])
	}
	return chunks
}

// buildChunkUserMessage creates the user message for a single chunk.
func buildChunkUserMessage(chunk []postSummary, chunkNum, totalChunks, totalPosts int, starred, other []postSummary, hasStarred bool) string {
	if hasStarred {
		starredJSON, _ := json.MarshalIndent(starred, "", "  ")
		otherJSON, _ := json.MarshalIndent(other, "", "  ")
		return fmt.Sprintf("CHUNK %d OF %d (total posts across all chunks: %d)\n\nAnalyze these %d scouted posts.\n\n## PRIORITY POSTS (%d posts)\nThe researcher flagged these as particularly important.\n\n%s\n\n## OTHER POSTS (%d posts)\n\n%s",
			chunkNum, totalChunks, totalPosts, len(chunk), len(starred), starredJSON, len(other), otherJSON)
	}
	allJSON, _ := json.MarshalIndent(chunk, "", "  ")
	return fmt.Sprintf("CHUNK %d OF %d (total posts across all chunks: %d)\n\nAnalyze these %d scouted posts:\n\n%s",
		chunkNum, totalChunks, totalPosts, len(chunk), allJSON)
}

// buildMergeUserMessage creates the user message for the merge step from
// collected chunk-level clusters.
func buildMergeUserMessage(allClusters []json.RawMessage, totalPosts int) string {
	clustersJSON, _ := json.MarshalIndent(allClusters, "", "  ")
	return fmt.Sprintf("Total posts analyzed across all chunks: %d\n\nMerge the following cluster results from multiple chunks into one unified report:\n\n%s", totalPosts, clustersJSON)
}

// runChunkedClustering processes all post summaries in chunks, collecting
// clusters from each, then merges overlapping clusters into a unified result.
// Returns (rawJSON, modelID, error).
func runChunkedClustering(ctx context.Context, aiSvc *ai.Service, summaries []postSummary, starredIDs map[string]bool) (string, string, error) {
	hasStarred := len(starredIDs) > 0 && len(starredIDs) < len(summaries)

	// Separate starred and other within each chunk for priority handling.
	var starred, other []postSummary
	for _, s := range summaries {
		if starredIDs[s.ID] {
			starred = append(starred, s)
		} else {
			other = append(other, s)
		}
	}

	chunks := chunkSummaries(summaries)
	sysPrompt := buildChunkSystemPrompt(hasStarred)

	var allClusters []json.RawMessage
	var lastModelID string
	var skipped int

	for i, chunk := range chunks {
		chunkStarred, chunkOther := filterStarred(chunk, starredIDs)
		userMsg := buildChunkUserMessage(chunk, i+1, len(chunks), len(summaries), chunkStarred, chunkOther, hasStarred && len(chunkStarred) > 0 && len(chunkStarred) < len(chunk))

		// A single chunk returning non-JSON (e.g. a fallback model emitting prose)
		// must not abort the whole report. Try once, retry once on a parse failure,
		// then skip the chunk and continue so the remaining chunks still contribute.
		var result analyzerResult
		var parsed bool
		for attempt := 1; attempt <= 2; attempt++ {
			raw, modelID, err := aiSvc.GenerateForTask(ctx, "report_clustering", sysPrompt, userMsg)
			if err != nil {
				return "", "", fmt.Errorf("chunk %d/%d clustering failed: %w", i+1, len(chunks), err)
			}
			lastModelID = modelID

			cleaned := stripMarkdownFences(raw)
			result = analyzerResult{}
			if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
				log.Printf("[analyzer] chunk %d/%d parse failed (attempt %d/2) with %s: %v", i+1, len(chunks), attempt, modelID, err)
				continue
			}
			parsed = true
			break
		}
		if !parsed {
			skipped++
			log.Printf("[analyzer] chunk %d/%d skipped after 2 failed parse attempts", i+1, len(chunks))
			continue
		}
		allClusters = append(allClusters, result.Clusters...)
	}

	if skipped == len(chunks) {
		return "", "", fmt.Errorf("all %d chunks failed to parse", len(chunks))
	}
	if skipped > 0 {
		log.Printf("[analyzer] proceeding with %d/%d chunks (%d skipped due to parse failures)", len(chunks)-skipped, len(chunks), skipped)
	}

	// Merge overlapping clusters from all chunks.
	mergeSysPrompt := buildMergeSystemPrompt()
	mergeUserMsg := buildMergeUserMessage(allClusters, len(summaries))

	mergedRaw, modelID, err := aiSvc.GenerateForTask(ctx, "report_clustering", mergeSysPrompt, mergeUserMsg)
	if err != nil {
		return "", "", fmt.Errorf("cluster merge failed: %w", err)
	}
	if modelID != "" {
		lastModelID = modelID
	}

	return mergedRaw, lastModelID, nil
}

// filterStarred splits a chunk into starred and other based on starredIDs.
func filterStarred(chunk []postSummary, starredIDs map[string]bool) (starred, other []postSummary) {
	for _, s := range chunk {
		if starredIDs[s.ID] {
			starred = append(starred, s)
		} else {
			other = append(other, s)
		}
	}
	return
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

	// Identify starred posts.
	starredIDs := map[string]bool{}
	for _, p := range posts {
		if p.Status == "starred" {
			starredIDs[p.ID] = true
		}
	}

	var raw string
	var modelID string

	if len(summaries) > maxPostsPerChunk {
		// Too many posts for a single request: chunk, cluster each chunk,
		// then merge overlapping clusters into the final report.
		raw, modelID, err = runChunkedClustering(ctx, aiSvc, summaries, starredIDs)
	} else {
		// Fast path: send everything in one call.
		var starred, other []postSummary
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
		raw, modelID, err = aiSvc.GenerateForTask(ctx, "report_clustering", sysPrompt, userMessage)
	}
	if err != nil {
		return repo.MarkReportFailed(ctx, reportID, err.Error())
	}

	cleaned := stripMarkdownFences(raw)

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
