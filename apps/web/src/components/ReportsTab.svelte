<script>
  import { onDestroy } from 'svelte';
  import { reports as reportsApi } from '../lib/api.js';
  import { formatDate } from '../lib/format.js';
  import Surface from './ui/Surface.svelte';
  import EmptyState from './ui/EmptyState.svelte';
  import LoadingSkeleton from './ui/LoadingSkeleton.svelte';

  let { projectId, onViewReport } = $props();

  let reportsList = $state([]);
  let loading = $state(false);
  let running = $state(false);
  let error = $state(null);
  let pollTimer = $state(null);

  let minScore = $state('');
  let showOptions = $state(false);

  let lastLoadedProjectId = $state(null);
  $effect(() => {
    if (projectId && projectId !== lastLoadedProjectId) {
      lastLoadedProjectId = projectId;
      loadReports();
    }
  });

  async function loadReports() {
    loading = true;
    try {
      reportsList = await reportsApi.list(projectId);
      updatePolling();
    } catch (e) {
      console.error('Failed to load reports:', e);
    } finally {
      loading = false;
    }
  }

  function needsPolling(list) {
    return list.some(r => r.status === 'running' || r.council_status === 'running');
  }

  function updatePolling() {
    const active = needsPolling(reportsList);
    if (active && !pollTimer) {
      schedulePoll();
    } else if (!active && pollTimer) {
      stopPolling();
    }
  }

  function schedulePoll() {
    if (pollTimer) return;
    pollTimer = setTimeout(async () => {
      pollTimer = null;
      try {
        const updated = await reportsApi.list(projectId);
        const sig = updated.map(r => `${r.id}:${r.status}:${r.council_status || ''}`).join(',');
        const oldSig = reportsList.map(r => `${r.id}:${r.status}:${r.council_status || ''}`).join(',');
        if (sig !== oldSig) reportsList = updated;
        if (needsPolling(updated)) {
          schedulePoll();
        }
      } catch {
        schedulePoll();
      }
    }, 3000);
  }

  function stopPolling() {
    if (pollTimer) {
      clearTimeout(pollTimer);
      pollTimer = null;
    }
  }

  async function runAnalysis() {
    running = true;
    error = null;
    try {
      const options = {};
      if (minScore) options.min_score = parseFloat(minScore);
      await reportsApi.create(projectId, options);
      await loadReports();
    } catch (e) {
      error = e.message || 'Failed to run analysis';
    } finally {
      running = false;
    }
  }

  function viewReport(reportId) {
    onViewReport?.({ reportId });
  }

  function parseClusterNames(clusters) {
    return (clusters || []).slice(0, 3).map(c => c.name);
  }

  onDestroy(stopPolling);
</script>

<div class="reports-tab">
  <div class="reports-header">
    <h2>Research Reports</h2>
    <div class="header-actions">
      <button
        class="options-toggle"
        onclick={() => showOptions = !showOptions}
      >
        Options {showOptions ? '▴' : '▾'}
      </button>
      <button
        class="run-btn"
        onclick={runAnalysis}
        disabled={running}
      >
        {running ? 'Analyzing...' : 'Run Analysis'}
      </button>
    </div>
  </div>

  {#if showOptions}
    <div class="options-bar">
      <label class="option-field">
        <span class="option-label">Min Score</span>
        <input type="number" bind:value={minScore} placeholder="0" min="0" max="10" step="0.5" />
      </label>
    </div>
  {/if}

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#if running}
    <div class="running-indicator">
      <div class="spinner"></div>
      <p>Running analysis with Claude Opus - this may take a few minutes...</p>
    </div>
  {/if}

  {#if loading}
    <div class="max-w-4xl mx-auto p-6">
      <LoadingSkeleton type="card" count={3} />
    </div>
  {:else if reportsList.length === 0}
    <div class="max-w-4xl mx-auto p-6 mt-12">
      <EmptyState
        title="No Reports Yet"
        description="Select posts from your Inbox and click 'Create Report' to generate an AI analysis of pain points and product angles."
        icon="✨"
      />
    </div>
  {:else}
    <div class="reports-list">
      {#each reportsList as report (report.id)}
        <button
          class="report-card"
          class:failed={report.status === 'failed'}
          onclick={() => viewReport(report.id)}
        >
          <div class="report-card-top">
            <span class="report-title">{report.title || 'Untitled Report'}</span>
            <span class="report-status" class:completed={report.status === 'completed'} class:failed={report.status === 'failed'} class:running={report.status === 'running'}>
              {report.status === 'running' ? 'analyzing...' : report.status}
            </span>
          </div>
          <div class="report-meta">
            <span>{report.post_count} posts analyzed</span>
            <span>{formatDate(report.created_at)}</span>
          </div>
          {#if report.status === 'completed'}
            <div class="report-clusters">
              {#each parseClusterNames(report.clusters) as name}
                <span class="cluster-tag">{name}</span>
              {/each}
              {#if report.council_status}
                <span class="council-badge council-badge--{report.council_status}">
                  Council: {report.council_status === 'completed' ? 'ready' : report.council_status}
                </span>
              {/if}
            </div>
          {/if}
          {#if report.status === 'failed' && report.error}
            <p class="report-error">{report.error}</p>
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

  .reports-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .reports-header h2 {
    font-size: 20px;
    font-weight: 700;
    color: #e2e2e8;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .options-toggle {
    padding: 8px 14px;
    background: #23233a;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #888;
    font-size: 13px;
    cursor: pointer;
  }

  .options-toggle:hover {
    color: #e2e2e8;
    border-color: #3a3a55;
  }

  .run-btn {
    padding: 9px 20px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .run-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .run-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .options-bar {
    display: flex;
    gap: 16px;
    padding: 12px 16px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
  }

  .option-field {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: #888;
  }

  .option-field input {
    width: 70px;
    padding: 6px 8px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 4px;
    color: #e2e2e8;
    font-size: 13px;
  }

  .option-field input:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .running-indicator {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px;
    background: #1a1a30;
    border: 1px solid #3a3a60;
    border-radius: 8px;
    color: #9090b0;
    font-size: 14px;
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid #3a3a60;
    border-top-color: #7c6af5;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    flex-shrink: 0;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
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

  .report-card.failed {
    opacity: 0.6;
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

  .report-status {
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 3px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    background: #2a2a3a;
    color: #888;
  }

  .report-status.completed {
    background: #98c37920;
    color: #98c379;
  }

  .report-status.failed {
    background: #e06c7520;
    color: #e06c75;
  }

  .report-status.running {
    background: #7c6af520;
    color: #7c6af5;
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
  }

  .report-meta {
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: #666;
  }

  .report-clusters {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }

  .cluster-tag {
    font-size: 11px;
    padding: 2px 8px;
    background: #23233a;
    border: 1px solid #2a2a3a;
    border-radius: 4px;
    color: #9090b0;
  }

  .report-error {
    font-size: 12px;
    color: #e06c75;
  }

  .council-badge {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 4px;
    color: #888;
    background: #23233a;
    border: 1px solid #2a2a3a;
  }

  .council-badge--running {
    background: #7c6af520;
    border-color: #7c6af540;
    color: #7c6af5;
  }

  .council-badge--completed {
    background: #98c37920;
    border-color: #98c37940;
    color: #98c379;
  }

  .council-badge--failed {
    background: #e06c7520;
    border-color: #e06c7540;
    color: #e06c75;
  }
</style>
