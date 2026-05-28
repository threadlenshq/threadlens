package entitlements

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

type RuntimeMode string

const (
	RuntimeModeSelfHosted RuntimeMode = "self_hosted"
	RuntimeModeHosted     RuntimeMode = "hosted"
)

type Capability string

const (
	CapabilityProjectsCreate       Capability = "core.projects.create"
	CapabilityProjectsClone        Capability = "core.projects.clone"
	CapabilityProjectsGraduate     Capability = "core.projects.graduate"
	CapabilityScoutRunReddit       Capability = "core.scout.run.reddit"
	CapabilityScoutRunBluesky      Capability = "core.scout.run.bluesky"
	CapabilityScoutRunGoogle       Capability = "core.scout.run.google"
	CapabilityReportsCreate        Capability = "core.reports.create"
	CapabilityModelsConfigure      Capability = "core.models.configure"
	CapabilityManagedAIUse         Capability = "ai.managed_provider.use"
	CapabilityPromptTemplatesList  Capability = "templates.prompt_pack.list"
	CapabilityPromptTemplatesApply Capability = "templates.prompt_pack.apply"
	CapabilitySupportActive        Capability = "support.maintenance.active"
	CapabilityTenantsManage        Capability = "hosted.tenants.manage"
)

var CoreCapabilities = []Capability{
	CapabilityProjectsCreate,
	CapabilityProjectsClone,
	CapabilityProjectsGraduate,
	CapabilityScoutRunReddit,
	CapabilityScoutRunBluesky,
	CapabilityScoutRunGoogle,
	CapabilityReportsCreate,
	CapabilityModelsConfigure,
	CapabilityPromptTemplatesList,
}

type Subject struct {
	ActorID     string      `json:"actorId"`
	TenantID    string      `json:"tenantId"`
	ProjectID   string      `json:"projectId,omitempty"`
	RuntimeMode RuntimeMode `json:"runtimeMode"`
}

type CheckRequest struct {
	Subject    Subject    `json:"subject"`
	Capability Capability `json:"capability"`
	ProjectID  string     `json:"projectId,omitempty"`
	Action     string     `json:"action,omitempty"`
}

type Limit struct {
	Key       string `json:"key"`
	Unit      string `json:"unit"`
	Used      int64  `json:"used"`
	Limit     int64  `json:"limit"`
	Unlimited bool   `json:"unlimited"`
}

type Decision struct {
	Allowed    bool       `json:"allowed"`
	Capability Capability `json:"capability"`
	Reason     string     `json:"reason,omitempty"`
	StatusCode int        `json:"statusCode,omitempty"`
	Limit      *Limit     `json:"limit,omitempty"`
}

type Snapshot struct {
	RuntimeMode  RuntimeMode         `json:"runtimeMode"`
	Edition      string              `json:"edition"`
	ActorID      string              `json:"actorId"`
	TenantID     string              `json:"tenantId"`
	Capabilities map[Capability]bool `json:"capabilities"`
	Limits       map[string]Limit    `json:"limits"`
	Support      SupportStatus       `json:"support"`
	Modules      []ModuleStatus      `json:"modules"`
	Messages     []RuntimeMessage    `json:"messages"`
}

type SupportStatus struct {
	Available         bool   `json:"available"`
	MaintenanceActive bool   `json:"maintenanceActive"`
	Level             string `json:"level"`
}

type ModuleStatus struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type RuntimeMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Level   string `json:"level"`
}

type Resolver interface {
	Snapshot(ctx context.Context, subject Subject) (Snapshot, error)
	Check(ctx context.Context, req CheckRequest) (Decision, error)
}

type LocalResolver struct {
	runtimeMode RuntimeMode
	modules     []ModuleStatus
}

func NewLocalResolver(runtimeMode RuntimeMode, modules []ModuleStatus) *LocalResolver {
	if runtimeMode == "" {
		runtimeMode = RuntimeModeSelfHosted
	}
	copyModules := append([]ModuleStatus(nil), modules...)
	return &LocalResolver{runtimeMode: runtimeMode, modules: copyModules}
}

