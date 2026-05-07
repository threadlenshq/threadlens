package templates

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/entitlements"
)

type PromptPack struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Platforms   []string `json:"platforms"`
	Free        bool     `json:"free"`
}

type ApplyRequest struct {
	PackID    string `json:"packId"`
	ProjectID string `json:"projectId"`
}

type ApplyResult struct {
	PackID         string `json:"packId"`
	ProjectID      string `json:"projectId"`
	PromptsCreated int    `json:"promptsCreated"`
	QueriesCreated int    `json:"queriesCreated"`
}

type Catalog interface {
	List(ctx context.Context, subject entitlements.Subject) ([]PromptPack, error)
	Apply(ctx context.Context, subject entitlements.Subject, req ApplyRequest) (ApplyResult, error)
}

type LocalCatalog struct {
	resolver entitlements.Resolver
}

var ErrNilResolver = errors.New("templates: resolver must not be nil")

func NewLocalCatalog(resolver entitlements.Resolver) *LocalCatalog {
	if resolver == nil {
		panic(ErrNilResolver)
	}
	return &LocalCatalog{resolver: resolver}
}

func (c *LocalCatalog) List(ctx context.Context, subject entitlements.Subject) ([]PromptPack, error) {
	decision, err := c.resolver.Check(ctx, entitlements.CheckRequest{Subject: subject, Capability: entitlements.CapabilityPromptTemplatesList, Action: "templates.list"})
	if err != nil {
		return nil, err
	}
	if err := entitlements.EnsureAllowed(decision); err != nil {
		return nil, err
	}
	return []PromptPack{}, nil
}

func (c *LocalCatalog) Apply(ctx context.Context, subject entitlements.Subject, req ApplyRequest) (ApplyResult, error) {
	if strings.TrimSpace(req.PackID) == "" {
		return ApplyResult{}, fmt.Errorf("packId is required")
	}
	if strings.TrimSpace(req.ProjectID) == "" {
		return ApplyResult{}, fmt.Errorf("projectId is required")
	}
	decision, err := c.resolver.Check(ctx, entitlements.CheckRequest{Subject: subject, Capability: entitlements.CapabilityPromptTemplatesApply, ProjectID: req.ProjectID, Action: "templates.apply"})
	if err != nil {
		return ApplyResult{}, err
	}
	if err := entitlements.EnsureAllowed(decision); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{PackID: req.PackID, ProjectID: req.ProjectID, PromptsCreated: 0, QueriesCreated: 0}, nil
}
