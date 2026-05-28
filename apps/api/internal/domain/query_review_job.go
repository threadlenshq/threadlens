package domain

import "encoding/json"

type QueryReviewKind string
type QueryReviewStatus string
type QueryReviewResolution string

const (
	QueryReviewKindSuggest QueryReviewKind = "suggest"
	QueryReviewKindRefine  QueryReviewKind = "refine"

	QueryReviewStatusRunning   QueryReviewStatus = "running"
	QueryReviewStatusCompleted QueryReviewStatus = "completed"
	QueryReviewStatusFailed    QueryReviewStatus = "failed"

	QueryReviewResolutionApplied QueryReviewResolution = "applied"
	QueryReviewResolutionDenied  QueryReviewResolution = "denied"
)

type QueryReviewJob struct {
	ID          int64                  `json:"id"`
	ProjectID   string                 `json:"project_id"`
	Kind        QueryReviewKind        `json:"kind"`
	Status      QueryReviewStatus      `json:"status"`
	Step        string                 `json:"step"`
	Refinement  string                 `json:"refinement,omitempty"`
	StartedAt   string                 `json:"started_at"`
	CompletedAt *string                `json:"completed_at"`
	ReviewedAt  *string                `json:"reviewed_at"`
	Resolution  *QueryReviewResolution `json:"resolution"`
	Error       *string                `json:"error"`
	Result      json.RawMessage        `json:"result"`
}
