package onboarding

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// StarterProjectRequest carries input for CreateStarterProject.
type StarterProjectRequest struct {
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
	Query       string `json:"query"`
	Platform    string `json:"platform"`
	Description string `json:"description"`
}

// StarterProjectResult carries output from CreateStarterProject.
type StarterProjectResult struct {
	Project        domain.Project `json:"project"`
	Query          domain.Query   `json:"query"`
	CreatedProject bool           `json:"createdProject"`
	CreatedQuery   bool           `json:"createdQuery"`
}

// createStarterProject implements the idempotent starter project/query creation
// logic on behalf of Service.CreateStarterProject.
func createStarterProject(ctx context.Context, s *Service, req StarterProjectRequest) (StarterProjectResult, error) {
	if s.cfg.Disabled {
		return StarterProjectResult{}, ErrDisabled
	}
	if s.projectRepo == nil {
		return StarterProjectResult{}, errors.New("onboarding: projectRepo must not be nil")
	}

	// Trim and validate inputs.
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	req.ProjectName = strings.TrimSpace(req.ProjectName)
	req.Query = strings.TrimSpace(req.Query)
	req.Platform = strings.TrimSpace(req.Platform)
	if req.ProjectID == "" {
		return StarterProjectResult{}, errors.New("onboarding: StarterProjectRequest.ProjectID must not be empty")
	}
	if req.ProjectName == "" {
		return StarterProjectResult{}, errors.New("onboarding: StarterProjectRequest.ProjectName must not be empty")
	}
	if req.Query == "" {
		return StarterProjectResult{}, errors.New("onboarding: StarterProjectRequest.Query must not be empty")
	}
	if req.Platform == "" {
		req.Platform = "reddit"
	}

	// Create or reuse the project.
	project, createdProject, err := getOrCreateProject(ctx, s.projectRepo, req.ProjectID, req.ProjectName)
	if err != nil {
		return StarterProjectResult{}, fmt.Errorf("onboarding: get or create project: %w", err)
	}

	// Persist description if provided (non-empty after trim).
	desc := strings.TrimSpace(req.Description)
	if desc != "" {
		if utf8.RuneCountInString(desc) > 1000 {
			runes := []rune(desc)
			desc = string(runes[:1000])
		}
		patched, patchErr := s.projectRepo.PatchProject(ctx, project.ID, map[string]any{"description": desc})
		if patchErr != nil {
			return StarterProjectResult{}, fmt.Errorf("onboarding: patch project description: %w", patchErr)
		}
		project.Description = patched.Description
		project.UpdatedAt = patched.UpdatedAt
	}

	// Normalize the query value into a valid platform-specific format.
	queryURL := normalizeQueryURL(req.Platform, req.Query)

	// Create or reuse the query.
	query, createdQuery, err := getOrCreateQuery(ctx, s.projectRepo, project.ID, req.Platform, queryURL)
	if err != nil {
		return StarterProjectResult{}, fmt.Errorf("onboarding: get or create query: %w", err)
	}

	// Update onboarding progress to reflect the starter items.
	if err := updateProgressForStarter(ctx, s, project.ID, query.ID); err != nil {
		return StarterProjectResult{}, fmt.Errorf("onboarding: update progress: %w", err)
	}

	return StarterProjectResult{
		Project:        project,
		Query:          query,
		CreatedProject: createdProject,
		CreatedQuery:   createdQuery,
	}, nil
}

// getOrCreateProject fetches the project by ID if it exists, or creates it.
// Returns the project and whether it was newly created.
func getOrCreateProject(ctx context.Context, repo *repository.Repository, id, name string) (domain.Project, bool, error) {
	existing, err := repo.GetProject(ctx, id)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return domain.Project{}, false, err
	}
	created, err := repo.CreateProject(ctx, domain.Project{
		ID:   id,
		Name: name,
		Mode: "research",
	})
	if err != nil {
		return domain.Project{}, false, err
	}
	return created, true, nil
}

// getOrCreateQuery finds an existing query matching platform+queryURL on the
// project, or creates a new one. Returns the query and whether it was created.
func getOrCreateQuery(ctx context.Context, repo *repository.Repository, projectID, platform, queryURL string) (domain.Query, bool, error) {
	queries, err := repo.ListAllQueries(ctx, projectID)
	if err != nil {
		return domain.Query{}, false, err
	}
	for _, q := range queries {
		if q.Platform == platform && q.QueryURL == queryURL {
			return q, false, nil
		}
	}
	created, err := repo.CreateQuery(ctx, projectID, platform, queryURL, "", true)
	if err != nil {
		return domain.Query{}, false, err
	}
	return created, true, nil
}

// updateProgressForStarter marks the starter_project and starter_query
// exploration items as completed, records the project/query IDs in context,
// and persists the updated progress.
func updateProgressForStarter(ctx context.Context, s *Service, projectID string, queryID int64) error {
	p, err := s.loadProgress(ctx)
	if err != nil {
		return err
	}

	// Record context IDs.
	p.Context.StarterProjectID = projectID
	p.Context.StarterQueryID = queryID
	p.Context.SelectedProjectID = projectID

	// Mark starter items complete (idempotent — don't regress terminal states).
	markComplete := func(item ExplorationItem) {
		if state := p.Exploration.Items[item]; state != ItemStateCompleted {
			p.Exploration.Items[item] = ItemStateCompleted
		}
	}
	markComplete(ExplorationItemStarterProject)
	markComplete(ExplorationItemStarterQuery)

	// Recompute current item and status.
	next := firstPendingItem(p.Exploration.Items)
	p.Exploration.CurrentItem = next
	if next == "" && p.Exploration.Status != ExplorationStatusComplete {
		p.Exploration.Status = ExplorationStatusComplete
	} else if next != "" {
		// At least one non-terminal item remains — keep active.
		if p.Exploration.Status == ExplorationStatusNotStarted {
			p.Exploration.Status = ExplorationStatusActive
		}
	}

	return s.saveProgress(ctx, p)
}

// normalizeQueryURL converts a user-entered query string into a
// platform-appropriate format. For Reddit, plain-text keywords are turned
// into a site-wide search JSON URL. Other platforms use the raw value.
func normalizeQueryURL(platform, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if platform != "reddit" {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	// Build a Reddit site-wide search URL with default params.
	return fmt.Sprintf(
		"https://www.reddit.com/search.json?q=%s&sort=new&t=month&limit=100",
		url.QueryEscape(trimmed),
	)
}
