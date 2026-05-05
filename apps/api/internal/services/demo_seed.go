package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

const (
	DemoProjectID   = "demo-project"
	demoSeedKey     = "seed.demo.default"
	demoSeedVersion = 1
)

// DemoSeedResult is returned by SeedDemoData.
type DemoSeedResult struct {
	Status    string // "seeded" or "noop"
	ProjectID string
	Version   int
}

type demoPost struct {
	ID                string
	Platform          string
	Title             string
	Body              string
	Author            string
	URL               string
	Subreddit         *string
	RedditScore       *int
	NumComments       *int
	LikeCount         *int
	ReplyCount        *int
	RepostCount       *int
	BlueskyURI        *string
	BlueskyCD         *string
	PostScore         float64
	CommentScore      float64
	FinalScore        float64
	Angle             string
	Why               string
	EngagementType    string
	KarmaTopic        *string
	TopCommentSignals []string
	Status            string
	SignalType        string
	CreatedAt         string
	FoundAt           string
	ScoutedAt         string
}

func str(s string) *string { return &s }
func iptr(i int) *int      { return &i }

var demoPosts = []demoPost{
	{
		ID: "demo-reddit-habit-001", Platform: "reddit",
		Title:     "Every habit tracker turns into another inbox I ignore",
		Body:      "I start strong for four days, then the app asks me to configure tags, streak rules, reminders, and reviews. I just need something that notices when I am slipping before I disappear for two weeks.",
		Author:    "routine_reset",
		URL:       "https://reddit.com/r/productivity/comments/demo001/every_habit_tracker_turns_into_another_inbox/",
		Subreddit: str("productivity"), RedditScore: iptr(184), NumComments: iptr(39),
		PostScore: 9.1, CommentScore: 8.2, FinalScore: 9.4,
		Angle:             "habit tracking frustration",
		Why:               "Strong frustration with habit tools becoming too complex and a clear request for early slip detection.",
		EngagementType:    "product",
		TopCommentSignals: []string{"configuration fatigue", "streak anxiety", "early intervention"},
		Status:            "starred", SignalType: "frustration",
		CreatedAt: "2026-04-23 14:10:00", FoundAt: "2026-05-04 09:05:00", ScoutedAt: "2026-05-04 09:05:00",
	},
	{
		ID: "demo-reddit-habit-002", Platform: "reddit",
		Title:     "Looking for accountability that is not another group chat",
		Body:      "Group chats work for two days and then everyone goes silent. I want a small nudge from something that understands my routine and asks one useful question when I skip the gym.",
		Author:    "silent_checkin",
		URL:       "https://reddit.com/r/getdisciplined/comments/demo002/accountability_not_group_chat/",
		Subreddit: str("getdisciplined"), RedditScore: iptr(97), NumComments: iptr(26),
		PostScore: 8.7, CommentScore: 7.6, FinalScore: 8.9,
		Angle:             "accountability drop-off",
		Why:               "The poster rejects generic accountability groups and asks for context-aware nudges.",
		EngagementType:    "product",
		TopCommentSignals: []string{"group accountability fatigue", "personalized nudges"},
		Status:            "new", SignalType: "seeking_solution",
		CreatedAt: "2026-04-24 08:45:00", FoundAt: "2026-05-04 09:06:00", ScoutedAt: "2026-05-04 09:06:00",
	},
	{
		ID: "demo-bsky-habit-003", Platform: "bluesky",
		Title:     "Habit apps keep punishing normal weeks",
		Body:      "The streak broke because my kid was sick and now the app acts like I failed. I need habit tracking that adapts to messy real life instead of shaming me.",
		Author:    "messyweeks.bsky.social",
		URL:       "https://bsky.app/profile/messyweeks.bsky.social/post/demo003",
		LikeCount: iptr(76), ReplyCount: iptr(18), RepostCount: iptr(9),
		BlueskyURI: str("at://did:plc:demo003/app.bsky.feed.post/demo003"),
		BlueskyCD:  str("bafyreibskyhabit003"),
		PostScore:  8.4, CommentScore: 7.9, FinalScore: 8.7,
		Angle:             "overwhelming habit apps",
		Why:               "Clear emotional pain around streak punishment and a specific need for adaptive tracking.",
		EngagementType:    "product",
		TopCommentSignals: []string{"streak shame", "adaptive routines"},
		Status:            "new", SignalType: "frustration",
		CreatedAt: "2026-04-25 19:20:00", FoundAt: "2026-05-04 09:07:00", ScoutedAt: "2026-05-04 09:07:00",
	},
	{
		ID: "demo-reddit-habit-004", Platform: "reddit",
		Title:     "Is there a habit app for people who hate dashboards?",
		Body:      "I do not want charts. I want a daily text that says the one thing I promised myself yesterday and lets me reply done, skipped, or blocked.",
		Author:    "nodashboards",
		URL:       "https://reddit.com/r/productivity/comments/demo004/habit_app_for_people_who_hate_dashboards/",
		Subreddit: str("productivity"), RedditScore: iptr(143), NumComments: iptr(44),
		PostScore: 8.1, CommentScore: 8.0, FinalScore: 8.3,
		Angle:             "habit tracking frustration",
		Why:               "Specific lightweight workflow request with strong dislike for dashboard-heavy tools.",
		EngagementType:    "product",
		TopCommentSignals: []string{"dashboard aversion", "text-based check-in"},
		Status:            "new", SignalType: "seeking_solution",
		CreatedAt: "2026-04-26 11:30:00", FoundAt: "2026-05-04 09:08:00", ScoutedAt: "2026-05-04 09:08:00",
	},
	{
		ID: "demo-bsky-habit-005", Platform: "bluesky",
		Title:     "Tiny accountability beats streak fireworks",
		Body:      "I would pay for a habit coach that just remembers my context and checks in gently. Streak fireworks make me uninstall faster.",
		Author:    "gentlenudges.bsky.social",
		URL:       "https://bsky.app/profile/gentlenudges.bsky.social/post/demo005",
		LikeCount: iptr(58), ReplyCount: iptr(11), RepostCount: iptr(6),
		BlueskyURI: str("at://did:plc:demo005/app.bsky.feed.post/demo005"),
		BlueskyCD:  str("bafyreibskyhabit005"),
		PostScore:  7.9, CommentScore: 7.3, FinalScore: 8.0,
		Angle:             "daily routine accountability",
		Why:               "Mentions willingness to pay and contrasts gentle accountability with gamified streaks.",
		EngagementType:    "product",
		TopCommentSignals: []string{"willingness to pay", "gentle accountability"},
		Status:            "reviewed", SignalType: "buying_intent",
		CreatedAt: "2026-04-27 16:05:00", FoundAt: "2026-05-04 09:09:00", ScoutedAt: "2026-05-04 09:09:00",
	},
	{
		ID: "demo-reddit-habit-006", Platform: "reddit",
		Title:     "My ADHD routine falls apart when the reminder is too easy to dismiss",
		Body:      "Calendar reminders are invisible now. I need an accountability loop that makes me choose why I am skipping so tomorrow can be easier.",
		Author:    "contextswitcher",
		URL:       "https://reddit.com/r/ADHD/comments/demo006/routine_falls_apart_when_reminder_is_easy_to_dismiss/",
		Subreddit: str("ADHD"), RedditScore: iptr(212), NumComments: iptr(67),
		PostScore: 7.7, CommentScore: 8.1, FinalScore: 7.9,
		Angle:             "accountability drop-off",
		Why:               "The pain is about reminder habituation and a desired skip-reason loop.",
		EngagementType:    "product",
		TopCommentSignals: []string{"reminder blindness", "skip reason capture"},
		Status:            "new", SignalType: "frustration",
		CreatedAt: "2026-04-28 07:55:00", FoundAt: "2026-05-04 09:10:00", ScoutedAt: "2026-05-04 09:10:00",
	},
	{
		ID: "demo-bsky-habit-007", Platform: "bluesky",
		Title:     "Habit tracking needs a blocked button",
		Body:      "Done or failed is not enough. Half the time I am blocked by travel, sleep, or childcare. The app should learn from that instead of resetting the streak.",
		Author:    "blockedbutton.bsky.social",
		URL:       "https://bsky.app/profile/blockedbutton.bsky.social/post/demo007",
		LikeCount: iptr(41), ReplyCount: iptr(7), RepostCount: iptr(4),
		BlueskyURI: str("at://did:plc:demo007/app.bsky.feed.post/demo007"),
		BlueskyCD:  str("bafyreibskyhabit007"),
		PostScore:  7.5, CommentScore: 6.9, FinalScore: 7.6,
		Angle:             "overwhelming habit apps",
		Why:               "Asks for a concrete blocked state that could improve adaptive coaching.",
		EngagementType:    "product",
		TopCommentSignals: []string{"blocked state", "context learning"},
		Status:            "new", SignalType: "feature_request",
		CreatedAt: "2026-04-29 12:15:00", FoundAt: "2026-05-04 09:11:00", ScoutedAt: "2026-05-04 09:11:00",
	},
	{
		ID: "demo-reddit-habit-008", Platform: "reddit",
		Title:     "I keep rebuilding the same morning routine spreadsheet",
		Body:      "Every month I make a new tracker because the old one stopped matching my life. There has to be a better way to adjust routines without starting over.",
		Author:    "spreadsheetloop",
		URL:       "https://reddit.com/r/getdisciplined/comments/demo008/rebuilding_same_morning_routine_spreadsheet/",
		Subreddit: str("getdisciplined"), RedditScore: iptr(88), NumComments: iptr(21),
		PostScore: 7.1, CommentScore: 6.8, FinalScore: 7.2,
		Angle:          "habit tracking frustration",
		Why:            "Repeated workaround behavior suggests dissatisfaction with current tools.",
		EngagementType: "karma", KarmaTopic: str("manual routine tracking workarounds"),
		TopCommentSignals: []string{"spreadsheet workaround", "routine drift"},
		Status:            "new", SignalType: "workaround",
		CreatedAt: "2026-04-30 09:40:00", FoundAt: "2026-05-04 09:12:00", ScoutedAt: "2026-05-04 09:12:00",
	},
	{
		ID: "demo-bsky-habit-009", Platform: "bluesky",
		Title:     "Accountability buddy apps feel performative",
		Body:      "I do not want to broadcast my goals. I want private accountability that asks what changed when I miss two days.",
		Author:    "privateprogress.bsky.social",
		URL:       "https://bsky.app/profile/privateprogress.bsky.social/post/demo009",
		LikeCount: iptr(37), ReplyCount: iptr(5), RepostCount: iptr(3),
		BlueskyURI: str("at://did:plc:demo009/app.bsky.feed.post/demo009"),
		BlueskyCD:  str("bafyreibskyhabit009"),
		PostScore:  6.9, CommentScore: 6.5, FinalScore: 7.0,
		Angle:             "daily routine accountability",
		Why:               "Highlights privacy concerns and a trigger for contextual accountability after missed days.",
		EngagementType:    "product",
		TopCommentSignals: []string{"private accountability", "missed-day trigger"},
		Status:            "new", SignalType: "privacy_concern",
		CreatedAt: "2026-05-01 17:25:00", FoundAt: "2026-05-04 09:13:00", ScoutedAt: "2026-05-04 09:13:00",
	},
	{
		ID: "demo-reddit-habit-010", Platform: "reddit",
		Title:     "What actually helped you recover after missing a week?",
		Body:      "I can build habits for a while, but missing a week makes me abandon the whole system. Has anyone used a tool that helps restart without guilt?",
		Author:    "restartneeded",
		URL:       "https://reddit.com/r/productivity/comments/demo010/recover_after_missing_a_week/",
		Subreddit: str("productivity"), RedditScore: iptr(65), NumComments: iptr(33),
		PostScore: 6.6, CommentScore: 6.7, FinalScore: 6.8,
		Angle:          "accountability drop-off",
		Why:            "Seeks a restart mechanism after lapse, reinforcing the recovery theme.",
		EngagementType: "karma", KarmaTopic: str("habit restart after missed week"),
		TopCommentSignals: []string{"restart without guilt", "lapse recovery"},
		Status:            "new", SignalType: "seeking_solution",
		CreatedAt: "2026-05-01 21:05:00", FoundAt: "2026-05-04 09:14:00", ScoutedAt: "2026-05-04 09:14:00",
	},
	{
		ID: "demo-bsky-habit-011", Platform: "bluesky",
		Title:     "The best habit reminder is the one I can answer in ten seconds",
		Body:      "A quick check-in beats a giant dashboard. Ask me what happened, remember the answer, and move on.",
		Author:    "tensecondcheck.bsky.social",
		URL:       "https://bsky.app/profile/tensecondcheck.bsky.social/post/demo011",
		LikeCount: iptr(29), ReplyCount: iptr(4), RepostCount: iptr(2),
		BlueskyURI: str("at://did:plc:demo011/app.bsky.feed.post/demo011"),
		BlueskyCD:  str("bafyreibskyhabit011"),
		PostScore:  6.3, CommentScore: 6.0, FinalScore: 6.4,
		Angle:          "daily routine accountability",
		Why:            "Concise workflow preference for low-friction conversational check-ins.",
		EngagementType: "karma", KarmaTopic: str("ten-second check-in loop"),
		TopCommentSignals: []string{"low-friction check-in", "memory of context"},
		Status:            "new", SignalType: "workflow_preference",
		CreatedAt: "2026-05-02 14:00:00", FoundAt: "2026-05-04 09:15:00", ScoutedAt: "2026-05-04 09:15:00",
	},
	{
		ID: "demo-reddit-habit-012", Platform: "reddit",
		Title:     "Do any habit tools separate planning from accountability?",
		Body:      "Planning my routine is easy on Sunday. Following it on Wednesday is the problem. Most apps optimize the plan, not the moment I need help.",
		Author:    "wednesdayproblem",
		URL:       "https://reddit.com/r/getdisciplined/comments/demo012/planning_vs_accountability/",
		Subreddit: str("getdisciplined"), RedditScore: iptr(54), NumComments: iptr(18),
		PostScore: 6.1, CommentScore: 5.8, FinalScore: 6.2,
		Angle:          "accountability drop-off",
		Why:            "Frames the opportunity around in-the-moment support rather than planning.",
		EngagementType: "karma", KarmaTopic: str("planning versus follow-through"),
		TopCommentSignals: []string{"follow-through gap", "moment-of-need support"},
		Status:            "new", SignalType: "problem_framing",
		CreatedAt: "2026-05-02 18:50:00", FoundAt: "2026-05-04 09:16:00", ScoutedAt: "2026-05-04 09:16:00",
	},
}

