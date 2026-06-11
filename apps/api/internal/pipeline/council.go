package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/ai"
	"github.com/kyle/scout/open-core/apps/api/internal/repository"
)

// advisorDef defines one council advisor.
type advisorDef struct {
	Role string
	Lens string
}

var advisors = []advisorDef{
	{Role: "Contrarian", Lens: "Question the assumptions. What is most likely to fail? What is the downside risk?"},
	{Role: "First Principles", Lens: "Strip away context. What is actually being solved? Is the framing correct?"},
	{Role: "Expansionist", Lens: "Look for adjacent opportunities. What upside is being missed?"},
	{Role: "Outsider", Lens: "Fresh eyes. What would someone with no context immediately notice?"},
	{Role: "Executor", Lens: "Practical next actions. What is the fastest path to validating this?"},
}

const advisorSystemPromptTemplate = `You are an expert advisor reviewing a research report about product opportunities.
You play the role of [ADVISOR_ROLE] and provide your unique perspective.
Respond only with valid JSON matching the exact schema provided.`

const synthesisSystemPrompt = `You are the chair of an advisor council reviewing a research report. You have received perspectives from five advisors. Synthesize their views into a final verdict.
Respond only with valid JSON matching the exact schema provided.`

// dbReport is a minimal report struct for council queries.
type dbReport struct {
	Title      string
	Assessment string
	Clusters   string
}

func getReportForCouncil(ctx context.Context, db *sql.DB, reportID int64) (dbReport, error) {
	var rep dbReport
	err := db.QueryRowContext(ctx, "SELECT title, assessment, clusters FROM research_reports WHERE id = ?", reportID).
		Scan(&rep.Title, &rep.Assessment, &rep.Clusters)
	return rep, err
}

func buildAdvisorUserMessage(adv advisorDef, rep dbReport) string {
	// Pretty-print clusters if valid JSON, else use raw string.
	var clustersText string
	var parsed any
	if err := json.Unmarshal([]byte(rep.Clusters), &parsed); err == nil {
		b, _ := json.MarshalIndent(parsed, "", "  ")
		clustersText = string(b)
	} else {
		clustersText = rep.Clusters
	}

	return fmt.Sprintf(`You are the %s advisor.

Report title: %s
Assessment: %s
Clusters: %s

Your lens: %s

Respond with ONLY this JSON schema:
{
  "lens": "%s",
  "key_claim": "One sentence thesis",
  "critique": ["point 1", "point 2"],
  "blind_spots": ["blind spot 1"],
  "risks": ["risk 1"],
  "questions_to_test": ["question 1"]
}`, adv.Role, rep.Title, rep.Assessment, clustersText, adv.Lens, adv.Role)
}

func buildSynthesisUserMessage(rep dbReport, advisorOutputs []json.RawMessage) string {
	outputsJSON, _ := json.MarshalIndent(advisorOutputs, "", "  ")
	return fmt.Sprintf(`Report title: %s
Assessment: %s

Advisor outputs:
%s

Synthesize the council advice into this JSON schema:
{
  "agreements": ["shared conclusion 1"],
  "conflicts": ["advisor conflict 1"],
  "blind_spots": ["missed angle 1"],
  "chair_verdict": "One sentence verdict",
  "next_experiment": {
    "hypothesis": "What should be tested",
    "test": "Concrete next step",
    "success_signal": "What success looks like"
  }
}`, rep.Title, rep.Assessment, outputsJSON)
}

// stripMarkdownFences removes a surrounding ```json ... ``` code fence that
// models sometimes wrap JSON responses in, returning the trimmed inner text.
func stripMarkdownFences(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}
	return cleaned
}

