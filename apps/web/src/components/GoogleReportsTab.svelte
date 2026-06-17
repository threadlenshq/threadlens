<script>
  import { onDestroy, onMount } from 'svelte';
  import { google as googleApi } from '../lib/api.js';
  import { formatDate } from '../lib/format.js';

  let { projectId, onViewReport, googleLocked = false } = $props();

  let reportsList = $state([]);
  let loading = $state(false);
  let error = $state(null);
  let pollTimer = $state(null);

  async function loadReports() {
    loading = true;
    error = null;
    try {
      reportsList = await googleApi.reports(projectId);
      updatePolling();
    } catch (e) {
      error = e.message || 'Failed to load Google reports';
    } finally {
      loading = false;
    }
  }

  function updatePolling() {
    const hasRunning = reportsList.some((report) => report.run?.status === 'running');
    if (hasRunning && !pollTimer) schedulePoll();
    if (!hasRunning && pollTimer) stopPolling();
  }

  function schedulePoll() {
    if (pollTimer) return;
    pollTimer = setTimeout(async () => {
      pollTimer = null;
      try {
        const updated = await googleApi.reports(projectId);
        const sig = updated.map((r) => `${r.id}:${r.run?.status || ''}`).join(',');
        const prevSig = reportsList.map((r) => `${r.id}:${r.run?.status || ''}`).join(',');
        if (sig !== prevSig) reportsList = updated;
        updatePolling();
      } catch {
        schedulePoll();
      }
    }, 3000);
  }

  function stopPolling() {
    if (!pollTimer) return;
    clearTimeout(pollTimer);
    pollTimer = null;
  }

  function openReport(reportId) {
    onViewReport?.({ reportId });
  }

  onMount(loadReports);
  onDestroy(stopPolling);
</script>

<div class="reports-tab">
  <div class="reports-header">
    <h2>Google Reports <a class="doc-link" href="https://docs.threadlens.dev/user-guide/reports/#google-reports" target="_blank" rel="noopener" title="How Google reports work">?</a></h2>
  </div>

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#if loading}
    <div class="loading">Loading reports...</div>
  {:else if reportsList.length === 0}
    <div class="empty-state">
      <p class="empty-title">No Google reports yet</p>
      {#if googleLocked}
        <p class="empty-desc">Existing Google reports would appear here. New Google scouting requires <code>PARALLEL_API_KEY</code> in the Scout API environment.</p>
      {:else}
        <p class="empty-desc">Add and enable Google queries, then run ThreadLens with Google to generate your first report.</p>
      {/if}
    </div>
  {:else}
    <div class="reports-list">
      {#each reportsList as report (report.id)}
        <button class="report-card" onclick={() => openReport(report.id)}>
          <div class="report-card-top">
            <span class="report-title">Google Report #{report.id}</span>
            <span class="report-status" class:running={report.run?.status === 'running'}>
              {report.run?.status || 'ready'}
            </span>
          </div>
          <div class="report-meta">
            <span>Run #{report.run_id}</span>
            <span>{formatDate(report.created_at)}</span>
          </div>
          {#if report.executive_summary?.top_insights?.length}
            <div class="insight-list">
              {#each report.executive_summary.top_insights.slice(0, 2) as insight}
                <span class="insight-tag">{insight}</span>
              {/each}
            </div>
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .reports-tab {
    max-width: 900px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .reports-header h2 {
    font-size: 20px;
    font-weight: 700;
    color: #e2e2e8;
  }

  .loading {
    padding: 40px;
    text-align: center;
    color: #666;
  }

  .empty-state {
    padding: 60px 20px;
    text-align: center;
  }

  .empty-title {
    font-size: 18px;
    color: #e2e2e8;
    margin-bottom: 8px;
  }

  .empty-desc {
    font-size: 14px;
    color: #666;
  }

  .empty-desc code {
    font-family: monospace;
    font-size: 13px;
    background: #1e1e2e;
    border: 1px solid #2a2a3a;
    border-radius: 3px;
    padding: 1px 5px;
    color: #a0a0c8;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .reports-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .report-card {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 16px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    cursor: pointer;
    text-align: left;
    transition: border-color 0.15s;
    width: 100%;
  }

  .report-card:hover {
    border-color: #7c6af5;
  }

  .report-card-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .report-title {
    font-size: 15px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .report-meta {
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: #666;
  }

  .report-status {
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 3px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    background: #98c37920;
    color: #98c379;
  }

  .report-status.running {
    background: #7c6af520;
    color: #7c6af5;
  }

  .insight-list {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }

  .insight-tag {
    font-size: 11px;
    padding: 2px 8px;
    background: #23233a;
    border: 1px solid #2a2a3a;
    border-radius: 4px;
    color: #9090b0;
  }

  .doc-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    font-size: 9px;
    font-weight: 700;
    color: #4a4a60;
    background: #2a2a3a;
    border-radius: 50%;
    text-decoration: none;
    margin-left: 5px;
    vertical-align: middle;
    transition: color 0.15s, background 0.15s;
    cursor: help;
  }
  .doc-link:hover {
    color: #61afef;
    background: #61afef20;
  }
</style>
