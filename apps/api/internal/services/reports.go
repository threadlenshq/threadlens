package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/pipeline"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// ReportService handles business logic for research reports.
type ReportService struct {
	repo  *repository.Repository
	db    *sql.DB
	aiSvc *ai.Service
}

// NewReportService creates a new ReportService.
func NewReportService(repo *repository.Repository, db *sql.DB, aiSvc *ai.Service) *ReportService {
	return &ReportService{repo: repo, db: db, aiSvc: aiSvc}
}

// ListReports returns all reports for a project.
func (s *ReportService) ListReports(ctx context.Context, projectID string) ([]domain.ResearchReport, int, string) {
	reports, err := s.repo.ListReports(ctx, projectID)
	if err != nil {
		return nil, http.StatusInternalServerError, "Internal server error"
	}
	return reports, http.StatusOK, ""
}

// GetReport returns a single report by ID.
func (s *ReportService) GetReport(ctx context.Context, projectID string, reportID int64) (domain.ResearchReport, int, string) {
	rep, err := s.repo.GetReport(ctx, projectID, reportID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Report not found"
		}
		return domain.ResearchReport{}, code, msg
	}
	return rep, http.StatusOK, ""
}

// GetReportCouncil returns the parsed council JSON for a report.
func (s *ReportService) GetReportCouncil(ctx context.Context, projectID string, reportID int64) (json.RawMessage, int, string) {
	raw, err := s.repo.GetReportCouncilJSON(ctx, projectID, reportID)
	if err != nil {
		code, msg := mapError(err)
		if msg == "not found" {
			msg = "Council not found"
		}
		return nil, code, msg
	}
	return raw, http.StatusOK, ""
}

// CreateReportRequest is the deserialized body for POST /reports.
type CreateReportRequest struct {
	MinScore *float64 `json:"min_score"`
	DateFrom string   `json:"date_from"`
	DateTo   string   `json:"date_to"`
}

// StartReport creates a running report record, kicks off analysis in the background,
// and returns the initial report immediately (status: running).
func (s *ReportService) StartReport(ctx context.Context, projectID string, req CreateReportRequest) (domain.ResearchReport, int, string) {
	// Resolve the model to record in the row.
	resolved, err := ResolveTaskModel(ctx, s.repo, "report_clustering")
	if err != nil {
		return domain.ResearchReport{}, http.StatusInternalServerError, err.Error()
	}

	reportID, err := s.repo.StartAnalysis(ctx, projectID, resolved.ModelID)
	if err != nil {
		return domain.ResearchReport{}, http.StatusInternalServerError, "Internal server error"
	}

	// Fetch the newly created row to return.
	rep, err := s.repo.GetReport(ctx, projectID, reportID)
	if err != nil {
		return domain.ResearchReport{}, http.StatusInternalServerError, "Internal server error"
	}

	// Run analysis in background.
	go func() {
		bgCtx := context.Background()
		opts := pipeline.AnalysisOptions{
			MinScore: req.MinScore,
			DateFrom: req.DateFrom,
			DateTo:   req.DateTo,
		}
		if err := pipeline.RunAnalysis(bgCtx, s.db, s.aiSvc, s.repo, projectID, reportID, opts); err != nil {
			// RunAnalysis already marks the report failed internally; log unexpected errors.
			_ = s.repo.MarkReportFailed(bgCtx, reportID, err.Error())
		}
	}()

	return rep, http.StatusCreated, ""
}