type demoClusterProductAngle struct {
	Idea        string `json:"idea"`
	TargetNiche string `json:"target_niche"`
	Why         string `json:"why"`
}

type demoCluster struct {
	Name         string                  `json:"name"`
	PostCount    int                     `json:"post_count"`
	AvgPainScore float64                 `json:"avg_pain_score"`
	Signals      map[string]int          `json:"signals"`
	KeyQuotes    []string                `json:"key_quotes"`
	PostIDs      []string                `json:"post_ids"`
	ProductAngle demoClusterProductAngle `json:"product_angle"`
}

var demoClusters = []demoCluster{
	{
		Name: "Lightweight check-ins beat dashboards", PostCount: 4, AvgPainScore: 8.0,
		Signals: map[string]int{"frustration": 1, "seeking_solution": 2, "workflow_preference": 1},
		KeyQuotes: []string{
			"I do not want charts. I want a daily text that says the one thing I promised myself yesterday.",
			"The best habit reminder is the one I can answer in ten seconds.",
		},
		PostIDs: []string{"demo-reddit-habit-001", "demo-reddit-habit-004", "demo-bsky-habit-011", "demo-reddit-habit-012"},
		ProductAngle: demoClusterProductAngle{
			Idea:        "A conversational habit check-in that replaces dashboards with one daily reply loop.",
			TargetNiche: "People who abandon configurable habit trackers and prefer text-like workflows.",
			Why:         "Multiple posts ask for a simpler interaction model and describe dashboard fatigue as the blocker.",
		},
	},
	{
		Name: "Adaptive accountability after life interruptions", PostCount: 5, AvgPainScore: 7.8,
		Signals: map[string]int{"frustration": 3, "feature_request": 1, "privacy_concern": 1},
		KeyQuotes: []string{
			"The streak broke because my kid was sick and now the app acts like I failed.",
			"Done or failed is not enough. Half the time I am blocked by travel, sleep, or childcare.",
		},
		PostIDs: []string{"demo-bsky-habit-003", "demo-bsky-habit-007", "demo-reddit-habit-006", "demo-bsky-habit-009", "demo-reddit-habit-010"},
		ProductAngle: demoClusterProductAngle{
			Idea:        "An adaptive accountability coach that records blockers and helps users restart without shame.",
			TargetNiche: "Busy adults whose routines are interrupted by caregiving, travel, ADHD, or changing schedules.",
			Why:         "The pain is specific, repeated, and tied to existing tools punishing normal interruptions.",
		},
	},
	{
		Name: "Private accountability that remembers context", PostCount: 3, AvgPainScore: 7.6,
		Signals: map[string]int{"seeking_solution": 1, "buying_intent": 1, "workaround": 1},
		KeyQuotes: []string{
			"I would pay for a habit coach that just remembers my context and checks in gently.",
			"Every month I make a new tracker because the old one stopped matching my life.",
		},
		PostIDs: []string{"demo-reddit-habit-002", "demo-bsky-habit-005", "demo-reddit-habit-008"},
		ProductAngle: demoClusterProductAngle{
			Idea:        "A private AI habit coach that learns routine changes and asks one contextual accountability question.",
			TargetNiche: "Users who dislike public accountability groups but still want external follow-through support.",
			Why:         "The cluster includes willingness to pay, repeated spreadsheet workarounds, and rejection of group-chat accountability.",
		},
	},
}

