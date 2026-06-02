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
      findings = result ?? [];
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

  function formatDate(val) {
    if (!val) return '—';
    try {
      return new Date(val).toLocaleString();
    } catch {
      return val;
    }
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
                  aria-label={`Select ${finding.source_identity ?? finding.id}`}
                />
              </td>
              <td>{finding.platform ?? '—'}</td>
              <td class="identity-cell">{finding.source_identity ?? '—'}</td>
              <td>{finding.score != null ? finding.score : '—'}</td>
              <td>{reasonLabel(finding.reason)}</td>
              <td>{finding.confidence != null ? `${Math.round(finding.confidence * 100)}%` : '—'}</td>
              <td>{formatDate(finding.filtered_at)}</td>
              <td>{finding.filter_source === 'ai' ? 'AI' : finding.filter_source === 'rules' ? 'Rules' : finding.filter_source ?? '—'}</td>
              <td>
                <button
                  class="btn-restore"
                  aria-label={`Restore ${finding.source_identity ?? finding.id}`}
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
    color: var(--color-text-muted, #6b7280);
    gap: 0.25rem;
  }

  .filter-label select {
    font-size: 0.875rem;
    padding: 0.25rem 0.5rem;
    border: 1px solid var(--color-border, #e5e7eb);
    border-radius: 4px;
    background: var(--color-surface, #fff);
    color: var(--color-text, #111);
  }

  .btn-action {
    padding: 0.4rem 0.9rem;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    font-size: 0.875rem;
    font-weight: 500;
    background: var(--color-accent, #4f46e5);
    color: #fff;
    align-self: flex-end;
  }

  .btn-action:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .status {
    color: var(--color-text-muted, #6b7280);
  }

  .error {
    color: var(--color-danger, #dc2626);
  }

  .empty-state {
    padding: 2rem 1rem;
    text-align: center;
  }

  .empty-title {
    font-weight: 600;
    font-size: 1rem;
    margin: 0 0 0.5rem;
  }

  .empty-body {
    color: var(--color-text-muted, #6b7280);
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
    font-size: 0.875rem;
  }

  .findings-table th,
  .findings-table td {
    padding: 0.5rem 0.75rem;
    text-align: left;
    border-bottom: 1px solid var(--color-border, #e5e7eb);
    white-space: nowrap;
  }

  .findings-table th {
    font-weight: 600;
    color: var(--color-text-muted, #6b7280);
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .row-selected {
    background: var(--color-surface-alt, #f9fafb);
  }

  .identity-cell {
    max-width: 180px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .btn-restore {
    padding: 0.25rem 0.6rem;
    border-radius: 4px;
    border: 1px solid var(--color-border, #e5e7eb);
    background: var(--color-surface, #fff);
    cursor: pointer;
    font-size: 0.8rem;
    color: var(--color-text, #111);
  }

  .btn-restore:hover {
    background: var(--color-surface-alt, #f3f4f6);
  }
</style>