func (r *LocalResolver) Snapshot(_ context.Context, subject Subject) (Snapshot, error) {
	mode := subject.RuntimeMode
	if mode == "" {
		mode = r.runtimeMode
	}
	capabilities := map[Capability]bool{}
	for _, cap := range CoreCapabilities {
		capabilities[cap] = true
	}
	googleScoutConfigured := strings.TrimSpace(os.Getenv("PARALLEL_API_KEY")) != ""
	capabilities[CapabilityScoutRunGoogle] = googleScoutConfigured
	capabilities[CapabilityManagedAIUse] = false
	capabilities[CapabilityPromptTemplatesApply] = true
	capabilities[CapabilitySupportActive] = false
	capabilities[CapabilityTenantsManage] = false

	modules := append([]ModuleStatus(nil), r.modules...)
	sort.Slice(modules, func(i, j int) bool { return modules[i].ID < modules[j].ID })

	edition := "Open Core"
	if mode == RuntimeModeHosted {
		edition = "Hosted"
	}

	message := RuntimeMessage{
		Code:    "open_core_local_entitlements",
		Message: "Core ThreadLens capabilities are enabled by the local entitlement resolver. Premium and hosted capabilities still require server-side checks.",
		Level:   "info",
	}
	if mode == RuntimeModeHosted {
		message = RuntimeMessage{
			Code:    "hosted_local_resolver",
			Message: "Hosted runtime mode is active with the local resolver. Protected hosted-only capabilities remain disabled until hosted adapters are wired.",
			Level:   "warning",
		}
	}

	messages := []RuntimeMessage{message}
	if !googleScoutConfigured {
		messages = append(messages, RuntimeMessage{
			Code:    "google_parallel_api_key_missing",
			Message: "Google Scout is disabled: PARALLEL_API_KEY is not set. Configure the key to enable Google search scouting.",
			Level:   "warning",
		})
	}

	return Snapshot{
		RuntimeMode:  mode,
		Edition:      edition,
		ActorID:      defaultString(subject.ActorID, "local-user"),
		TenantID:     defaultString(subject.TenantID, "local-instance"),
		Capabilities: capabilities,
		Limits: map[string]Limit{
			"core.projects":           {Key: "core.projects", Unit: "projects", Unlimited: true},
			"ai.usage.monthly_tokens": {Key: "ai.usage.monthly_tokens", Unit: "tokens", Unlimited: true},
		},
		Support:  SupportStatus{Available: false, MaintenanceActive: false, Level: "community"},
		Modules:  modules,
		Messages: messages,
	}, nil
}

func (r *LocalResolver) Check(ctx context.Context, req CheckRequest) (Decision, error) {
	snapshot, err := r.Snapshot(ctx, req.Subject)
	if err != nil {
		return Decision{}, err
	}
	if snapshot.Capabilities[req.Capability] {
		return Decision{Allowed: true, Capability: req.Capability}, nil
	}
	return Decision{
		Allowed:    false,
		Capability: req.Capability,
		Reason:     "capability_not_granted",
		StatusCode: http.StatusPaymentRequired,
	}, nil
}

type DeniedError struct {
	Decision Decision
}

func (e *DeniedError) Error() string {
	if e.Decision.Reason == "" {
		return fmt.Sprintf("capability denied: %s", e.Decision.Capability)
	}
	return fmt.Sprintf("capability denied: %s: %s", e.Decision.Capability, e.Decision.Reason)
}

func EnsureAllowed(decision Decision) error {
	if decision.Allowed {
		return nil
	}
	if decision.StatusCode == 0 {
		decision.StatusCode = http.StatusForbidden
	}
	if decision.Reason == "" {
		decision.Reason = "capability_not_granted"
	}
	return &DeniedError{Decision: decision}
}

func StatusCode(err error) int {
	var denied *DeniedError
	if errors.As(err, &denied) {
		if denied.Decision.StatusCode != 0 {
			return denied.Decision.StatusCode
		}
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

func CapabilityForScoutPlatform(platform string) Capability {
	switch platform {
	case "reddit":
		return CapabilityScoutRunReddit
	case "bluesky":
		return CapabilityScoutRunBluesky
	case "google":
		return CapabilityScoutRunGoogle
	default:
		return Capability("core.scout.run." + platform)
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
