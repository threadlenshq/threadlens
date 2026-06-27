<script>
  import { onDestroy } from 'svelte';
  import { reports as reportsApi, posts as postsApi } from '../lib/api.js';
  import { formatDate } from '../lib/format.js';
  import { renderAssessment } from '../lib/assessment.js';
  import ClusterCard from './ClusterCard.svelte';
  import AdvisorCouncilView from './AdvisorCouncilView.svelte';

  let { projectId, reportId, onBack, onSelectAngle } = $props();

  let report = $state(null);
  let reportPosts = $state([]);
  let loading = $state(true);
  let error = $state(null);
  let pollTimer = $state(null);
  let destroyed = $state(false);
  let councilData = $state(null);
  let councilLoading = $state(false);

  async function loadReferencedPosts(completedReport) {
    const allPosts = await postsApi.list(projectId, { status: 'all' });
    if (destroyed) return;
    const referencedIds = new Set(
      (completedReport.clusters || []).flatMap(c => c.post_ids || [])
    );
    reportPosts = allPosts.filter(p => referencedIds.has(p.id));
  }

  async function loadCouncil() {
    councilLoading = true;
    try {
      councilData = await reportsApi.council(projectId, reportId);
    } catch {
      councilData = null;
    } finally {
      councilLoading = false;
    }
  }

  async function loadReport() {
    loading = true;
    error = null;
    try {
      report = await reportsApi.get(projectId, reportId);
      if (report.status === 'running') {
        schedulePoll();
      } else {
        stopPolling();
        loadCouncil();
        await loadReferencedPosts(report);
      }
    } catch (e) {
      error = e.message || 'Failed to load report';
    } finally {
      loading = false;
    }
  }

  function schedulePoll() {
    if (pollTimer || destroyed) return;
    pollTimer = setTimeout(async () => {
      pollTimer = null;
      try {
        const updated = await reportsApi.get(projectId, reportId);
        if (destroyed) return;
        if (updated.status !== 'running') {
          report = updated;
          await loadReferencedPosts(updated);
        } else {
          schedulePoll();
        }
      } catch {
        if (!destroyed) schedulePoll();
      }
    }, 3000);
  }

  function stopPolling() {
    if (pollTimer) {
      clearTimeout(pollTimer);
      pollTimer = null;
    }
  }

  function goBack() {
    onBack?.();
  }

  function handleSelectAngle(e) {
    onSelectAngle?.({
      reportId: report.id,
      clusterIndex: e.clusterIndex,
      cluster: report.clusters[e.clusterIndex],
    });
  }

  let lastLoadedKey = $state(null);
  $effect(() => {
    const key = projectId && reportId ? `${projectId}:${reportId}` : null;
    if (key && key !== lastLoadedKey) {
      lastLoadedKey = key;
      loadReport();
    }
  });

  onDestroy(() => { destroyed = true; stopPolling(); });
</script>

<div class="report-view">
  <button class="back-btn" onclick={goBack}>&larr; Back to Reports</button>

  {#if loading}
    <div class="loading">Loading report...</div>
  {:else if error}
    <div class="error-msg">{error}</div>
  {:else if report && report.status === 'running'}
    <div class="report-header">
      <h2>Research Report <a class="doc-link" href="https://docs.threadlens.dev/user-guide/reports/" target="_blank" rel="noopener" title="How reports work">?</a></h2>
      <div class="report-meta">
        <span>Model: {report.model_used}</span>
      </div>
    </div>
    <div class="running-indicator">
      <div class="spinner"></div>
      <p>Analysis in progress - this may take a few minutes...</p>
    </div>
  {:else if report}
    <div class="report-header">
      <h2>{report.title || 'Research Report'} <a class="doc-link" href="https://docs.threadlens.dev/user-guide/reports/" target="_blank" rel="noopener" title="How reports work">?</a></h2>
      <div class="report-meta">
        <span>{report.post_count} posts analyzed</span>
        <span>Model: {report.model_used}</span>
        <span>{formatDate(report.completed_at)}</span>
      </div>
    </div>

    {#if report.status === 'failed'}
      <div class="error-msg">{report.error || 'Analysis failed'}</div>
    {/if}

    {#if report.assessment}
      <div class="assessment">
        <h3 class="section-label">Overall Assessment <a class="doc-link" href="https://docs.threadlens.dev/user-guide/reports/#research-reports" target="_blank" rel="noopener" title="AI-generated summary of findings">?</a></h3>
        <div class="assessment-text">{@html renderAssessment(report.assessment)}</div>
      </div>
    {/if}

    {#if councilLoading}
      <p class="council-loading">Loading council...</p>
    {:else if councilData}
      <AdvisorCouncilView council={councilData} />
    {/if}

    <div class="clusters-section">
      <h3 class="section-label">Pain Point Clusters ({(report.clusters || []).length}) <a class="doc-link" href="https://docs.threadlens.dev/user-guide/reports/#research-reports" target="_blank" rel="noopener" title="AI-identified groups of related pain points">?</a></h3>
      {#each report.clusters || [] as cluster, i (i)}
        <ClusterCard
          {cluster}
          index={i}
          posts={reportPosts}
          onSelectAngle={handleSelectAngle}
        />
      {/each}
    </div>
  {/if}
</div>

<style>
  .report-view {
    max-width: 900px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .back-btn {
    align-self: flex-start;
    padding: 6px 12px;
    background: none;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #888;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .back-btn:hover {
    color: #e2e2e8;
    border-color: #3a3a55;
  }

  .loading {
    padding: 40px;
    text-align: center;
    color: #666;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .report-header h2 {
    font-size: 22px;
    font-weight: 700;
    color: #e2e2e8;
    margin-bottom: 8px;
  }

  .report-meta {
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: #666;
  }

  .assessment {
    padding: 20px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
  }

  .section-label {
    font-size: 11px;
    font-weight: 600;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 12px;
  }

  .assessment-text {
    font-size: 14px;
    color: #c0c0d0;
    line-height: 1.7;
  }

  .assessment-text :global(p) {
    margin: 0 0 12px 0;
  }

  .assessment-text :global(p:last-child) {
    margin-bottom: 0;
  }

  .assessment-text :global(strong) {
    color: #e2e2e8;
    font-weight: 600;
  }

  .assessment-text :global(ol) {
    margin: 12px 0;
    padding-left: 20px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .assessment-text :global(li) {
    padding-left: 4px;
  }

  .assessment-text :global(ol:last-child) {
    margin-bottom: 0;
  }

  .clusters-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .council-loading {
    font-size: 13px;
    color: #666;
    padding: 8px 0;
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
