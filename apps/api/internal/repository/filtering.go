package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

// FilteredFindingFilters holds optional filter values for listing filtered findings.
type FilteredFindingFilters struct {
	Platform      string
	Reason        string
	Source        string
	AIUsed        *bool
	MinConfidence *float64
	MaxConfidence *float64
}

// ListTrustRecords returns all trust records for a project.
func (r *Repository) ListTrustRecords(ctx context.Context, projectID string) ([]domain.TrustRecord, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, project_id, platform, trust_type, source_kind, source_key, reason, created_at, created_by
		 FROM filter_trust_records WHERE project_id = ? ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []domain.TrustRecord
	for rows.Next() {
		var rec domain.TrustRecord
		if err := rows.Scan(&rec.ID, &rec.ProjectID, &rec.Platform, &rec.TrustType,
			&rec.SourceKind, &rec.SourceKey, &rec.Reason, &rec.CreatedAt, &rec.CreatedBy); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if records == nil {
		records = []domain.TrustRecord{}
	}
	return records, nil
}

// CreateTrustRecord inserts a trust record using INSERT OR IGNORE, then returns the existing or new row.
func (r *Repository) CreateTrustRecord(ctx context.Context, input domain.TrustRecord) (domain.TrustRecord, error) {
	_, err := r.DB.ExecContext(ctx,
		`INSERT OR IGNORE INTO filter_trust_records
		 (project_id, platform, trust_type, source_kind, source_key, reason, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.ProjectID, input.Platform, input.TrustType,
		input.SourceKind, input.SourceKey, input.Reason,
		coalesceString(input.CreatedBy, "self_host_owner"),
	)
	if err != nil {
		return domain.TrustRecord{}, err
	}

	var rec domain.TrustRecord
	err = r.DB.QueryRowContext(ctx,
		`SELECT id, project_id, platform, trust_type, source_kind, source_key, reason, created_at, created_by
		 FROM filter_trust_records
		 WHERE project_id = ? AND platform = ? AND trust_type = ? AND source_kind = ? AND source_key = ?`,
		input.ProjectID, input.Platform, input.TrustType, input.SourceKind, input.SourceKey,
	).Scan(&rec.ID, &rec.ProjectID, &rec.Platform, &rec.TrustType,
		&rec.SourceKind, &rec.SourceKey, &rec.Reason, &rec.CreatedAt, &rec.CreatedBy)
	if err != nil {
		return domain.TrustRecord{}, err
	}
	return rec, nil
}

// ListFilteredFindings returns a paged union of filtered posts and google_results.
func (r *Repository) ListFilteredFindings(
	ctx context.Context,
	projectID string,
	filters FilteredFindingFilters,
	page, limit int,
) (domain.PagedFilteredFindings, error) {
	postClauses, postParams := buildFilteredFindingClauses("posts", projectID, filters)
	googleClauses, googleParams := buildFilteredFindingClauses("google_results", projectID, filters)

	postWhere := "WHERE " + strings.Join(postClauses, " AND ")
	googleWhere := "WHERE " + strings.Join(googleClauses, " AND ")

	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT id FROM posts %s
			UNION ALL
			SELECT CAST(id AS TEXT) FROM google_results %s
		)`, postWhere, googleWhere)

	countParams := append(postParams, googleParams...)
	var total int64
	if err := r.DB.QueryRowContext(ctx, countSQL, countParams...).Scan(&total); err != nil {
		return domain.PagedFilteredFindings{}, err
	}

	offset := (page - 1) * limit

	querySQL := fmt.Sprintf(`
		SELECT finding_type, id, project_id, platform, title, snippet, url,
			source_identity_json, final_score,
			filter_state, filter_reason, filter_reasons_json, filter_explanation,
			filter_confidence, filter_source, filter_signature, filter_job_id,
			filtered_at, recovered_at, recovery_note
		FROM (
			SELECT 'post' AS finding_type,
				id, project_id, platform, title, body AS snippet, url,
				source_identity_json,
				final_score,
				filter_state, filter_reason, filter_reasons_json, filter_explanation,
				filter_confidence, filter_source, filter_signature, filter_job_id,
				filtered_at, recovered_at, recovery_note
			FROM posts %s
			UNION ALL
			SELECT 'google_result' AS finding_type,
				CAST(id AS TEXT), project_id, 'google', title, snippet, url,
				source_identity_json,
				relevance_score,
				filter_state, filter_reason, filter_reasons_json, filter_explanation,
				filter_confidence, filter_source, filter_signature, filter_job_id,
				filtered_at, recovered_at, recovery_note
			FROM google_results %s
		)
		ORDER BY filtered_at DESC
		LIMIT ? OFFSET ?`,
		postWhere, googleWhere)

	queryParams := append(countParams, limit, offset)
	rows, err := r.DB.QueryContext(ctx, querySQL, queryParams...)
	if err != nil {
		return domain.PagedFilteredFindings{}, err
	}
	defer rows.Close()

	var items []domain.FilteredFinding
	for rows.Next() {
		var f domain.FilteredFinding
		var sourceIdentityJSON, filterReasonsJSON sql.NullString
		var filterReason, filterExplanation, filterSource, filterSignature sql.NullString
		var filterConfidence sql.NullFloat64
		var filterJobID sql.NullInt64
		var filteredAt, recoveredAt, recoveryNote sql.NullString
		var score sql.NullFloat64

		if err := rows.Scan(
			&f.FindingType, &f.ID, &f.ProjectID, &f.Platform, &f.Title, &f.Snippet, &f.URL,
			&sourceIdentityJSON, &score,
			&f.FilterState, &filterReason, &filterReasonsJSON, &filterExplanation,
			&filterConfidence, &filterSource, &filterSignature, &filterJobID,
			&filteredAt, &recoveredAt, &recoveryNote,
		); err != nil {
			return domain.PagedFilteredFindings{}, err
		}

		if score.Valid {
			f.Score = &score.Float64
		}
		if filterReason.Valid {
			f.FilterReason = &filterReason.String
		}
		if filterExplanation.Valid {
			f.FilterExplanation = filterExplanation.String
		}
		if filterConfidence.Valid {
			f.FilterConfidence = &filterConfidence.Float64
		}
		if filterSource.Valid {
			f.FilterSource = filterSource.String
		} else {
			f.FilterSource = domain.FilterSourceNone
		}
		if filterSignature.Valid {
			f.FilterSignature = filterSignature.String
		}
		if filterJobID.Valid {
			f.FilterJobID = &filterJobID.Int64
		}
		if filteredAt.Valid {
			f.FilteredAt = &filteredAt.String
		}
		if recoveredAt.Valid {
			f.RecoveredAt = &recoveredAt.String
		}
		if recoveryNote.Valid {
			f.RecoveryNote = &recoveryNote.String
		}

		f.FilterReasons = []string{}
		if filterReasonsJSON.Valid && json.Valid([]byte(filterReasonsJSON.String)) {
			_ = json.Unmarshal([]byte(filterReasonsJSON.String), &f.FilterReasons)
		}

		f.SourceIdentity = domain.SourceIdentity{}
		if sourceIdentityJSON.Valid && json.Valid([]byte(sourceIdentityJSON.String)) {
			_ = json.Unmarshal([]byte(sourceIdentityJSON.String), &f.SourceIdentity)
		}

		// derive trust options from source identity + filter signature
		f.TrustOptions = deriveTrustOptions(f.Platform, f.SourceIdentity, f.FilterSignature)

		items = append(items, f)
	}
	if err := rows.Err(); err != nil {
		return domain.PagedFilteredFindings{}, err
	}
	if items == nil {
		items = []domain.FilteredFinding{}
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 || totalPages == 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	return domain.PagedFilteredFindings{
		Items: items,
		Pagination: domain.Pagination{
			Page:            page,
			Limit:           limit,
			Total:           total,
			TotalPages:      totalPages,
			HasPreviousPage: page > 1,
			HasNextPage:     page < totalPages,
		},
	}, nil
}

// RestoreFindingVisibility restores visibility for a post or google result.
// It only updates filter-related fields and recovery note/timestamps.
// It does NOT change status, post_score, final_score, draft_comment, draft_provider, or Google relevance fields.
func (r *Repository) RestoreFindingVisibility(ctx context.Context, projectID, findingType, id, note string) error {
	switch findingType {
	case domain.FindingTypePost:
		_, err := r.DB.ExecContext(ctx,
			`UPDATE posts SET
				filter_state = ?,
				filter_reason = NULL,
				filter_reasons_json = '[]',
				filter_explanation = '',
				filter_confidence = NULL,
				filter_source = ?,
				filter_signature = '',
				filter_job_id = NULL,
				recovered_at = datetime('now'),
				recovery_note = ?
			WHERE id = ? AND project_id = ?`,
			domain.FilterStateVisible, domain.FilterSourceNone, note, id, projectID,
		)
		return err
	case domain.FindingTypeGoogleResult:
		_, err := r.DB.ExecContext(ctx,
			`UPDATE google_results SET
				filter_state = ?,
				filter_reason = NULL,
				filter_reasons_json = '[]',
				filter_explanation = '',
				filter_confidence = NULL,
				filter_source = ?,
				filter_signature = '',
				filter_job_id = NULL,
				recovered_at = datetime('now'),
				recovery_note = ?
			WHERE id = ? AND project_id = ?`,
			domain.FilterStateVisible, domain.FilterSourceNone, note, id, projectID,
		)
		return err
	default:
		return fmt.Errorf("%w: unknown finding_type %q", ErrValidation, findingType)
	}
}

// ApplyPostFilterDecision persists a filter decision onto a social post row.
func (r *Repository) ApplyPostFilterDecision(ctx context.Context, projectID, postID string, decision domain.FilterDecision, jobID *int64) error {
	reasonsJSON, err := json.Marshal(decision.Reasons)
	if err != nil {
		return err
	}
	sourceIdentityJSON, err := json.Marshal(decision.SourceIdentity)
	if err != nil {
		return err
	}

	filteredAtSQL := "NULL"
	if decision.State == domain.FilterStateFiltered {
		filteredAtSQL = "datetime('now')"
	}

	query := fmt.Sprintf(`UPDATE posts SET
		filter_state = ?,
		filter_reason = ?,
		filter_reasons_json = ?,
		filter_explanation = ?,
		filter_confidence = ?,
		filter_source = ?,
		filter_signature = ?,
		filter_job_id = ?,
		filtered_at = %s,
		recovered_at = NULL,
		recovery_note = NULL,
		source_identity_json = ?
	WHERE id = ? AND project_id = ?`, filteredAtSQL)

	_, err = r.DB.ExecContext(ctx, query,
		decision.State,
		nullableString(decision.Reason),
		string(reasonsJSON),
		decision.Explanation,
		decision.Confidence,
		coalesceString(decision.Source, domain.FilterSourceNone),
		decision.Signature,
		jobID,
		string(sourceIdentityJSON),
		postID, projectID,
	)
	return err
}

// ApplyGoogleFilterDecision persists a filter decision onto a google_result row.
func (r *Repository) ApplyGoogleFilterDecision(ctx context.Context, projectID string, resultID int64, decision domain.FilterDecision, jobID *int64) error {
	reasonsJSON, err := json.Marshal(decision.Reasons)
	if err != nil {
		return err
	}
	sourceIdentityJSON, err := json.Marshal(decision.SourceIdentity)
	if err != nil {
		return err
	}

	filteredAtSQL := "NULL"
	if decision.State == domain.FilterStateFiltered {
		filteredAtSQL = "datetime('now')"
	}

	query := fmt.Sprintf(`UPDATE google_results SET
		filter_state = ?,
		filter_reason = ?,
		filter_reasons_json = ?,
		filter_explanation = ?,
		filter_confidence = ?,
		filter_source = ?,
		filter_signature = ?,
		filter_job_id = ?,
		filtered_at = %s,
		recovered_at = NULL,
		recovery_note = NULL,
		source_identity_json = ?
	WHERE id = ? AND project_id = ?`, filteredAtSQL)

	_, err = r.DB.ExecContext(ctx, query,
		decision.State,
		nullableString(decision.Reason),
		string(reasonsJSON),
		decision.Explanation,
		decision.Confidence,
		coalesceString(decision.Source, domain.FilterSourceNone),
		decision.Signature,
		jobID,
		string(sourceIdentityJSON),
		resultID, projectID,
	)
	return err
}

// CreateFilterJob creates a new filter_jobs row and returns it.
func (r *Repository) CreateFilterJob(ctx context.Context, projectID, scope string, targets []domain.FilterJobTarget) (domain.FilterJob, error) {
	targetJSON, err := json.Marshal(targets)
	if err != nil {
		return domain.FilterJob{}, err
	}

	res, err := r.DB.ExecContext(ctx,
		`INSERT INTO filter_jobs (project_id, status, requested_scope, target_ids_json)
		 VALUES (?, ?, ?, ?)`,
		projectID, domain.FilterJobStatusRunning, scope, string(targetJSON),
	)
	if err != nil {
		return domain.FilterJob{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.FilterJob{}, err
	}
	return r.GetFilterJob(ctx, projectID, id)
}

// GetFilterJob returns a single filter_jobs row by ID.
func (r *Repository) GetFilterJob(ctx context.Context, projectID string, jobID int64) (domain.FilterJob, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT id, project_id, status, step, requested_scope, target_ids_json,
			result_json, error, started_at, completed_at
		 FROM filter_jobs WHERE id = ? AND project_id = ?`,
		jobID, projectID,
	)
	return scanFilterJob(row)
}

// ListFilterJobs returns recent filter_jobs for a project ordered newest first.
func (r *Repository) ListFilterJobs(ctx context.Context, projectID string, limit int) ([]domain.FilterJob, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, project_id, status, step, requested_scope, target_ids_json,
			result_json, error, started_at, completed_at
		 FROM filter_jobs WHERE project_id = ?
		 ORDER BY started_at DESC LIMIT ?`,
		projectID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.FilterJob
	for rows.Next() {
		job, err := scanFilterJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if jobs == nil {
		jobs = []domain.FilterJob{}
	}
	return jobs, nil
}

// UpdateFilterJobStep updates the step field of a running filter job.
func (r *Repository) UpdateFilterJobStep(ctx context.Context, projectID string, jobID int64, step string) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE filter_jobs SET step = ? WHERE id = ? AND project_id = ?`,
		step, jobID, projectID,
	)
	return err
}

// CompleteFilterJob marks a filter job as completed with a result payload.
func (r *Repository) CompleteFilterJob(ctx context.Context, projectID string, jobID int64, result domain.FilterJobResult) (domain.FilterJob, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return domain.FilterJob{}, err
	}

	_, err = r.DB.ExecContext(ctx,
		`UPDATE filter_jobs SET
			status = ?,
			result_json = ?,
			completed_at = datetime('now')
		 WHERE id = ? AND project_id = ?`,
		domain.FilterJobStatusCompleted, string(resultJSON), jobID, projectID,
	)
	if err != nil {
		return domain.FilterJob{}, err
	}
	return r.GetFilterJob(ctx, projectID, jobID)
}

// FailFilterJob marks a filter job as failed with an error message.
func (r *Repository) FailFilterJob(ctx context.Context, projectID string, jobID int64, msg string) (domain.FilterJob, error) {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE filter_jobs SET
			status = ?,
			error = ?,
			completed_at = datetime('now')
		 WHERE id = ? AND project_id = ?`,
		domain.FilterJobStatusFailed, msg, jobID, projectID,
	)
	if err != nil {
		return domain.FilterJob{}, err
	}
	return r.GetFilterJob(ctx, projectID, jobID)
}

// --- scanner helpers ---

type filterJobScanner interface {
	Scan(dest ...any) error
}

func scanFilterJob(row filterJobScanner) (domain.FilterJob, error) {
	var job domain.FilterJob
	var step, resultJSON, errMsg, completedAt sql.NullString
	var targetIDsJSON string

	if err := row.Scan(
		&job.ID, &job.ProjectID, &job.Status, &step, &job.RequestedScope,
		&targetIDsJSON, &resultJSON, &errMsg, &job.StartedAt, &completedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.FilterJob{}, ErrNotFound
		}
		return domain.FilterJob{}, err
	}

	if step.Valid {
		job.Step = &step.String
	}
	if errMsg.Valid {
		job.Error = &errMsg.String
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.String
	}

	job.Targets = []domain.FilterJobTarget{}
	if targetIDsJSON != "" && json.Valid([]byte(targetIDsJSON)) {
		_ = json.Unmarshal([]byte(targetIDsJSON), &job.Targets)
	}

	if resultJSON.Valid && resultJSON.String != "" {
		var res domain.FilterJobResult
		if err := json.Unmarshal([]byte(resultJSON.String), &res); err == nil {
			job.Result = &res
		}
	}

	return job, nil
}

// --- filter clause builders ---

func buildFilteredFindingClauses(table, projectID string, filters FilteredFindingFilters) ([]string, []any) {
	clauses := []string{"project_id = ?", "filter_state = 'filtered'"}
	params := []any{projectID}

	if filters.Platform != "" {
		if table == "posts" {
			clauses = append(clauses, "platform = ?")
			params = append(params, filters.Platform)
		} else if table == "google_results" && filters.Platform == "google" {
			// google_results are always platform=google; no extra clause needed
		} else if table == "google_results" && filters.Platform != "google" {
			// exclude all google results if filtering by non-google platform
			clauses = append(clauses, "1 = 0")
		}
	}

	if filters.Reason != "" {
		clauses = append(clauses, "filter_reason = ?")
		params = append(params, filters.Reason)
	}

	if filters.Source != "" {
		clauses = append(clauses, "filter_source = ?")
		params = append(params, filters.Source)
	}

	if filters.AIUsed != nil {
		if *filters.AIUsed {
			clauses = append(clauses, "filter_source = 'ai'")
		} else {
			clauses = append(clauses, "filter_source != 'ai'")
		}
	}

	if filters.MinConfidence != nil {
		clauses = append(clauses, "filter_confidence >= ?")
		params = append(params, *filters.MinConfidence)
	}

	if filters.MaxConfidence != nil {
		clauses = append(clauses, "filter_confidence <= ?")
		params = append(params, *filters.MaxConfidence)
	}

	return clauses, params
}

// --- trust option derivation ---

func deriveTrustOptions(platform string, identity domain.SourceIdentity, signature string) []domain.FilterTrustOption {
	var opts []domain.FilterTrustOption
	for _, kind := range []string{"reddit_author", "subreddit", "bluesky_did", "bluesky_handle", "domain", "canonical_url"} {
		if key, ok := identity[kind]; ok && key != "" {
			opts = append(opts, domain.FilterTrustOption{
				Platform:   platform,
				TrustType:  domain.TrustTypeSource,
				SourceKind: kind,
				SourceKey:  key,
				Label:      deriveTrustLabel(platform, kind, key),
			})
		}
	}
	if signature != "" {
		opts = append(opts, domain.FilterTrustOption{
			Platform:   platform,
			TrustType:  domain.TrustTypeFilterSignature,
			SourceKind: "filter_signature",
			SourceKey:  signature,
			Label:      "Trust exact pattern: " + signature,
		})
	}
	return opts
}

func deriveTrustLabel(platform, kind, key string) string {
	switch kind {
	case "reddit_author":
		return fmt.Sprintf("Trust Reddit author u/%s", key)
	case "subreddit":
		return fmt.Sprintf("Trust subreddit r/%s", key)
	case "bluesky_did":
		return fmt.Sprintf("Trust Bluesky DID %s", key)
	case "bluesky_handle":
		return fmt.Sprintf("Trust Bluesky handle %s", key)
	case "domain":
		return fmt.Sprintf("Trust domain %s", key)
	case "canonical_url":
		return fmt.Sprintf("Trust URL %s", key)
	default:
		return fmt.Sprintf("Trust %s %s", platform, key)
	}
}

// --- small helpers ---

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
