<script>
  import { google as googleApi } from '../lib/api.js';
  import { formatDate } from '../lib/format.js';

  let { projectId, reportId, onBack } = $props();

  const modes = ['seo', 'messaging', 'competitor', 'outreach'];

  let report = $state(null);
  let keywordSummaries = $state([]);
  let selectedMode = $state('seo');
  let ranked = $state([]);
  let loading = $state(true);
  let loadingRanked = $state(false);
  let pageError = $state(null);
  let rankedError = $state(null);

  async function loadReportData() {
    loading = true;
    pageError = null;
    rankedError = null;
    try {
      const [reportData, summaryData] = await Promise.all([
        googleApi.report(projectId, reportId),
        googleApi.keywordSummaries(projectId, reportId),
      ]);
      report = reportData;
      keywordSummaries = summaryData;
      ranked = [];
      await loadRanked(selectedMode);
    } catch (e) {
      pageError = e.message || 'Failed to load Google report';
    } finally {
      loading = false;
    }
  }

  async function loadRanked(mode) {
    loadingRanked = true;
    try {
      selectedMode = mode;
      rankedError = null;
      const data = await googleApi.rankedResults(projectId, reportId, { mode });
      ranked = data.results || [];
    } catch (e) {
      ranked = [];
      rankedError = e.message || 'Failed to load ranked results';
    } finally {
      loadingRanked = false;
    }
  }

  function goBack() {
    onBack?.();
  }

  let lastLoadKey = $state(null);
  $effect(() => {
    const key = projectId && reportId ? `${projectId}:${reportId}` : null;
    if (key && key !== lastLoadKey) {
      lastLoadKey = key;
      loadReportData();
    }
  });
</script>

