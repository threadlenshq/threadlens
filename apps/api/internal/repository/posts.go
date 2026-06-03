package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
)

// PostFilters holds optional filter values for listing posts.
type PostFilters struct {
	Status         string
	Platform       string
	MinScore       *float64
	MaxScore       *float64
	EngagementType string
	SignalType     string
	HasDMTargets   bool
}

// PatchPostRequest contains mutable fields for a single post patch.
type PatchPostRequest struct {
	Status        *string
	DraftComment  *string
	DraftProvider *string
}

func (r *Repository) ListPosts(ctx context.Context, projectID string, filters PostFilters) ([]domain.Post, error) {
	clauses, params := buildPostFilterClauses(filters)
	whereSql := ""
	if len(clauses) > 0 {
		whereSql = " AND " + strings.Join(clauses, " AND ")
	}
	allParams := append([]any{projectID}, params...)
	sql := "SELECT * FROM posts WHERE project_id = ?" + whereSql + " ORDER BY final_score DESC, found_at DESC"
	rows, err := r.DB.QueryContext(ctx, sql, allParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, err
	}
	// attach dm_targets for each post
	for i := range posts {
		targets, err := r.listDMTargets(ctx, posts[i].ID)
		if err != nil {
			return nil, err
		}
		posts[i].DMTargets = targets
	}
	if posts == nil {
		posts = []domain.Post{}
	}
	return posts, nil
}

