package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kyle/scout/open-core/apps/api/internal/domain"
	"github.com/kyle/scout/open-core/apps/api/internal/querynorm"
)

// dedupeQueriesByTarget removes queries that resolve to the same effective fetch
// target for the platform, keeping the first occurrence.
func dedupeQueriesByTarget(platform string, queries []domain.Query) []domain.Query {
	seen := make(map[string]bool, len(queries))
	out := make([]domain.Query, 0, len(queries))
	for _, q := range queries {
		key := querynorm.Key(platform, q.QueryURL)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, q)
	}
	return out
}

// runAll merges the given social platforms into a single run: it fetches each
// platform's deduped query pool under runID, aggregates counts, and completes the
// run once. A single platform's non-cancellation failure is recorded as a warning
// and the run continues with the remaining platforms.
func (r *Runner) runAll(ctx context.Context, projectID string, runID int64, platforms []string) (Result, error) {
	project, err := r.Repo.GetProject(ctx, projectID)
	if err != nil {
		return Result{RunID: runID}, fmt.Errorf("project not found: %s", projectID)
	}

	var totalChecked, totalFound int64
	var warnings []string

	for _, platform := range platforms {
		queries, err := r.Repo.EnabledQueries(ctx, projectID, platform)
		if err != nil {
			return Result{RunID: runID}, err
		}
		if len(queries) == 0 {
			continue
		}
		queries = dedupeQueriesByTarget(platform, queries)

		checked, found, w, perr := r.processSocialPlatform(ctx, project, projectID, platform, queries, runID)
		totalChecked += checked
		totalFound += found
		if perr != nil {
			// A true cancellation/timeout aborts the whole run; any other
			// per-platform error is a partial failure - record it and continue.
			if errors.Is(perr, context.Canceled) || errors.Is(perr, context.DeadlineExceeded) {
				r.failRun(runID, ctxErrMessage(ctx))
				return Result{RunID: runID, PostsChecked: totalChecked, PostsFound: totalFound}, nil
			}
			warnings = append(warnings, fmt.Sprintf("%s: %s", platform, perr.Error()))
			continue
		}
		for _, line := range w {
			warnings = append(warnings, platform+": "+line)
		}
	}

	var warningsText *string
	if len(warnings) > 0 {
		s := strings.Join(warnings, "\n")
		warningsText = &s
	}
	if err := r.Repo.CompleteScoutRun(ctx, runID, totalChecked, totalFound, warningsText); err != nil {
		return Result{RunID: runID}, err
	}
	return Result{RunID: runID, PostsChecked: totalChecked, PostsFound: totalFound}, nil
}

// runAllAndReport runs the merged social all-run and, on success, invokes the
// ReportTrigger hook when a report was requested.
func (r *Runner) runAllAndReport(ctx context.Context, projectID string, runID int64, platforms []string, generateReport bool) (Result, error) {
	res, err := r.runAll(ctx, projectID, runID, platforms)
	if err != nil {
		return res, err
	}
	if generateReport && r.ReportTrigger != nil {
		r.ReportTrigger(projectID)
	}
	return res, nil
}

// StartAllAsync runs the merged all-run in a background goroutine, mirroring
// StartAsync's context registration and failure handling.
func (r *Runner) StartAllAsync(projectID string, runID int64, platforms []string, generateReport bool) {
	bgCtx, cancel := context.WithTimeout(context.Background(), pipelineTimeout)

	r.mu.Lock()
	r.runs[runID] = cancel
	r.mu.Unlock()

	go func() {
		defer func() {
			cancel()
			r.mu.Lock()
			delete(r.runs, runID)
			r.mu.Unlock()
		}()
		if _, err := r.runAllAndReport(bgCtx, projectID, runID, platforms, generateReport); err != nil {
			r.failRun(runID, err.Error())
		}
	}()
}