func parseCouncilJSON(raw string) (json.RawMessage, error) {
	cleaned := stripMarkdownFences(raw)
	if !json.Valid([]byte(cleaned)) {
		return nil, fmt.Errorf("invalid JSON from AI: %s", cleaned[:min(len(cleaned), 200)])
	}
	return json.RawMessage(cleaned), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StartCouncil inserts a report_councils row with status='running' and returns the council ID.
func StartCouncil(ctx context.Context, db *sql.DB, repo *repository.Repository, projectID string, reportID int64) (int64, error) {
	// Resolve the model for advisor_council task.
	resolved, err := resolveCouncilModel(ctx, repo)
	if err != nil {
		return 0, err
	}
	res, err := db.ExecContext(ctx,
		`INSERT INTO report_councils (report_id, project_id, status, model_used) VALUES (?, ?, 'running', ?)`,
		reportID, projectID, resolved,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func resolveCouncilModel(ctx context.Context, repo *repository.Repository) (string, error) {
	raw, ok, err := repo.GetSetting(ctx, "model.advisor_council")
	if err != nil {
		return "", err
	}
	if ok && raw != "" {
		var obj map[string]string
		if jsonErr := json.Unmarshal([]byte(raw), &obj); jsonErr == nil {
			if modelID, exists := obj["modelId"]; exists && ai.GetModel(modelID) != nil {
				return modelID, nil
			}
		}
	}
	task := ai.GetTask("advisor_council")
	if task == nil {
		return "claude-cli:opus", nil
	}
	return task.Default, nil
}

// RunCouncil runs the full 5-advisor + synthesis loop and updates report_councils on completion or failure.
func RunCouncil(ctx context.Context, db *sql.DB, aiSvc *ai.Service, projectID string, reportID int64, councilID int64) error {
	rep, err := getReportForCouncil(ctx, db, reportID)
	if err != nil {
		_ = markCouncilFailed(db, councilID, err.Error())
		return err
	}

	// Get current model_used from the row.
	var modelUsed string
	_ = db.QueryRowContext(ctx, "SELECT model_used FROM report_councils WHERE id = ?", councilID).Scan(&modelUsed)

	var advisorOutputs []json.RawMessage
	for _, adv := range advisors {
		sysPrompt := strings.ReplaceAll(advisorSystemPromptTemplate, "[ADVISOR_ROLE]", adv.Role)
		userMsg := buildAdvisorUserMessage(adv, rep)
		raw, _, genErr := aiSvc.GenerateForTask(ctx, "advisor_council", sysPrompt, userMsg)
		if genErr != nil {
			_ = markCouncilFailed(db, councilID, genErr.Error())
			return genErr
		}
		parsed, parseErr := parseCouncilJSON(raw)
		if parseErr != nil {
			_ = markCouncilFailed(db, councilID, parseErr.Error())
			return parseErr
		}
		advisorOutputs = append(advisorOutputs, parsed)
	}

	synthMsg := buildSynthesisUserMessage(rep, advisorOutputs)
	rawSynth, _, synthErr := aiSvc.GenerateForTask(ctx, "advisor_council", synthesisSystemPrompt, synthMsg)
	if synthErr != nil {
		_ = markCouncilFailed(db, councilID, synthErr.Error())
		return synthErr
	}
	synthesis, parseErr := parseCouncilJSON(rawSynth)
	if parseErr != nil {
		_ = markCouncilFailed(db, councilID, parseErr.Error())
		return parseErr
	}

	councilData := map[string]any{
		"advisors":  advisorOutputs,
		"synthesis": synthesis,
	}
	councilJSON, _ := json.Marshal(councilData)

	_, err = db.ExecContext(ctx, `
		UPDATE report_councils
		SET status = 'completed', council_json = ?, model_used = ?, completed_at = datetime('now')
		WHERE id = ?
	`, string(councilJSON), modelUsed, councilID)
	return err
}

func markCouncilFailed(db *sql.DB, councilID int64, message string) error {
	_, err := db.ExecContext(context.Background(),
		`UPDATE report_councils SET status = 'failed', error = ? WHERE id = ?`,
		message, councilID,
	)
	return err
}

// ReconcileCouncils ports Express reconcileCouncils(db) startup behavior.
// It marks stale running councils as failed and starts background runs for completed
// reports that have no council row.
func ReconcileCouncils(ctx context.Context, db *sql.DB, aiSvc *ai.Service, repo *repository.Repository) {
	// Mark stale running councils as failed (older than 30 minutes).
	_, _ = db.ExecContext(ctx, `
		UPDATE report_councils
		SET status = 'failed', error = 'Council run timed out (stale)'
		WHERE status = 'running' AND created_at < datetime('now', '-30 minutes')
	`)

	// Find completed reports with no council row.
	rows, err := db.QueryContext(ctx, `
		SELECT rr.id, rr.project_id
		FROM research_reports rr
		LEFT JOIN report_councils rc ON rc.report_id = rr.id
		WHERE rr.status = 'completed' AND rc.id IS NULL
	`)
	if err != nil {
		fmt.Printf("[council] reconcile query failed: %v\n", err)
		return
	}
	defer rows.Close()

	type orphan struct {
		reportID  int64
		projectID string
	}
	var orphans []orphan
	for rows.Next() {
		var o orphan
		if err := rows.Scan(&o.reportID, &o.projectID); err == nil {
			orphans = append(orphans, o)
		}
	}

	for _, o := range orphans {
		councilID, err := StartCouncil(ctx, db, repo, o.projectID, o.reportID)
		if err != nil {
			fmt.Printf("[council] reconcile start failed for report %d: %v\n", o.reportID, err)
			continue
		}
		go func(rID, cID int64, pID string) {
			if aiSvc == nil {
				return
			}
			bgCtx := context.Background()
			if err := RunCouncil(bgCtx, db, aiSvc, pID, rID, cID); err != nil {
				fmt.Printf("[council] reconcile background run failed for report %d: %v\n", rID, err)
			}
		}(o.reportID, councilID, o.projectID)
	}
}
