package domain

import "encoding/json"

const (
	QueryReviewKindSuggest = "suggest"
	QueryReviewKindRefine  = "refine"

	QueryReviewStatusRunning   = "running"
	QueryReviewStatusCompleted = "completed"
	QueryReviewStatusFailed    = "failed"

	QueryReviewResolutionApplied = "applied"
	QueryReviewResolutionDenied  = "denied"
)

type QueryReviewJob struct {
	ID          int64            `json:"id"`
	ProjectID   string           `json:"project_id"`
	Kind        string           `json:"kind"`
	Status      string           `json:"status"`
	Step        string           `json:"step"`
	Refinement  string           `json:"refinement,omitempty"`
	StartedAt   string           `json:"started_at"`
	CompletedAt *string          `json:"completed_at"`
	ReviewedAt  *string          `json:"reviewed_at"`
	Resolution  *string          `json:"resolution"`
	Error       *string          `json:"error"`
	Result      *json.RawMessage `json:"result"`
}
