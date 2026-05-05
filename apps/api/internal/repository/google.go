package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

// GoogleKeywordSummary represents a row from google_keyword_summaries.
type GoogleKeywordSummary struct {
	ID                 int64           `json:"id"`
	RunID              int64           `json:"run_id"`
	ProjectID          string          `json:"project_id"`
	RootKeyword        string          `json:"root_keyword"`
	TotalResults       int64           `json:"total_results"`
	RelevantResults    int64           `json:"relevant_results"`
	OutreachCandidates int64           `json:"outreach_candidates"`
	AvgRelevanceScore  *float64        `json:"avg_relevance_score"`
	AvgConfidenceScore *float64        `json:"avg_confidence_score"`
	ResultTypes        json.RawMessage `json:"result_types"`
	ContentTypes       json.RawMessage `json:"content_types"`
	IntentTypes        json.RawMessage `json:"intent_types"`
	Recommendation     json.RawMessage `json:"recommendation"`
	CreatedAt          string          `json:"created_at"`
}

// GoogleResult represents a row from google_results.
type GoogleResult struct {
	ID                   int64           `json:"id"`
	RunID                int64           `json:"run_id"`
	ProjectID            string          `json:"project_id"`
	RootKeyword          string          `json:"root_keyword"`
	Query                string          `json:"query"`
	Title                string          `json:"title"`
	URL                  string          `json:"url"`
	DisplayURL           string          `json:"display_url"`
	Snippet              string          `json:"snippet"`
	Rank                 *int64          `json:"rank"`
	ResultType           string          `json:"result_type"`
	Domain               string          `json:"domain"`
	PublishedAt          *string         `json:"published_at"`
	Author               string          `json:"author"`
	ContentType          string          `json:"content_type"`
	IntentType           string          `json:"intent_type"`
	RelevanceFit         string          `json:"relevance_fit"`
	RelevanceScore       *float64        `json:"relevance_score"`
	ConfidenceScore      *float64        `json:"confidence_score"`
	OpportunityTypes     json.RawMessage `json:"opportunity_types"`
	KeepgoingFitReasons  json.RawMessage `json:"keepgoing_fit_reasons"`
	Disqualifiers        json.RawMessage `json:"disqualifiers"`
	Summary              string          `json:"summary"`
	ActionRecommendation string          `json:"action_recommendation"`
	OutreachCandidate    int64           `json:"outreach_candidate"`
	CanonicalURL         string          `json:"canonical_url"`
	ContentHash          string          `json:"content_hash"`
}

const googleReportBaseQuery = `
	SELECT
		gr.id, gr.run_id, gr.project_id,
		gr.executive_summary_json, gr.keyword_summary_json,
		gr.opportunities_json, gr.risks_json, gr.next_actions_json,
		gr.created_at, gr.updated_at,
		sr.platform, sr.status, sr.started_at, sr.completed_at,
		sr.posts_checked, sr.posts_found, sr.step, sr.error, sr.warnings
	FROM google_reports gr
	JOIN scout_runs sr ON sr.id = gr.run_id
	WHERE gr.project_id = ?
`