func (r *Repository) ListPostsPage(ctx context.Context, projectID string, filters PostFilters, page int, limit int) (domain.PagedPosts, error) {
	clauses, params := buildPostFilterClauses(filters)
	whereSql := ""
	if len(clauses) > 0 {
		whereSql = " AND " + strings.Join(clauses, " AND ")
	}
	baseParams := append([]any{projectID}, params...)

	var total int64
	countSQL := "SELECT COUNT(*) FROM posts WHERE project_id = ?" + whereSql
	if err := r.DB.QueryRowContext(ctx, countSQL, baseParams...).Scan(&total); err != nil {
		return domain.PagedPosts{}, err
	}

	offset := (page - 1) * limit
	querySQL := "SELECT * FROM posts WHERE project_id = ?" + whereSql + " ORDER BY final_score DESC, found_at DESC LIMIT ? OFFSET ?"
	pageParams := append(baseParams, limit, offset)
	rows, err := r.DB.QueryContext(ctx, querySQL, pageParams...)
	if err != nil {
		return domain.PagedPosts{}, err
	}
	defer rows.Close()
	posts, err := scanPosts(rows)
	if err != nil {
		return domain.PagedPosts{}, err
	}
	for i := range posts {
		targets, err := r.listDMTargets(ctx, posts[i].ID)
		if err != nil {
			return domain.PagedPosts{}, err
		}
		posts[i].DMTargets = targets
	}
	if posts == nil {
		posts = []domain.Post{}
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 || totalPages == 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	return domain.PagedPosts{
		Items: posts,
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

func (r *Repository) GetPost(ctx context.Context, projectID string, postID string) (domain.Post, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT * FROM posts WHERE id = ? AND project_id = ?",
		postID, projectID,
	)
	post, err := scanPost(row)
	if err == sql.ErrNoRows {
		return domain.Post{}, ErrNotFound
	}
	if err != nil {
		return domain.Post{}, err
	}
	targets, err := r.listDMTargets(ctx, post.ID)
	if err != nil {
		return domain.Post{}, err
	}
	post.DMTargets = targets
	return post, nil
}

func (r *Repository) PatchPost(ctx context.Context, projectID string, postID string, req PatchPostRequest) (domain.Post, error) {
	// verify post exists and belongs to project
	existing, err := r.GetPost(ctx, projectID, postID)
	if err != nil {
		return domain.Post{}, err
	}

	updates := []string{}
	values := []any{}

	if req.DraftComment != nil {
		updates = append(updates, "draft_comment = ?")
		values = append(values, *req.DraftComment)
		// auto-set to drafted if no explicit status
		if req.Status == nil {
			updates = append(updates, "status = ?")
			values = append(values, "drafted")
		}
	}

	if req.DraftProvider != nil {
		updates = append(updates, "draft_provider = ?")
		values = append(values, *req.DraftProvider)
	}

	if req.Status != nil {
		updates = append(updates, "status = ?")
		values = append(values, *req.Status)
	}

	if len(updates) == 0 {
		return existing, nil
	}

	values = append(values, postID)
	updateSQL := fmt.Sprintf("UPDATE posts SET %s WHERE id = ?", strings.Join(updates, ", "))
	if _, err := r.DB.ExecContext(ctx, updateSQL, values...); err != nil {
		return domain.Post{}, err
	}

	return r.GetPost(ctx, projectID, postID)
}

func (r *Repository) BulkPatchPosts(ctx context.Context, projectID string, ids []string, status string) (int64, error) {
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+2)
	args = append(args, status)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	args = append(args, projectID)

	updateSQL := fmt.Sprintf(
		"UPDATE posts SET status = ? WHERE id IN (%s) AND project_id = ?",
		strings.Join(placeholders, ", "),
	)
	result, err := r.DB.ExecContext(ctx, updateSQL, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) PatchDMTarget(ctx context.Context, projectID string, postID string, username string, draftDM *string, dmStatus *string) (domain.DMTarget, error) {
	// verify post belongs to project
	var checkID string
	err := r.DB.QueryRowContext(ctx, "SELECT id FROM posts WHERE id = ? AND project_id = ?", postID, projectID).Scan(&checkID)
	if err == sql.ErrNoRows {
		return domain.DMTarget{}, ErrNotFound
	}
	if err != nil {
		return domain.DMTarget{}, err
	}

	target, err := r.GetDMTarget(ctx, postID, username)
	if err != nil {
		return domain.DMTarget{}, err
	}

	updates := []string{}
	values := []any{}

	if draftDM != nil {
		updates = append(updates, "draft_dm = ?")
		values = append(values, *draftDM)
	}
	if dmStatus != nil {
		updates = append(updates, "dm_status = ?")
		values = append(values, *dmStatus)
	}

	if len(updates) == 0 {
		return target, nil
	}

	values = append(values, target.ID)
	updateSQL := fmt.Sprintf("UPDATE dm_targets SET %s WHERE id = ?", strings.Join(updates, ", "))
	if _, err := r.DB.ExecContext(ctx, updateSQL, values...); err != nil {
		return domain.DMTarget{}, err
	}

	return r.GetDMTarget(ctx, postID, username)
}

func (r *Repository) GetDMTarget(ctx context.Context, postID string, username string) (domain.DMTarget, error) {
	row := r.DB.QueryRowContext(ctx,
		"SELECT id, post_id, username, intent_score, signal, context, approach, draft_dm, draft_provider, dm_status FROM dm_targets WHERE post_id = ? AND username = ?",
		postID, username,
	)
	target, err := scanDMTarget(row)
	if err == sql.ErrNoRows {
		return domain.DMTarget{}, ErrNotFound
	}
	return target, err
}

// CountDMTargets returns the number of dm_target rows for the given post.
func (r *Repository) CountDMTargets(ctx context.Context, postID string) (int, error) {
	var count int
	err := r.DB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM dm_targets WHERE post_id = ?",
		postID,
	).Scan(&count)
	return count, err
}

// InsertDMTargets inserts the provided targets for the given post inside a
// transaction using INSERT OR IGNORE to avoid duplicates. Returns the number
// of rows actually inserted.
func (r *Repository) InsertDMTargets(ctx context.Context, postID string, targets []domain.DMTargetInsert) (int64, error) {
	if len(targets) == 0 {
		return 0, nil
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO dm_targets (post_id, username, intent_score, signal, context, approach, dm_status) VALUES (?, ?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	defer stmt.Close()

	var inserted int64
	for _, t := range targets {
		status := t.DMStatus
		if status == "" {
			status = "new"
		}
		res, err := stmt.ExecContext(ctx, postID, t.Username, t.IntentScore, t.Signal, t.Context, t.Approach, status)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		n, _ := res.RowsAffected()
		inserted += n
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

// listDMTargets fetches dm_targets for a post ordered by intent_score DESC.
func (r *Repository) listDMTargets(ctx context.Context, postID string) ([]domain.DMTarget, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, post_id, username, intent_score, signal, context, approach, draft_dm, draft_provider, dm_status FROM dm_targets WHERE post_id = ? ORDER BY intent_score DESC",
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []domain.DMTarget
	for rows.Next() {
		var t domain.DMTarget
		var draftDM, draftProvider sql.NullString
		if err := rows.Scan(&t.ID, &t.PostID, &t.Username, &t.IntentScore, &t.Signal, &t.Context, &t.Approach, &draftDM, &draftProvider, &t.DMStatus); err != nil {
			return nil, err
		}
		if draftDM.Valid {
			t.DraftDM = &draftDM.String
		}
		if draftProvider.Valid {
			t.DraftProvider = &draftProvider.String
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if targets == nil {
		targets = []domain.DMTarget{}
	}
	return targets, nil
}

func buildPostFilterClauses(filters PostFilters) ([]string, []any) {
	var clauses []string
	var params []any

	clauses = append(clauses, "filter_state = 'visible'")

	if filters.Status != "" {
		clauses = append(clauses, "status = ?")
		params = append(params, filters.Status)
	}
	if filters.Platform != "" {
		clauses = append(clauses, "platform = ?")
		params = append(params, filters.Platform)
	}
	if filters.MinScore != nil {
		clauses = append(clauses, "final_score >= ?")
		params = append(params, *filters.MinScore)
	}
	if filters.MaxScore != nil {
		clauses = append(clauses, "final_score < ?")
		params = append(params, *filters.MaxScore)
	}
	if filters.EngagementType != "" {
		clauses = append(clauses, "engagement_type = ?")
		params = append(params, filters.EngagementType)
	}
	if filters.SignalType != "" {
		clauses = append(clauses, "signal_type = ?")
		params = append(params, filters.SignalType)
	}
	if filters.HasDMTargets {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM dm_targets WHERE dm_targets.post_id = posts.id)")
	}

	return clauses, params
}

// scanPosts scans all rows into Post slice.
func scanPosts(rows *sql.Rows) ([]domain.Post, error) {
	var posts []domain.Post
	for rows.Next() {
		post, err := scanPostRow(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

// postScanner is implemented by both *sql.Row and *sql.Rows.
type postScanner interface {
	Scan(dest ...any) error
}

func scanPost(row *sql.Row) (domain.Post, error) {
	return scanPostRow(row)
}

func scanPostRow(row postScanner) (domain.Post, error) {
	var p domain.Post
	var subreddit, blueskyURI, blueskyCID sql.NullString
	var redditScore, numComments, likeCount, replyCount, repostCount sql.NullInt64
	var commentScore sql.NullFloat64
	var angle, why, karmaTopic, topCommentSignals sql.NullString
	var draftComment, draftProvider, signalType sql.NullString
	var createdAt sql.NullString
	var filterReason, filterExplanation, filterSource, filterSignature, filterReasonsJSON, sourceIdentityJSON sql.NullString
	var filterConfidence sql.NullFloat64
	var filterJobID sql.NullInt64
	var filteredAt, recoveredAt, recoveryNote sql.NullString

	err := row.Scan(
		&p.ID, &p.ProjectID, &p.Platform,
		&p.Title, &p.Body, &p.Author, &p.URL,
		&subreddit, &redditScore, &numComments,
		&likeCount, &replyCount, &repostCount,
		&blueskyURI, &blueskyCID,
		&p.PostScore, &commentScore, &p.FinalScore,
		&angle, &why, &p.EngagementType,
		&karmaTopic, &topCommentSignals,
		&p.Status, &draftComment, &draftProvider,
		&signalType,
		&createdAt, &p.FoundAt, &p.ScoutedAt,
		&p.FilterState, &filterReason, &filterReasonsJSON, &filterExplanation, &filterConfidence,
		&filterSource, &filterSignature, &filterJobID, &filteredAt, &recoveredAt, &recoveryNote, &sourceIdentityJSON,
	)
	if err != nil {
		return domain.Post{}, err
	}

	if subreddit.Valid {
		p.Subreddit = &subreddit.String
	}
	if redditScore.Valid {
		p.RedditScore = &redditScore.Int64
	}
	if numComments.Valid {
		p.NumComments = &numComments.Int64
	}
	if likeCount.Valid {
		p.LikeCount = &likeCount.Int64
	}
	if replyCount.Valid {
		p.ReplyCount = &replyCount.Int64
	}
	if repostCount.Valid {
		p.RepostCount = &repostCount.Int64
	}
	if blueskyURI.Valid {
		p.BlueskyURI = &blueskyURI.String
	}
	if blueskyCID.Valid {
		p.BlueskyCID = &blueskyCID.String
	}
	if commentScore.Valid {
		p.CommentScore = &commentScore.Float64
	}
	if angle.Valid {
		p.Angle = &angle.String
	}
	if why.Valid {
		p.Why = why.String
	}
	if karmaTopic.Valid {
		p.KarmaTopic = &karmaTopic.String
	}
	if topCommentSignals.Valid {
		p.TopCommentSignals = &topCommentSignals.String
	}
	if draftComment.Valid {
		p.DraftComment = &draftComment.String
	}
	if draftProvider.Valid {
		p.DraftProvider = &draftProvider.String
	}
	if signalType.Valid {
		p.SignalType = &signalType.String
	}
	if createdAt.Valid {
		p.CreatedAt = &createdAt.String
	}

	if p.FilterState == "" {
		p.FilterState = domain.FilterStateVisible
	}
	if filterReason.Valid {
		p.FilterReason = &filterReason.String
	}
	if filterExplanation.Valid {
		p.FilterExplanation = filterExplanation.String
	}
	if filterConfidence.Valid {
		p.FilterConfidence = &filterConfidence.Float64
	}
	if filterSource.Valid {
		p.FilterSource = filterSource.String
	} else {
		p.FilterSource = domain.FilterSourceNone
	}
	if filterSignature.Valid {
		p.FilterSignature = filterSignature.String
	}
	if filterJobID.Valid {
		p.FilterJobID = &filterJobID.Int64
	}
	if filteredAt.Valid {
		p.FilteredAt = &filteredAt.String
	}
	if recoveredAt.Valid {
		p.RecoveredAt = &recoveredAt.String
	}
	if recoveryNote.Valid {
		p.RecoveryNote = &recoveryNote.String
	}
	p.FilterReasons = []string{}
	if filterReasonsJSON.Valid && json.Valid([]byte(filterReasonsJSON.String)) {
		_ = json.Unmarshal([]byte(filterReasonsJSON.String), &p.FilterReasons)
	}
	p.SourceIdentity = domain.SourceIdentity{}
	if sourceIdentityJSON.Valid && json.Valid([]byte(sourceIdentityJSON.String)) {
		_ = json.Unmarshal([]byte(sourceIdentityJSON.String), &p.SourceIdentity)
	}

	return p, nil
}

func scanDMTarget(row *sql.Row) (domain.DMTarget, error) {
	var t domain.DMTarget
	var draftDM, draftProvider sql.NullString
	err := row.Scan(&t.ID, &t.PostID, &t.Username, &t.IntentScore, &t.Signal, &t.Context, &t.Approach, &draftDM, &draftProvider, &t.DMStatus)
	if err != nil {
		return domain.DMTarget{}, err
	}
	if draftDM.Valid {
		t.DraftDM = &draftDM.String
	}
	if draftProvider.Valid {
		t.DraftProvider = &draftProvider.String
	}
	return t, nil
}

// UpdateDMTargetProvider sets the draft_provider field on a dm_target row.
func (r *Repository) UpdateDMTargetProvider(ctx context.Context, dmTargetID int64, providerID string) (sql.Result, error) {
	return r.DB.ExecContext(ctx, "UPDATE dm_targets SET draft_provider = ? WHERE id = ?", providerID, dmTargetID)
}

// SeenIDs returns the set of post IDs already recorded in seen_posts for
// the given project and platform.
func (r *Repository) SeenIDs(ctx context.Context, projectID string, platform string) (map[string]bool, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id FROM seen_posts WHERE project_id = ? AND platform = ?",
		projectID, platform,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// MarkSeen inserts the given post IDs into seen_posts for the project+platform,
// ignoring duplicates.
func (r *Repository) MarkSeen(ctx context.Context, projectID string, platform string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx,
		"INSERT OR IGNORE INTO seen_posts (id, project_id, platform) VALUES (?, ?, ?)",
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, id := range ids {
		if _, err := stmt.ExecContext(ctx, id, projectID, platform); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// InsertSocialPosts inserts posts into the posts table using INSERT OR IGNORE,
// mirroring the Express storeAll transaction. Returns the number of newly
// inserted rows.
func (r *Repository) InsertSocialPosts(ctx context.Context, posts []domain.Post) (int64, error) {
	if len(posts) == 0 {
		return 0, nil
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO posts (
			id, project_id, platform, title, body, author, url,
			subreddit, reddit_score, num_comments,
			like_count, reply_count, repost_count, bluesky_uri, bluesky_cid,
			post_score, final_score, angle, why, signal_type,
			filter_state, filter_reason, filter_reasons_json, filter_explanation, filter_confidence,
			filter_source, filter_signature, filter_job_id, filtered_at, recovered_at, recovery_note, source_identity_json,
			created_at, found_at, scouted_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?,
			?, datetime('now'), datetime('now')
		)
	`)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	defer stmt.Close()

	var inserted int64
	for _, p := range posts {
		res, err := stmt.ExecContext(ctx,
			p.ID, p.ProjectID, p.Platform, p.Title, p.Body, p.Author, p.URL,
			p.Subreddit, p.RedditScore, p.NumComments,
			p.LikeCount, p.ReplyCount, p.RepostCount, p.BlueskyURI, p.BlueskyCID,
			p.PostScore, p.FinalScore, p.Angle, p.Why, p.SignalType,
			coalesceString(p.FilterState, domain.FilterStateVisible), p.FilterReason, safeJSON(p.FilterReasons, []string{}), p.FilterExplanation, p.FilterConfidence,
			coalesceString(p.FilterSource, domain.FilterSourceNone), p.FilterSignature, p.FilterJobID, p.FilteredAt, p.RecoveredAt, p.RecoveryNote, p.SourceIdentity.JSON(),
			p.CreatedAt,
		)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		n, _ := res.RowsAffected()
		inserted += n
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func coalesceString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
