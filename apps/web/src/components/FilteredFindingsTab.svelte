<script>
  import { filters as filtersApi } from '../lib/api.js';
  import {
    FILTERED_FINDINGS_LABEL,
    RECHECK_SELECTED_LABEL,
    FILTERED_EMPTY_TITLE,
    FILTERED_EMPTY_BODY,
    reasonLabel,
  } from '../lib/filterLabels.js';
  import FilterRecoveryModal from './FilterRecoveryModal.svelte';

  /** @type {string} */
  let { projectId, api = filtersApi, onJobCreated, onRestored } = $props();

  // Filters
  let platformFilter = $state('');
  let reasonFilter = $state('');

  // Findings data
  let findings = $state([]);
  let loading = $state(false);
  let errorMsg = $state('');

  // Selection — keyed by composite "finding_type:id" to avoid collisions across types
  /** @type {Set<string>} */
  let selected = $state(new Set());

  /** @param {object} f */
  function rowKey(f) {
    return `${f.finding_type ?? 'post'}:${f.id}`;
  }

  // Recovery modal
  /** @type {object | null} */
  let recoveryFinding = $state(null);

  // Abort controller so rapid filter changes cancel in-flight requests
  let loadAbortCtrl = /** @type {AbortController | null} */ (null);

  async function loadFindings() {
    if (loadAbortCtrl) loadAbortCtrl.abort();
    const ctrl = new AbortController();
    loadAbortCtrl = ctrl;

    loading = true;
    errorMsg = '';
    try {
      const params = {};
      if (platformFilter) params.platform = platformFilter;
      if (reasonFilter) params.reason = reasonFilter;
      const result = await api.findings(projectId, params);
      if (ctrl.signal.aborted) return;
      findings = result?.items ?? [];
      // Clear selection on reload
      selected = new Set();
    } catch (e) {
      if (ctrl.signal.aborted) return;
      errorMsg = e?.message ?? 'Failed to load filtered findings.';
    } finally {
      if (!ctrl.signal.aborted) loading = false;
    }
  }

  // Load on mount and when filters change
  $effect(() => {
    // Track filter dependencies
    void platformFilter;
    void reasonFilter;
    loadFindings();
  });

  function toggleAll(checked) {
    if (checked) {
      selected = new Set(findings.map(rowKey));
    } else {
      selected = new Set();
    }
  }

  function toggleRow(finding) {
    const key = rowKey(finding);
    const next = new Set(selected);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    selected = next;
  }

  async function handleRecheckSelected() {
    if (selected.size === 0) return;
    try {
      const targets = [...selected].map((key) => {
        const [finding_type, ...rest] = key.split(':');
        return { finding_type, id: rest.join(':') };
      });
      const job = await api.createJob(projectId, {
        requested_scope: 'selected_filtered_findings',
        targets,
      });
      onJobCreated?.(job);
    } catch (e) {
      errorMsg = e?.message ?? 'Failed to create re-check job.';
    }
  }

  async function handleRecover(finding, { mode, trust }) {
    try {
      const body = { finding_type: finding.finding_type, id: finding.id, mode };
      if (mode === 'restore_and_trust' && trust) {
        body.trust = trust;
      }
      await api.recover(projectId, body);
      recoveryFinding = null;
      await loadFindings();
      onRestored?.();
    } catch (e) {
      errorMsg = e?.message ?? 'Failed to recover finding.';
    }
  }

  function formatSourceIdentity(si) {
    if (!si || typeof si !== 'object') return { text: si ?? '—', url: null };
    if (si.reddit_author) return { text: `u/${si.reddit_author}`, url: `https://reddit.com/user/${si.reddit_author}` };
    if (si.subreddit) return { text: `r/${si.subreddit}`, url: `https://reddit.com/r/${si.subreddit}` };
    if (si.bluesky_cid) return { text: si.bluesky_cid, url: null };
    if (si.domain) return { text: si.domain, url: null };
    return { text: '—', url: null };
  }

  function formatDate(val) {
    if (!val) return '—';
    try {
      return new Date(val).toLocaleString();
    } catch {
      return val;
    }
  }

  function reasonDetails(finding) {
    const primary = finding.filter_reason;
    const all = finding.filter_reasons ?? [];
    const detailReasons = all.filter(r => r !== primary);
    const explanation = finding.filter_explanation;
    return { primary, detailReasons, explanation };
  }

  let allSelected = $derived(findings.length > 0 && selected.size === findings.length);
</script>

