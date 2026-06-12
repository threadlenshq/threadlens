// Package telemetry provides an anonymous, opt-in telemetry recorder for the
// Scout API. Events are queued in memory, batched, and flushed to a remote
// Cloudflare Worker ingest endpoint. The recorder is a no-op when the
// infrastructure-level opt-in (SCOUT_TELEMETRY_OPT_IN) is not set.
package telemetry

import "time"

// EventName is a closed enum of telemetry event names. The worker rejects
// any event_name not in this set.
type EventName string

const (
	EventInstanceStarted        EventName = "instance_started"
	EventInstancePing           EventName = "instance_ping"
	EventInstanceConsentChanged EventName = "instance_consent_changed"

	EventFeatureScoutRun       EventName = "feature_used:scout_run"
	EventFeatureQuerySuggest   EventName = "feature_used:query_suggest"
	EventFeatureReportCreate  EventName = "feature_used:report_create"
	EventFeatureScheduleCreate EventName = "feature_used:schedule_create"
	EventFeatureFilterJob     EventName = "feature_used:filter_job"

	EventErrorOnboardingSave EventName = "error:onboarding_save"
	EventErrorScoutRun       EventName = "error:scout_run"
)

// validEventNames is the set of all allowed event names. Used by the
// recorder to reject unknown names at enqueue time.
var validEventNames = map[EventName]bool{
	EventInstanceStarted:         true,
	EventInstancePing:            true,
	EventInstanceConsentChanged:  true,
	EventFeatureScoutRun:         true,
	EventFeatureQuerySuggest:     true,
	EventFeatureReportCreate:     true,
	EventFeatureScheduleCreate:   true,
	EventFeatureFilterJob:        true,
	EventErrorOnboardingSave:     true,
	EventErrorScoutRun:           true,
}

// IsValidEventName reports whether name is in the closed allow-list.
func IsValidEventName(name EventName) bool {
	return validEventNames[name]
}

// Event is a single telemetry event on the wire. The struct has a fixed
// shape; no extra fields are permitted.
type Event struct {
	EventName       EventName `json:"event_name"`
	EventTimeUnixMs int64     `json:"event_time_unix_ms"`
	ScoutVersion    string    `json:"scout_version"`
	DeploymentType  string    `json:"deployment_type"`
	OSPlatform      string    `json:"os_platform"`
	Source          string    `json:"source"`
}

// Batch is the wire format sent to the worker.
type Batch struct {
	InstanceID string  `json:"instance_id"`
	Events     []Event `json:"events"`
}

// NewEvent creates an Event with the current timestamp and the provided
// metadata fields pre-populated.
func NewEvent(name EventName, version, deploymentType, osPlatform, source string) Event {
	return Event{
		EventName:       name,
		EventTimeUnixMs: time.Now().UnixMilli(),
		ScoutVersion:    version,
		DeploymentType:  deploymentType,
		OSPlatform:      osPlatform,
		Source:          source,
	}
}
