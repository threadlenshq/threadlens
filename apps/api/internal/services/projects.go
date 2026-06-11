package services

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
	"github.com/kyle/scout/open-core/apps/api/internal/tenant"
)

type ProjectService struct {
	repo     *repository.Repository
	mode     entitlements.RuntimeMode
	resolver entitlements.Resolver
}

func NewProjectService(repo *repository.Repository, mode entitlements.RuntimeMode, resolver entitlements.Resolver) *ProjectService {
	return &ProjectService{repo: repo, mode: mode, resolver: resolver}
}

type CreateProjectRequest struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Mode        string  `json:"mode"`
	Description *string `json:"description"`
}

type CloneProjectRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SelectAngleRequest struct {
	ReportID     *int64 `json:"report_id"`
	ClusterIndex *int64 `json:"cluster_index"`
}

func (s *ProjectService) List(ctx context.Context) ([]domain.Project, error) {
	return s.repo.ListProjects(ctx)
}

func (s *ProjectService) Create(ctx context.Context, body CreateProjectRequest) (domain.Project, int, string) {
	if status, msg := s.ensure(ctx, entitlements.CapabilityProjectsCreate, body.ID, "projects.create"); msg != "" {
		return domain.Project{}, status, msg
	}
	p, err := s.repo.CreateProject(ctx, domain.Project{
		ID:          body.ID,
		Name:        body.Name,
		Mode:        body.Mode,
		Description: body.Description,
	})
	if err != nil {
		code, msg := mapError(err)
		return domain.Project{}, code, msg
	}
	return p, http.StatusCreated, ""
}

func (s *ProjectService) Get(ctx context.Context, id string) (domain.ProjectWithStats, int, string) {
	p, err := s.repo.GetProjectWithStats(ctx, id)
	if err != nil {
		code, msg := mapError(err)
		return domain.ProjectWithStats{}, code, msg
	}
	return p, http.StatusOK, ""
}

func (s *ProjectService) Patch(ctx context.Context, id string, body map[string]any) (domain.Project, int, string) {
	p, err := s.repo.PatchProject(ctx, id, body)
	if err != nil {
		code, msg := mapError(err)
		return domain.Project{}, code, msg
	}
	return p, http.StatusOK, ""
}

func (s *ProjectService) Delete(ctx context.Context, id string) (int, string) {
	err := s.repo.DeleteProject(ctx, id)
	if err != nil {
		return mapError(err)
	}
	return http.StatusNoContent, ""
}

func (s *ProjectService) Clone(ctx context.Context, id string, body CloneProjectRequest) (domain.Project, int, string) {
	if status, msg := s.ensure(ctx, entitlements.CapabilityProjectsClone, id, "projects.clone"); msg != "" {
		return domain.Project{}, status, msg
	}
	p, err := s.repo.CloneProject(ctx, id, body.ID, body.Name)
	if err != nil {
		code, msg := mapError(err)
		return domain.Project{}, code, msg
	}
	return p, http.StatusCreated, ""
}

func (s *ProjectService) SelectAngle(ctx context.Context, id string, body SelectAngleRequest) (domain.Project, int, string) {
	if body.ReportID == nil || body.ClusterIndex == nil {
		return domain.Project{}, http.StatusBadRequest, "report_id and cluster_index are required"
	}
	p, err := s.repo.SelectAngle(ctx, id, *body.ReportID, *body.ClusterIndex)
	if err != nil {
		code, msg := mapError(err)
		return domain.Project{}, code, msg
	}
	return p, http.StatusOK, ""
}

func (s *ProjectService) Graduate(ctx context.Context, id string) (domain.Project, int, string) {
	if status, msg := s.ensure(ctx, entitlements.CapabilityProjectsGraduate, id, "projects.graduate"); msg != "" {
		return domain.Project{}, status, msg
	}
	p, err := s.repo.GraduateProject(ctx, id)
	if err != nil {
		code, msg := mapError(err)
		return domain.Project{}, code, msg
	}
	return p, http.StatusOK, ""
}

// mapEntityError is like mapError but substitutes a caller-specific message
// for the generic "not found" sentinel (e.g. "Post not found").
func mapEntityError(err error, notFoundMsg string) (int, string) {
	code, msg := mapError(err)
	if msg == "not found" {
		msg = notFoundMsg
	}
	return code, msg
}

// mapError converts repository errors to HTTP status codes and messages.
func mapError(err error) (int, string) {
	msg := err.Error()
	// Strip "sentinel: " prefix that fmt.Errorf("%w: msg") produces
	for _, prefix := range []string{"not found: ", "validation: ", "conflict: "} {
		msg = strings.TrimPrefix(msg, prefix)
	}
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound, msg
	case errors.Is(err, repository.ErrValidation):
		return http.StatusBadRequest, msg
	case errors.Is(err, repository.ErrConflict):
		return http.StatusConflict, msg
	default:
		return http.StatusInternalServerError, "Internal server error"
	}
}

func (s *ProjectService) ensure(ctx context.Context, capability entitlements.Capability, projectID string, action string) (int, string) {
	decision, err := s.resolver.Check(ctx, entitlements.CheckRequest{Subject: tenant.SubjectFromContext(ctx, s.mode), Capability: capability, ProjectID: projectID, Action: action})
	if err != nil {
		return http.StatusInternalServerError, "Internal server error"
	}
	if err := entitlements.EnsureAllowed(decision); err != nil {
		return entitlements.StatusCode(err), err.Error()
	}
	return http.StatusOK, ""
}