<section class="filtered-findings-tab">
  <div class="tab-header">
    <h2 class="tab-title">{FILTERED_FINDINGS_LABEL}</h2>

    <div class="filters">
      <label class="filter-label">
        Platform
        <select bind:value={platformFilter}>
          <option value="">All</option>
          <option value="reddit">Reddit</option>
          <option value="bluesky">Bluesky</option>
          <option value="google">Google</option>
        </select>
      </label>

      <label class="filter-label">
        Reason
        <select bind:value={reasonFilter}>
          <option value="">All</option>
          <option value="spam">Spam or promotion</option>
          <option value="bot_like">Bot-like activity</option>
          <option value="low_quality_account">Low-quality account</option>
          <option value="ai_generated">Likely AI-generated</option>
          <option value="trusted_override">Trusted override</option>
        </select>
      </label>

      <button
        class="btn-action"
        disabled={selected.size === 0}
        onclick={handleRecheckSelected}
      >
        {RECHECK_SELECTED_LABEL}
      </button>
    </div>
  </div>

  {#if errorMsg}
    <p class="error">{errorMsg}</p>
  {/if}

  {#if loading}
    <p class="status">Loading…</p>
  {:else if findings.length === 0}
    <div class="empty-state">
      <p class="empty-title">{FILTERED_EMPTY_TITLE}</p>
      <p class="empty-body">{FILTERED_EMPTY_BODY}</p>
    </div>
  {:else}
    <div class="table-wrapper">
      <table class="findings-table">
        <thead>
          <tr>
            <th>
              <input
                type="checkbox"
                checked={allSelected}
                onchange={(e) => toggleAll(e.currentTarget.checked)}
                aria-label="Select all"
              />
            </th>
            <th>Platform</th>
            <th>Source identity</th>
            <th>Score</th>
            <th>Reason</th>
            <th>Confidence</th>
            <th>Filtered time</th>
            <th>Source</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each findings as finding (rowKey(finding))}
            <tr class:row-selected={selected.has(rowKey(finding))}>
              <td>
                <input
                  type="checkbox"
                  checked={selected.has(rowKey(finding))}
                  onchange={() => toggleRow(finding)}
                  aria-label={`Select ${formatSourceIdentity(finding.source_identity).text}`}
                />
              </td>
              <td>{finding.platform ?? '—'}</td>
              <td class="identity-cell">
                  {#if formatSourceIdentity(finding.source_identity).url}
                    <a class="identity-link" href={formatSourceIdentity(finding.source_identity).url} target="_blank" rel="noopener">{formatSourceIdentity(finding.source_identity).text}</a>
                  {:else}
                    {formatSourceIdentity(finding.source_identity).text}
                  {/if}
                </td>
              <td>{finding.score != null ? finding.score : '—'}</td>
              <td class="reason-cell">
                  <span class="reason-primary" title={finding.filter_explanation ?? ''}>{reasonLabel(finding.filter_reason)}</span>
                  {#if reasonDetails(finding).detailReasons.length > 0}
                    <div class="sub-reasons">
                      {#each reasonDetails(finding).detailReasons as r}
                        <span class="reason-badge">{reasonLabel(r)}</span>
                      {/each}
                    </div>
                  {/if}
                </td>
              <td>{finding.filter_confidence != null ? `${Math.round(finding.filter_confidence * 100)}%` : '—'}</td>
              <td>{formatDate(finding.filtered_at)}</td>
              <td>{finding.filter_source === 'ai' ? 'AI' : finding.filter_source === 'rules' ? 'Rules' : finding.filter_source ?? '—'}</td>
              <td>
                <button
                  class="btn-restore"
                  aria-label={`Restore ${formatSourceIdentity(finding.source_identity).text}`}
                  onclick={() => { recoveryFinding = finding; }}
                >
                  Restore…
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</section>

{#if recoveryFinding}
  <FilterRecoveryModal
    finding={recoveryFinding}
    onClose={() => { recoveryFinding = null; }}
    onRecover={(opts) => handleRecover(recoveryFinding, opts)}
  />
{/if}

<style>
  .filtered-findings-tab {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .tab-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .tab-title {
    font-size: 1.125rem;
    font-weight: 600;
    margin: 0;
    color: #e2e2e8;
  }

  .filters {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .filter-label {
    display: flex;
    flex-direction: column;
    font-size: 0.75rem;
    font-weight: 500;
    color: #9a9ab0;
    gap: 0.25rem;
  }

  .filter-label select {
    font-size: 13px;
    padding: 7px 10px;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    background: #0f0f13;
    color: #e2e2e8;
    cursor: pointer;
  }

  .filter-label select:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .btn-action {
    padding: 7px 16px;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    font-size: 13px;
    font-weight: 500;
    background: #7c6af5;
    color: #fff;
    align-self: flex-end;
    transition: background 0.15s;
  }

  .btn-action:hover:not(:disabled) {
    background: #6a58e0;
  }

  .btn-action:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .status {
    color: #9a9ab0;
  }

  .error {
    color: #f87171;
  }

  .empty-state {
    padding: 2rem 1rem;
    text-align: center;
  }

  .empty-title {
    font-weight: 600;
    font-size: 1rem;
    margin: 0 0 0.5rem;
    color: #e2e2e8;
  }

  .empty-body {
    color: #9a9ab0;
    font-size: 0.875rem;
    max-width: 480px;
    margin: 0 auto;
  }

  .table-wrapper {
    overflow-x: auto;
  }

  .findings-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }

  .findings-table th,
  .findings-table td {
    padding: 0.5rem 0.75rem;
    text-align: left;
    border-bottom: 1px solid #2a2a3a;
    white-space: nowrap;
    color: #e2e2e8;
  }

  .findings-table th {
    font-weight: 600;
    color: #9a9ab0;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .row-selected {
    background: #1a1a2e;
  }

  .identity-cell {
    max-width: 180px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .btn-restore {
    padding: 5px 10px;
    border-radius: 6px;
    border: 1px solid #2a2a3a;
    background: #0f0f13;
    cursor: pointer;
    font-size: 12px;
    color: #e2e2e8;
  }

  .btn-restore:hover {
    background: #1a1a2e;
    border-color: #7c6af5;
  }

  .reason-cell {
    max-width: 200px;
  }

  .reason-primary {
    cursor: help;
  }

  .sub-reasons {
    display: flex;
    flex-wrap: wrap;
    gap: 3px;
    margin-top: 3px;
  }

  .reason-badge {
    display: inline-block;
    padding: 1px 6px;
    background: #2a2a3a;
    border-radius: 3px;
    font-size: 10px;
    color: #9a9ab0;
    white-space: nowrap;
  }

  .identity-link {
    color: #7c6af5;
    text-decoration: none;
  }

  .identity-link:hover {
    text-decoration: underline;
  }
</style>