// ListGoogleReports returns all google reports for a project ordered newest first.
func (r *Repository) ListGoogleReports(ctx context.Context, projectID string) ([]domain.GoogleReport, error) {
	rows, err := r.DB.QueryContext(ctx, googleReportBaseQuery+` ORDER BY gr.created_at DESC, gr.id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []domain.GoogleReport
	for rows.Next() {
		rep, err := scanGoogleReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if reports == nil {
		reports = []domain.GoogleReport{}
	}
	return reports, nil
}

// LatestGoogleReport returns the most recently created google report for a project.
func (r *Repository) LatestGoogleReport(ctx context.Context, projectID string) (domain.GoogleReport, error) {
	row := r.DB.QueryRowContext(ctx, googleReportBaseQuery+` ORDER BY gr.created_at DESC, gr.id DESC LIMIT 1`, projectID)
	rep, err := scanGoogleReportRow(row)
	if err == sql.ErrNoRows {
		return domain.GoogleReport{}, fmt.Errorf("%w: Google report not found", ErrNotFound)
	}
	return rep, err
}

// GetGoogleReport returns a single google report by project and report ID.
func (r *Repository) GetGoogleReport(ctx context.Context, projectID string, reportID int64) (domain.GoogleReport, error) {
	row := r.DB.QueryRowContext(ctx, googleReportBaseQuery+` AND gr.id = ? LIMIT 1`, projectID, reportID)
	rep, err := scanGoogleReportRow(row)
	if err == sql.ErrNoRows {
		return domain.GoogleReport{}, fmt.Errorf("%w: Google report not found", ErrNotFound)
	}
	return rep, err
}

// ListGoogleKeywordSummaries returns all keyword summaries for a project's run.
func (r *Repository) ListGoogleKeywordSummaries(ctx context.Context, projectID string, runID int64) ([]GoogleKeywordSummary, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, run_id, project_id, root_keyword,
		       total_results, relevant_results, outreach_candidates,
		       avg_relevance_score, avg_confidence_score,
		       result_types_json, content_types_json, intent_types_json, recommendation_json,
		       created_at
		FROM google_keyword_summaries
		WHERE project_id = ? AND run_id = ?
		ORDER BY total_results DESC, root_keyword ASC
	`, projectID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []GoogleKeywordSummary
	for rows.Next() {
		var s GoogleKeywordSummary
		var resultTypesStr, contentTypesStr, intentTypesStr, recommendationStr string
		var avgRel, avgConf sql.NullFloat64
		err := rows.Scan(
			&s.ID, &s.RunID, &s.ProjectID, &s.RootKeyword,
			&s.TotalResults, &s.RelevantResults, &s.OutreachCandidates,
			&avgRel, &avgConf,
			&resultTypesStr, &contentTypesStr, &intentTypesStr, &recommendationStr,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if avgRel.Valid {
			s.AvgRelevanceScore = &avgRel.Float64
		}
		if avgConf.Valid {
			s.AvgConfidenceScore = &avgConf.Float64
		}
		s.ResultTypes = parseJSONOrDefault(resultTypesStr, `{}`)
		s.ContentTypes = parseJSONOrDefault(contentTypesStr, `{}`)
		s.IntentTypes = parseJSONOrDefault(intentTypesStr, `{}`)
		s.Recommendation = parseJSONOrDefault(recommendationStr, `{}`)
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if summaries == nil {
		summaries = []GoogleKeywordSummary{}
	}
	return summaries, nil
}

// ListGoogleResults returns all results for a project's run (excluding page_text and mentioned_products).
func (r *Repository) ListGoogleResults(ctx context.Context, projectID string, runID int64) ([]GoogleResult, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, run_id, project_id, root_keyword, query,
		       title, url, display_url, snippet, rank,
		       result_type, domain, published_at, author,
		       content_type, intent_type, relevance_fit,
		       relevance_score, confidence_score,
		       opportunity_types, keepgoing_fit_reasons, disqualifiers,
		       summary, action_recommendation, outreach_candidate,
		       canonical_url, content_hash
		FROM google_results
		WHERE project_id = ? AND run_id = ?
	`, projectID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []GoogleResult
	for rows.Next() {
		var res GoogleResult
		var rank sql.NullInt64
		var publishedAt sql.NullString
		var relScore, confScore sql.NullFloat64
		var oppTypes, fitReasons, disqualifiers string
		err := rows.Scan(
			&res.ID, &res.RunID, &res.ProjectID, &res.RootKeyword, &res.Query,
			&res.Title, &res.URL, &res.DisplayURL, &res.Snippet, &rank,
			&res.ResultType, &res.Domain, &publishedAt, &res.Author,
			&res.ContentType, &res.IntentType, &res.RelevanceFit,
			&relScore, &confScore,
			&oppTypes, &fitReasons, &disqualifiers,
			&res.Summary, &res.ActionRecommendation, &res.OutreachCandidate,
			&res.CanonicalURL, &res.ContentHash,
		)
		if err != nil {
			return nil, err
		}
		if rank.Valid {
			res.Rank = &rank.Int64
		}
		if publishedAt.Valid {
			res.PublishedAt = &publishedAt.String
		}
		if relScore.Valid {
			res.RelevanceScore = &relScore.Float64
		}
		if confScore.Valid {
			res.ConfidenceScore = &confScore.Float64
		}
		res.OpportunityTypes = parseJSONOrDefault(oppTypes, `[]`)
		res.KeepgoingFitReasons = parseJSONOrDefault(fitReasons, `[]`)
		res.Disqualifiers = parseJSONOrDefault(disqualifiers, `[]`)
		results = append(results, res)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if results == nil {
		results = []GoogleResult{}
	}
	return results, nil
}

// GoogleRunResult holds a single analyzed result row for persistence.
type GoogleRunResult struct {
	RootKeyword          string
	Query                string
	Title                string
	URL                  string
	DisplayURL           string
	Snippet              string
	Rank                 *float64
	ResultType           string
	Domain               string
	PublishedAt          *string
	Author               string
	PageText             string
	ContentType          string
	IntentType           string
	RelevanceFit         string
	RelevanceScore       *float64
	ConfidenceScore      *float64
	OpportunityTypes     []string
	FitReasons           []string
	Disqualifiers        []string
	Summary              string
	ActionRecommendation string
	OutreachCandidate    int
	CanonicalURL         string
	ContentHash          string
	MentionedProducts    []string
	Sources              []string
}

// GoogleRunKeywordSummary holds keyword summary data for persistence.
type GoogleRunKeywordSummary struct {
	RootKeyword        string
	TotalResults       int
	RelevantResults    int
	OutreachCandidates int
	AvgRelevanceScore  *float64
	AvgConfidenceScore *float64
	ResultTypesJSON    interface{}
	ContentTypesJSON   interface{}
	IntentTypesJSON    interface{}
	RecommendationJSON interface{}
}

// GoogleRunDomainStat holds domain stat data for persistence.
type GoogleRunDomainStat struct {
	Domain                 string
	ResultCount            int
	RelevantCount          int
	OutreachCandidateCount int
	AvgRelevanceScore      *float64
	AvgConfidenceScore     *float64
	TopIntentTypesJSON     interface{}
	TopContentTypesJSON    interface{}
}

// GoogleRunReport holds the report JSON blobs for persistence.
type GoogleRunReport struct {
	ExecutiveSummaryJSON interface{}
	KeywordSummaryJSON   interface{}
	OpportunitiesJSON    interface{}
	RisksJSON            interface{}
	NextActionsJSON      interface{}
}

func safeJSON(v interface{}, fallback interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(fallback)
	}
	return string(b)
}

// ReplaceGoogleRunData deletes existing rows for the run and inserts fresh ones
// in a transaction, mirroring the Express db.transaction() block.
func (r *Repository) ReplaceGoogleRunData(
	ctx context.Context,
	runID int64,
	projectID string,
	results []GoogleRunResult,
	keywordSummaries []GoogleRunKeywordSummary,
	domainStats []GoogleRunDomainStat,
	report GoogleRunReport,
) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ReplaceGoogleRunData: begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, table := range []string{"google_results", "google_keyword_summaries", "google_domain_stats", "google_reports"} {
		if _, err = tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE run_id = ?", runID); err != nil {
			return fmt.Errorf("ReplaceGoogleRunData: delete %s: %w", table, err)
		}
	}

	for _, res := range results {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO google_results (
				run_id, project_id, root_keyword, query, title, url, display_url, snippet, rank, result_type,
				domain, published_at, author, page_text, content_type, intent_type, relevance_fit, relevance_score,
				confidence_score, opportunity_types, keepgoing_fit_reasons, disqualifiers, summary, action_recommendation,
				outreach_candidate, canonical_url, content_hash, mentioned_products
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?,
				?, ?, ?, ?
			)`,
			runID, projectID,
			res.RootKeyword, res.Query,
			res.Title, res.URL, res.DisplayURL, res.Snippet, res.Rank, res.ResultType,
			res.Domain, res.PublishedAt, res.Author, res.PageText,
			res.ContentType, res.IntentType, res.RelevanceFit, res.RelevanceScore,
			res.ConfidenceScore,
			safeJSON(res.OpportunityTypes, []string{}),
			safeJSON(res.FitReasons, []string{}),
			safeJSON(res.Disqualifiers, []string{}),
			res.Summary, res.ActionRecommendation, res.OutreachCandidate,
			res.CanonicalURL, res.ContentHash,
			safeJSON(res.MentionedProducts, []string{}),
		)
		if err != nil {
			return fmt.Errorf("ReplaceGoogleRunData: insert google_results: %w", err)
		}
	}

	for _, ks := range keywordSummaries {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO google_keyword_summaries (
				run_id, project_id, root_keyword, total_results, relevant_results, outreach_candidates,
				avg_relevance_score, avg_confidence_score, result_types_json, content_types_json, intent_types_json,
				recommendation_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			runID, projectID,
			ks.RootKeyword, ks.TotalResults, ks.RelevantResults, ks.OutreachCandidates,
			ks.AvgRelevanceScore, ks.AvgConfidenceScore,
			safeJSON(ks.ResultTypesJSON, map[string]int{}),
			safeJSON(ks.ContentTypesJSON, map[string]int{}),
			safeJSON(ks.IntentTypesJSON, map[string]int{}),
			safeJSON(ks.RecommendationJSON, map[string]int{}),
		)
		if err != nil {
			return fmt.Errorf("ReplaceGoogleRunData: insert google_keyword_summaries: %w", err)
		}
	}

	for _, ds := range domainStats {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO google_domain_stats (
				run_id, project_id, domain, result_count, relevant_count, outreach_candidate_count, avg_relevance_score,
				avg_confidence_score, top_intent_types_json, top_content_types_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			runID, projectID,
			ds.Domain, ds.ResultCount, ds.RelevantCount, ds.OutreachCandidateCount,
			ds.AvgRelevanceScore, ds.AvgConfidenceScore,
			safeJSON(ds.TopIntentTypesJSON, []interface{}{}),
			safeJSON(ds.TopContentTypesJSON, []interface{}{}),
		)
		if err != nil {
			return fmt.Errorf("ReplaceGoogleRunData: insert google_domain_stats: %w", err)
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO google_reports (
			run_id, project_id, executive_summary_json, keyword_summary_json, opportunities_json,
			risks_json, next_actions_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		runID, projectID,
		safeJSON(report.ExecutiveSummaryJSON, map[string]interface{}{}),
		safeJSON(report.KeywordSummaryJSON, []interface{}{}),
		safeJSON(report.OpportunitiesJSON, []interface{}{}),
		safeJSON(report.RisksJSON, []interface{}{}),
		safeJSON(report.NextActionsJSON, []interface{}{}),
	)
	if err != nil {
		return fmt.Errorf("ReplaceGoogleRunData: insert google_reports: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("ReplaceGoogleRunData: commit: %w", err)
	}
	return nil
}

// parseJSONOrDefault returns json.RawMessage for valid JSON, or the fallback string.
func parseJSONOrDefault(s, fallback string) json.RawMessage {
	if json.Valid([]byte(s)) {
		return json.RawMessage(s)
	}
	return json.RawMessage(fallback)
}

func scanGoogleReport(rows *sql.Rows) (domain.GoogleReport, error) {
	var rep domain.GoogleReport
	var execSumStr, kwSumStr, oppStr, risksStr, nextActStr string
	var run domain.ScoutRun
	var completedAt, step, runError, warnings sql.NullString

	err := rows.Scan(
		&rep.ID, &rep.RunID, &rep.ProjectID,
		&execSumStr, &kwSumStr,
		&oppStr, &risksStr, &nextActStr,
		&rep.CreatedAt, &rep.UpdatedAt,
		&run.Platform, &run.Status, &run.StartedAt, &completedAt,
		&run.PostsChecked, &run.PostsFound, &step, &runError, &warnings,
	)
	if err != nil {
		return domain.GoogleReport{}, err
	}
	run.ID = rep.RunID
	if completedAt.Valid {
		run.CompletedAt = &completedAt.String
	}
	if step.Valid {
		run.Step = &step.String
	}
	if runError.Valid {
		run.Error = &runError.String
	}
	if warnings.Valid {
		run.Warnings = &warnings.String
	}
	rep.ExecutiveSummary = parseJSONOrDefault(execSumStr, `{}`)
	rep.KeywordSummary = parseJSONOrDefault(kwSumStr, `[]`)
	rep.Opportunities = parseJSONOrDefault(oppStr, `[]`)
	rep.Risks = parseJSONOrDefault(risksStr, `[]`)
	rep.NextActions = parseJSONOrDefault(nextActStr, `[]`)
	rep.Run = &run
	return rep, nil
}

func scanGoogleReportRow(row *sql.Row) (domain.GoogleReport, error) {
	var rep domain.GoogleReport
	var execSumStr, kwSumStr, oppStr, risksStr, nextActStr string
	var run domain.ScoutRun
	var completedAt, step, runError, warnings sql.NullString

	err := row.Scan(
		&rep.ID, &rep.RunID, &rep.ProjectID,
		&execSumStr, &kwSumStr,
		&oppStr, &risksStr, &nextActStr,
		&rep.CreatedAt, &rep.UpdatedAt,
		&run.Platform, &run.Status, &run.StartedAt, &completedAt,
		&run.PostsChecked, &run.PostsFound, &step, &runError, &warnings,
	)
	if err != nil {
		return domain.GoogleReport{}, err
	}
	run.ID = rep.RunID
	if completedAt.Valid {
		run.CompletedAt = &completedAt.String
	}
	if step.Valid {
		run.Step = &step.String
	}
	if runError.Valid {
		run.Error = &runError.String
	}
	if warnings.Valid {
		run.Warnings = &warnings.String
	}
	rep.ExecutiveSummary = parseJSONOrDefault(execSumStr, `{}`)
	rep.KeywordSummary = parseJSONOrDefault(kwSumStr, `[]`)
	rep.Opportunities = parseJSONOrDefault(oppStr, `[]`)
	rep.Risks = parseJSONOrDefault(risksStr, `[]`)
	rep.NextActions = parseJSONOrDefault(nextActStr, `[]`)
	rep.Run = &run
	return rep, nil
}
