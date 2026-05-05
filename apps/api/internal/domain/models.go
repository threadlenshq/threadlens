package domain

import "encoding/json"

func IntPtr(v int64) *int64 { return &v }

type Project struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Mode                 string  `json:"mode"`
	ScoringPrompt        *string `json:"scoring_prompt"`
	Description          *string `json:"description"`
	SelectedReportID     *int64  `json:"selected_report_id"`
	SelectedClusterIndex *int64  `json:"selected_cluster_index"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}

type ProjectStats struct {
	TotalPosts   int64   `json:"total_posts"`
	NewPosts     int64   `json:"new_posts"`
	TotalQueries int64   `json:"total_queries"`
	LastRun      *string `json:"last_run"`
}

type ProjectWithStats struct {
	Project
	Stats ProjectStats `json:"stats"`
}

type Query struct {
	ID        int64  `json:"id"`
	ProjectID string `json:"project_id"`
	Platform  string `json:"platform"`
	QueryURL  string `json:"query_url"`
	Angle     string `json:"angle"`
	Enabled   int64  `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

type Prompt struct {
	ID         int64  `json:"id"`
	ProjectID  string `json:"project_id"`
	Type       string `json:"type"`
	Platform   string `json:"platform"`
	PromptText string `json:"prompt_text"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type Post struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"project_id"`
	Platform          string     `json:"platform"`
	Title             string     `json:"title"`
	Body              string     `json:"body"`
	Author            string     `json:"author"`
	URL               string     `json:"url"`
	Subreddit         *string    `json:"subreddit"`
	RedditScore       *int64     `json:"reddit_score"`
	NumComments       *int64     `json:"num_comments"`
	LikeCount         *int64     `json:"like_count"`
	ReplyCount        *int64     `json:"reply_count"`
	RepostCount       *int64     `json:"repost_count"`
	BlueskyURI        *string    `json:"bluesky_uri"`
	BlueskyCID        *string    `json:"bluesky_cid"`
	PostScore         float64    `json:"post_score"`
	CommentScore      *float64   `json:"comment_score"`
	FinalScore        float64    `json:"final_score"`
	Angle             *string    `json:"angle"`
	Why               string     `json:"why"`
	EngagementType    string     `json:"engagement_type"`
	KarmaTopic        *string    `json:"karma_topic"`
	TopCommentSignals *string    `json:"top_comment_signals"`
	Status            string     `json:"status"`
	DraftComment      *string    `json:"draft_comment"`
	DraftProvider     *string    `json:"draft_provider"`
	SignalType        *string    `json:"signal_type"`
	CreatedAt         *string    `json:"created_at"`
	FoundAt           string     `json:"found_at"`
	ScoutedAt         string     `json:"scouted_at"`
	DMTargets         []DMTarget `json:"dm_targets,omitempty"`
}

type DMTarget struct {
	ID            int64   `json:"id"`
	PostID        string  `json:"post_id"`
	Username      string  `json:"username"`
	IntentScore   float64 `json:"intent_score"`
	Signal        string  `json:"signal"`
	Context       string  `json:"context"`
	Approach      string  `json:"approach"`
	DraftDM       *string `json:"draft_dm"`
	DraftProvider *string `json:"draft_provider"`
	DMStatus      string  `json:"dm_status"`
}

type ScoutRun struct {
	ID           int64   `json:"id"`
	ProjectID    string  `json:"project_id"`
	Platform     string  `json:"platform"`
	StartedAt    string  `json:"started_at"`
	CompletedAt  *string `json:"completed_at"`
	PostsChecked int64   `json:"posts_checked"`
	PostsFound   int64   `json:"posts_found"`
	Status       string  `json:"status"`
	Error        *string `json:"error"`
	Step         *string `json:"step"`
	Warnings     *string `json:"warnings"`
}

type Schedule struct {
	ID        int64   `json:"id"`
	ProjectID string  `json:"project_id"`
	Platform  string  `json:"platform"`
	CronExpr  string  `json:"cron_expr"`
	Enabled   int64   `json:"enabled"`
	LastRunAt *string `json:"last_run_at"`
	CreatedAt string  `json:"created_at"`
}

type ResearchReport struct {
	ID                 int64           `json:"id"`
	ProjectID          string          `json:"project_id"`
	Title              string          `json:"title"`
	Status             string          `json:"status"`
	PostCount          int64           `json:"post_count"`
	Clusters           json.RawMessage `json:"clusters"`
	Assessment         string          `json:"assessment"`
	ModelUsed          string          `json:"model_used"`
	CreatedAt          string          `json:"created_at"`
	CompletedAt        *string         `json:"completed_at"`
	Error              *string         `json:"error"`
	CouncilStatus      *string         `json:"council_status,omitempty"`
	CouncilCompletedAt *string         `json:"council_completed_at,omitempty"`
	CouncilError       *string         `json:"council_error,omitempty"`
}

type GoogleReport struct {
	ID               int64           `json:"id"`
	RunID            int64           `json:"run_id"`
	ProjectID        string          `json:"project_id"`
	ExecutiveSummary json.RawMessage `json:"executive_summary"`
	KeywordSummary   json.RawMessage `json:"keyword_summary"`
	Opportunities    json.RawMessage `json:"opportunities"`
	Risks            json.RawMessage `json:"risks"`
	NextActions      json.RawMessage `json:"next_actions"`
	CreatedAt        string          `json:"created_at"`
	UpdatedAt        string          `json:"updated_at"`
	Run              *ScoutRun       `json:"run,omitempty"`
}

type Pagination struct {
	Page            int   `json:"page"`
	Limit           int   `json:"limit"`
	Total           int64 `json:"total"`
	TotalPages      int   `json:"totalPages"`
	HasPreviousPage bool  `json:"hasPreviousPage"`
	HasNextPage     bool  `json:"hasNextPage"`
}

type PagedPosts struct {
	Items      []Post     `json:"items"`
	Pagination Pagination `json:"pagination"`
}