<div class="report-view">
  <button class="back-btn" onclick={goBack}>&larr; Back to Google Reports</button>

  {#if loading}
    <div class="loading">Loading report...</div>
  {:else if pageError}
    <div class="error-msg">{pageError}</div>
  {:else if report}
    <div class="report-header">
      <h2>Google Report #{report.id}</h2>
      <div class="report-meta">
        <span>Run #{report.run_id}</span>
        <span>{formatDate(report.created_at)}</span>
      </div>
    </div>

    <div class="summary-grid">
      <div class="summary-card">
        <h3>Top Insights</h3>
        <ul>
          {#each report.executive_summary?.top_insights || [] as insight}
            <li>{insight}</li>
          {/each}
        </ul>
      </div>
      <div class="summary-card">
        <h3>Opportunities</h3>
        {#if (report.opportunities || []).length === 0}
          <div class="empty-hint">No opportunities identified.</div>
        {:else}
          <ul class="struct-list">
            {#each report.opportunities as item}
              <li class="struct-item">
                {#if typeof item === 'string'}
                  {item}
                {:else if item.kind === 'content_gap'}
                  <span class="tag tag-gap">Content gap</span>
                  {#if item.url}
                    <a href={item.url} target="_blank" rel="noopener noreferrer">{item.label}</a>
                  {:else}
                    <span>{item.label}</span>
                  {/if}
                {:else if item.kind === 'top_source'}
                  <span class="tag tag-src">Top Source</span>
                  {#if item.poster}
                    <span class="mono">{item.poster}</span>
                    <span class="muted">{item.domain}{item.count != null ? ` (${item.count} results)` : ''}</span>
                  {:else}
                    <span class="mono">{item.label}</span>
                    {#if item.count != null}<span class="muted">({item.count} results)</span>{/if}
                  {/if}
                {:else if item.kind === 'competitor'}
                  <span class="tag tag-comp">Competitor</span>
                  <span class="mono">{item.label}</span>
                  {#if item.count != null}<span class="muted">({item.count} results)</span>{/if}
                {:else}
                  <span>{item.label || JSON.stringify(item)}</span>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      </div>
      <div class="summary-card">
        <h3>Risks</h3>
        {#if (report.risks || []).length === 0}
          <div class="empty-hint">No risks flagged.</div>
        {:else}
          <ul class="struct-list">
            {#each report.risks as item}
              <li class="struct-item">
                {#if typeof item === 'string'}
                  {item}
                {:else}
                  <span class="tag tag-risk tag-risk-{item.level || 'low'}">{item.level || 'low'}</span>
                  <strong>{item.label}</strong>
                  {#if item.detail}<div class="detail">{item.detail}</div>{/if}
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      </div>
      <div class="summary-card">
        <h3>Next Actions</h3>
        {#if (report.next_actions || []).length === 0}
          <div class="empty-hint">No next actions suggested.</div>
        {:else}
          <ul class="struct-list">
            {#each report.next_actions as item}
              <li class="struct-item">
                {#if typeof item === 'string'}
                  {item}
                {:else}
                  <strong>{item.action}</strong>
                  {#if item.url}
                    <a href={item.url} target="_blank" rel="noopener noreferrer">{item.target}</a>
                  {:else}
                    <span>{item.target}</span>
                  {/if}
                  {#if item.reason}<div class="detail">{item.reason}</div>{/if}
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      </div>
      <div class="summary-card">
        <h3>Recurring Pain Themes</h3>
        {#if (report.executive_summary?.strongest_recurring_pain_themes || []).length === 0}
          <div class="empty-hint">No recurring pain terms detected in summaries.</div>
        {:else}
          <ul class="chip-list">
            {#each report.executive_summary.strongest_recurring_pain_themes as theme}
              <li class="chip" class:chip-problem={theme.category === 'problem'} class:chip-workflow={theme.category === 'workflow'}>
                <span class="chip-value">{theme.value}</span>
                <span class="chip-count">{theme.count}</span>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
      <div class="summary-card">
        <h3>Messaging Phrases</h3>
        {#if (report.executive_summary?.interesting_phrases_for_messaging || []).length === 0}
          <div class="empty-hint">No phrases above frequency threshold.</div>
        {:else}
          <ul class="chip-list">
            {#each report.executive_summary.interesting_phrases_for_messaging as phrase}
              <li class="chip">
                <span class="chip-value">{phrase.phrase}</span>
                <span class="chip-count">{phrase.count}</span>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
      <div class="summary-card">
        <h3>Mentioned Products</h3>
        {#if (report.executive_summary?.mentioned_products || []).length === 0}
          <div class="empty-hint">No competing products identified in results.</div>
        {:else}
          <ul class="chip-list">
            {#each report.executive_summary.mentioned_products as product}
              <li class="chip chip-product">
                <span class="chip-value">{product.name}</span>
                <span class="chip-count">{product.count}</span>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    </div>

    <div class="keywords-section">
      <h3>Keyword Summaries ({keywordSummaries.length})</h3>
      <div class="keyword-list">
        {#each keywordSummaries as summary}
          <div class="keyword-card">
            <div class="keyword-title">{summary.root_keyword}</div>
            <div class="keyword-metrics">
              <span>{summary.total_results} total</span>
              <span>{summary.relevant_results} relevant</span>
              <span>{summary.outreach_candidates} outreach</span>
            </div>
          </div>
        {/each}
      </div>
    </div>

    <div class="results-section">
      <div class="results-header">
        <h3>Ranked Results</h3>
        <div class="mode-tabs">
          {#each modes as mode}
            <button class="mode-btn" class:active={selectedMode === mode} onclick={() => loadRanked(mode)}>
              {mode}
            </button>
          {/each}
        </div>
      </div>

      {#if loadingRanked}
        <div class="loading">Loading ranked results...</div>
      {:else}
        {#if rankedError}
          <div class="error-msg">{rankedError}</div>
        {/if}
        {#if ranked.length === 0}
          <div class="loading">No ranked results available for this mode.</div>
        {:else}
          <div class="result-list">
            {#each ranked as item}
              <a class="result-card" href={item.url} target="_blank" rel="noopener noreferrer">
                <div class="result-title">{item.title || item.url}</div>
                <div class="result-meta">
                  <span>{item.domain}</span>
                  <span>Relevance {item.relevance_score ?? '-'}</span>
                  <span>Confidence {item.confidence_score ?? '-'}</span>
                </div>
                {#if item.summary}
                  <p class="result-summary">{item.summary}</p>
                {/if}
              </a>
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  {/if}
</div>

<style>
  .report-view {
    max-width: 1000px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 20px;
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
  }

  .back-btn:hover {
    color: #e2e2e8;
    border-color: #3a3a55;
  }

  .loading {
    padding: 24px;
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

  .summary-grid {
    display: grid;
    gap: 12px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .summary-card {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 14px;
  }

  .summary-card h3 {
    font-size: 13px;
    margin-bottom: 8px;
    color: #e2e2e8;
  }

  .summary-card ul {
    margin: 0;
    padding-left: 18px;
    color: #c0c0d0;
    font-size: 13px;
    line-height: 1.5;
  }

  .chip-list {
    list-style: none;
    padding-left: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 3px 8px;
    background: #23233a;
    border: 1px solid #2a2a3a;
    border-radius: 999px;
    font-size: 12px;
  }

  .chip-value {
    color: #e2e2e8;
  }

  .chip-count {
    color: #888;
    font-variant-numeric: tabular-nums;
  }

  .chip-problem {
    border-color: #6a2a2a;
    background: #2a1a1a;
  }

  .chip-problem .chip-value {
    color: #f87171;
  }

  .chip-workflow {
    border-color: #2a4a6a;
    background: #152030;
  }

  .chip-workflow .chip-value {
    color: #7fb6ff;
  }

  .chip-product {
    border-color: #4a3a1a;
    background: #2a1f0a;
  }

  .chip-product .chip-value {
    color: #fbbf24;
  }

  .empty-hint {
    font-size: 12px;
    color: #666;
  }

  .struct-list {
    list-style: none;
    padding-left: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .struct-item {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 6px;
    font-size: 13px;
    color: #c0c0d0;
    line-height: 1.5;
  }

  .struct-item a {
    color: #7fb6ff;
    text-decoration: none;
  }

  .struct-item a:hover {
    text-decoration: underline;
  }

  .struct-item .detail {
    flex-basis: 100%;
    color: #888;
    font-size: 12px;
  }

  .tag {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    padding: 2px 6px;
    border-radius: 4px;
    background: #23233a;
    color: #c0c0d0;
  }

  .tag-gap {
    background: #152030;
    color: #7fb6ff;
  }

  .tag-comp {
    background: #2a1a2a;
    color: #f0a0f0;
  }

  .tag-src {
    background: #1a2a1a;
    color: #86efac;
  }

  .tag-risk-high {
    background: #3a1a1a;
    color: #f87171;
  }

  .tag-risk-medium {
    background: #3a2a1a;
    color: #fbbf24;
  }

  .tag-risk-low {
    background: #1a2a1a;
    color: #86efac;
  }

  .mono {
    font-family: ui-monospace, SFMono-Regular, monospace;
  }

  .muted {
    color: #666;
    font-size: 12px;
  }

  .keywords-section h3,
  .results-section h3 {
    font-size: 16px;
    color: #e2e2e8;
    margin-bottom: 10px;
  }

  .keyword-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .keyword-card {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 12px;
  }

  .keyword-title {
    font-size: 14px;
    color: #e2e2e8;
    margin-bottom: 6px;
  }

  .keyword-metrics {
    display: flex;
    gap: 12px;
    font-size: 12px;
    color: #9090b0;
  }

  .results-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 10px;
  }

  .mode-tabs {
    display: flex;
    gap: 6px;
  }

  .mode-btn {
    padding: 6px 10px;
    background: #23233a;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #888;
    font-size: 12px;
    text-transform: uppercase;
    cursor: pointer;
  }

  .mode-btn.active {
    color: #e2e2e8;
    border-color: #7c6af5;
  }

  .result-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .result-card {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    background: #1a1a24;
    text-decoration: none;
  }

  .result-card:hover {
    border-color: #7c6af5;
  }

  .result-title {
    color: #e2e2e8;
    font-size: 14px;
    font-weight: 500;
  }

  .result-meta {
    display: flex;
    gap: 12px;
    font-size: 12px;
    color: #888;
  }

  .result-summary {
    color: #c0c0d0;
    font-size: 13px;
    line-height: 1.5;
  }
</style>