// SeedDemoData inserts the demo project, queries, posts, and report if they have
// not already been seeded (checked via app_settings marker). Matches Express semantics:
// returns "noop" if already seeded, "seeded" on success, error if the demo project ID
// exists without a marker (indicating a conflict).
func SeedDemoData(ctx context.Context, repo *repository.Repository) (DemoSeedResult, error) {
	db := repo.DB

	marker, err := readDemoMarker(ctx, db)
	if err != nil {
		return DemoSeedResult{}, fmt.Errorf("read demo marker: %w", err)
	}

	var existingID string
	err = db.QueryRowContext(ctx, `SELECT id FROM projects WHERE id = ?`, DemoProjectID).Scan(&existingID)
	projectExists := err == nil

	if marker != nil && projectExists {
		return DemoSeedResult{Status: "noop", ProjectID: DemoProjectID, Version: demoSeedVersion}, nil
	}

	if marker == nil && projectExists {
		return DemoSeedResult{}, errors.New("reserved demo project id demo-project already exists without the demo seed marker")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return DemoSeedResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = clearDemoRows(ctx, tx); err != nil {
		return DemoSeedResult{}, fmt.Errorf("clear demo rows: %w", err)
	}
	if err = insertDemoDataset(ctx, tx); err != nil {
		return DemoSeedResult{}, fmt.Errorf("insert demo dataset: %w", err)
	}
	if err = writeDemoMarker(ctx, tx); err != nil {
		return DemoSeedResult{}, fmt.Errorf("write demo marker: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return DemoSeedResult{}, fmt.Errorf("commit: %w", err)
	}

	return DemoSeedResult{Status: "seeded", ProjectID: DemoProjectID, Version: demoSeedVersion}, nil
}

type demoMarker struct {
	Version   int    `json:"version"`
	ProjectID string `json:"projectId"`
	Dataset   string `json:"dataset"`
}

func readDemoMarker(ctx context.Context, db *sql.DB) (*demoMarker, error) {
	var value string
	err := db.QueryRowContext(ctx, `SELECT value FROM app_settings WHERE key = ?`, demoSeedKey).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var m demoMarker
	if jsonErr := json.Unmarshal([]byte(value), &m); jsonErr != nil {
		return &demoMarker{Version: 0, ProjectID: DemoProjectID, Dataset: "default"}, nil
	}
	return &m, nil
}

func writeDemoMarker(ctx context.Context, tx *sql.Tx) error {
	value, _ := json.Marshal(demoMarker{Version: demoSeedVersion, ProjectID: DemoProjectID, Dataset: "default"})
	_, err := tx.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = datetime('now')
	`, demoSeedKey, string(value))
	return err
}

func clearDemoRows(ctx context.Context, tx *sql.Tx) error {
	stmts := []string{
		`UPDATE projects SET selected_report_id = NULL, selected_cluster_index = NULL WHERE id = ?`,
		`DELETE FROM report_posts WHERE report_id IN (SELECT id FROM research_reports WHERE project_id = ?) OR post_id IN (SELECT id FROM posts WHERE project_id = ?)`,
		`DELETE FROM research_reports WHERE project_id = ?`,
		`DELETE FROM scout_runs WHERE project_id = ?`,
		`DELETE FROM posts WHERE project_id = ?`,
		`DELETE FROM project_queries WHERE project_id = ?`,
		`DELETE FROM projects WHERE id = ?`,
	}
	for i, stmt := range stmts {
		var execErr error
		if i == 1 {
			// report_posts DELETE needs two args (same project ID repeated).
			_, execErr = tx.ExecContext(ctx, stmt, DemoProjectID, DemoProjectID)
		} else {
			_, execErr = tx.ExecContext(ctx, stmt, DemoProjectID)
		}
		if execErr != nil {
			return execErr
		}
	}
	return nil
}

func insertDemoDataset(ctx context.Context, tx *sql.Tx) error {
	// Insert project.
	_, err := tx.ExecContext(ctx, `
		INSERT INTO projects (id, name, mode, scoring_prompt, description, selected_report_id, selected_cluster_index, created_at, updated_at)
		VALUES (?, 'Scout Demo: AI Habit Coach Research', 'research', NULL,
		        'Demo research project with realistic sample data for exploring Scout without external API keys or live provider calls.',
		        NULL, NULL, '2026-05-04 09:00:00', '2026-05-04 09:00:00')
	`, DemoProjectID)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}

	// Insert queries.
	type demoQuery struct {
		Platform string
		QueryURL string
		Angle    string
	}
	queries := []demoQuery{
		{"reddit", "https://www.reddit.com/r/productivity/search/?q=habit%20tracking%20frustrated&restrict_sr=1", "habit tracking frustration"},
		{"reddit", "https://www.reddit.com/r/getdisciplined/search/?q=accountability%20app%20quit&restrict_sr=1", "accountability drop-off"},
		{"bluesky", "habit tracker app overwhelming", "overwhelming habit apps"},
		{"bluesky", "need accountability daily routine", "daily routine accountability"},
	}
	for _, q := range queries {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO project_queries (project_id, platform, query_url, angle, enabled, created_at) VALUES (?, ?, ?, ?, 1, '2026-05-04 09:01:00')`,
			DemoProjectID, q.Platform, q.QueryURL, q.Angle,
		); err != nil {
			return fmt.Errorf("insert query: %w", err)
		}
	}

	// Insert posts.
	for _, p := range demoPosts {
		signals, _ := json.Marshal(p.TopCommentSignals)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO posts (
				id, project_id, platform, title, body, author, url, subreddit, reddit_score, num_comments,
				like_count, reply_count, repost_count, bluesky_uri, bluesky_cid, post_score, comment_score,
				final_score, angle, why, engagement_type, karma_topic, top_comment_signals, status,
				signal_type, created_at, found_at, scouted_at
			) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			p.ID, DemoProjectID, p.Platform, p.Title, p.Body, p.Author, p.URL,
			p.Subreddit, p.RedditScore, p.NumComments,
			p.LikeCount, p.ReplyCount, p.RepostCount,
			p.BlueskyURI, p.BlueskyCD,
			p.PostScore, p.CommentScore, p.FinalScore,
			p.Angle, p.Why, p.EngagementType, p.KarmaTopic,
			string(signals), p.Status, p.SignalType,
			p.CreatedAt, p.FoundAt, p.ScoutedAt,
		); err != nil {
			return fmt.Errorf("insert post %s: %w", p.ID, err)
		}
	}

	// Insert scout run.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO scout_runs (project_id, platform, started_at, completed_at, posts_checked, posts_found, status, error)
		VALUES (?, 'reddit', '2026-05-04 09:04:00', '2026-05-04 09:17:00', 12, 12, 'completed', NULL)
	`, DemoProjectID); err != nil {
		return fmt.Errorf("insert scout run: %w", err)
	}

	// Insert report.
	clustersJSON, _ := json.Marshal(demoClusters)
	res, err := tx.ExecContext(ctx, `
		INSERT INTO research_reports (project_id, title, status, post_count, clusters, assessment, model_used, created_at, completed_at, error)
		VALUES (?, 'AI Habit Coach Opportunity Report', 'completed', 12, ?, ?, 'seeded-demo', '2026-05-04 09:18:00', '2026-05-04 09:19:00', NULL)
	`, DemoProjectID,
		string(clustersJSON),
		"Strong opportunity for a lightweight, private habit accountability product. The clearest wedge is not another analytics dashboard; it is an adaptive check-in loop that remembers blockers, avoids shame after interruptions, and helps users restart. The sample evidence suggests positioning around gentle accountability for people whose routines are disrupted by real life.",
	)
	if err != nil {
		return fmt.Errorf("insert report: %w", err)
	}
	reportID, _ := res.LastInsertId()

	// Link all posts to report.
	for _, p := range demoPosts {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO report_posts (report_id, post_id) VALUES (?, ?)`,
			reportID, p.ID,
		); err != nil {
			return fmt.Errorf("insert report_post %s: %w", p.ID, err)
		}
	}

	// Update project with selected report.
	if _, err := tx.ExecContext(ctx, `
		UPDATE projects SET selected_report_id = ?, selected_cluster_index = 0, updated_at = '2026-05-04 09:19:00'
		WHERE id = ?
	`, reportID, DemoProjectID); err != nil {
		return fmt.Errorf("update project selected_report_id: %w", err)
	}

	return nil
}
