package domain

import "encoding/json"

const (
	FilterStateVisible  = "visible"
	FilterStateFiltered = "filtered"

	FilterSourceNone            = "none"
	FilterSourceRules           = "rules"
	FilterSourceAI              = "ai"
	FilterSourceTrustedOverride = "trusted_override"

	FilterReasonSpam              = "spam"
	FilterReasonBotLike           = "bot_like"
	FilterReasonLowQualityAccount = "low_quality_account"
	FilterReasonAIGenerated       = "ai_generated"
	FilterReasonTrustedOverride   = "trusted_override"

	FindingTypePost         = "post"
	FindingTypeGoogleResult = "google_result"

	TrustTypeSource          = "source"
	TrustTypeFilterSignature = "filter_signature"

	FilterJobStatusRunning   = "running"
	FilterJobStatusCompleted = "completed"
	FilterJobStatusFailed    = "failed"

	FilterJobScopeSelectedVisiblePosts = "selected_visible_posts"
	FilterJobScopeSelectedFiltered     = "selected_filtered_findings"
	FilterJobScopeSelectedGoogle       = "selected_google_results"
)

type SourceIdentity map[string]string

func (s SourceIdentity) JSON() string {
	if s == nil {
		return "{}"
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "{}"
	}
	return string(b)
}

type FilterMetadata struct {
	FilterState       string         `json:"filter_state"`
	FilterReason      *string        `json:"filter_reason"`
	FilterReasons     []string       `json:"filter_reasons"`
	FilterExplanation string         `json:"filter_explanation"`
	FilterConfidence  *float64       `json:"filter_confidence"`
	FilterSource      string         `json:"filter_source"`
	FilterSignature   string         `json:"filter_signature"`
	FilterJobID       *int64         `json:"filter_job_id"`
	FilteredAt        *string        `json:"filtered_at"`
	RecoveredAt       *string        `json:"recovered_at"`
	RecoveryNote      *string        `json:"recovery_note"`
	SourceIdentity    SourceIdentity `json:"source_identity"`
}

type FilterDecision struct {
	State          string
	Reason         string
	Reasons        []string
	Explanation    string
	Confidence     *float64
	Source         string
	Signature      string
	SourceIdentity SourceIdentity
	AIUsed         bool
	Warning        string
}

type TrustRecord struct {
	ID         int64  `json:"id"`
	ProjectID  string `json:"project_id"`
	Platform   string `json:"platform"`
	TrustType  string `json:"trust_type"`
	SourceKind string `json:"source_kind"`
	SourceKey  string `json:"source_key"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"created_at"`
	CreatedBy  string `json:"created_by"`
}

type FilterTrustOption struct {
	Platform   string `json:"platform"`
	TrustType  string `json:"trust_type"`
	SourceKind string `json:"source_kind"`
	SourceKey  string `json:"source_key"`
	Label      string `json:"label"`
}

type FilteredFinding struct {
	ID                string              `json:"id"`
	FindingType       string              `json:"finding_type"`
	ProjectID         string              `json:"project_id"`
	Platform          string              `json:"platform"`
	Title             string              `json:"title"`
	Snippet           string              `json:"snippet"`
	URL               string              `json:"url"`
	SourceIdentity    SourceIdentity      `json:"source_identity"`
	Score             *float64            `json:"score"`
	FilterState       string              `json:"filter_state"`
	FilterReason      *string             `json:"filter_reason"`
	FilterReasons     []string            `json:"filter_reasons"`
	FilterExplanation string              `json:"filter_explanation"`
	FilterConfidence  *float64            `json:"filter_confidence"`
	FilterSource      string              `json:"filter_source"`
	FilterSignature   string              `json:"filter_signature"`
	FilterJobID       *int64              `json:"filter_job_id"`
	FilteredAt        *string             `json:"filtered_at"`
	RecoveredAt       *string             `json:"recovered_at"`
	RecoveryNote      *string             `json:"recovery_note"`
	TrustOptions      []FilterTrustOption `json:"trust_options"`
}

type PagedFilteredFindings struct {
	Items      []FilteredFinding `json:"items"`
	Pagination Pagination        `json:"pagination"`
}

type FilterJobTarget struct {
	FindingType string `json:"finding_type"`
	ID          string `json:"id"`
}

type FilterJobResult struct {
	Filtered        int64             `json:"filtered"`
	RestoredByTrust int64             `json:"restored_by_trust"`
	Unchanged       int64             `json:"unchanged"`
	Failed          int64             `json:"failed"`
	Errors          map[string]string `json:"errors"`
}

type FilterJob struct {
	ID             int64             `json:"id"`
	ProjectID      string            `json:"project_id"`
	Status         string            `json:"status"`
	Step           *string           `json:"step"`
	RequestedScope string            `json:"requested_scope"`
	Targets        []FilterJobTarget `json:"targets"`
	Result         *FilterJobResult  `json:"result"`
	Error          *string           `json:"error"`
	StartedAt      string            `json:"started_at"`
	CompletedAt    *string           `json:"completed_at"`
}
