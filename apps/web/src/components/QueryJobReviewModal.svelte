<script>
  import Modal from './ui/Modal.svelte';
  import { queries as queriesApi, queryReviewJobs } from '../lib/api.js';

  let {
    projectId,
    job = null,
    queries = [],
    onClose,
    onHandled,
    onQueriesChanged,
  } = $props();

  // --- Derived job metadata ---

  let jobType = $derived(job?.job_type ?? '');
  let jobStatus = $derived(job?.status ?? '');
  let isFailed = $derived(jobStatus === 'failed');
  let suggestions = $derived(job?.result?.suggestions ?? []);
  let addItems = $derived(job?.result?.add ?? []);
  let disableItems = $derived(job?.result?.disable ?? []);

  // --- Selection state ---

  /**
   * For 'suggest' jobs: a Set of suggestion indices that are selected.
   * For 'refine' jobs: Sets of indices for add/disable items selected.
   */
  let selectedSuggestions = $state(new Set());
  let selectedAdds = $state(new Set());
  let selectedDisables = $state(new Set());

  // Initialise selection whenever the job changes
  $effect(() => {
    if (!job) return;

    if (jobType === 'suggest') {
      // Preselect all suggestions
      selectedSuggestions = new Set(suggestions.map((_, i) => i));
    } else if (jobType === 'refine') {
      // Preselect all add items
      selectedAdds = new Set(addItems.map((_, i) => i));

      // Preselect disable items that still reference an *enabled* query
      const enabledIds = new Set(queries.filter(q => q.enabled).map(q => q.id));
      selectedDisables = new Set(
        disableItems
          .map((item, i) => ({ item, i }))
          .filter(({ item }) => enabledIds.has(item.query_id))
          .map(({ i }) => i),
      );
    }
  });

  // --- Modal title ---

  let modalTitle = $derived(
    isFailed
      ? 'Query Review — Failed'
      : jobType === 'suggest'
        ? 'Suggested Queries'
        : jobType === 'refine'
          ? 'Refine Queries'
          : 'Query Review',
  );

  // --- Submission state ---

  let submitting = $state(false);
  let error = $state('');

  // --- Helpers ---

  function toggleSuggestion(i) {
    const next = new Set(selectedSuggestions);
    if (next.has(i)) next.delete(i);
    else next.add(i);
    selectedSuggestions = next;
  }

  function toggleAdd(i) {
    const next = new Set(selectedAdds);
    if (next.has(i)) next.delete(i);
    else next.add(i);
    selectedAdds = next;
  }

  function toggleDisable(i) {
    const next = new Set(selectedDisables);
    if (next.has(i)) next.delete(i);
    else next.add(i);
    selectedDisables = next;
  }

  function selectAllSuggestions() {
    selectedSuggestions = new Set(suggestions.map((_, i) => i));
  }

  function deselectAllSuggestions() {
    selectedSuggestions = new Set();
  }

  // --- Apply ---

  async function handleApply() {
    if (submitting || !job) return;
    submitting = true;
    error = '';

    try {
      if (jobType === 'suggest') {
        const toCreate = suggestions.filter((_, i) => selectedSuggestions.has(i));
        await Promise.all(
          toCreate.map(s =>
            queriesApi.create(projectId, {
              query_url: s.query_url,
              platform: s.platform,
              angle: s.angle ?? undefined,
              enabled: true,
            }),
          ),
        );
      } else if (jobType === 'refine') {
        const toAdd = addItems.filter((_, i) => selectedAdds.has(i));
        const toDisable = disableItems.filter((_, i) => selectedDisables.has(i));

        await Promise.all([
          ...toAdd.map(a =>
            queriesApi.create(projectId, {
              query_url: a.query_url,
              platform: a.platform,
              angle: a.angle ?? undefined,
              enabled: true,
            }),
          ),
          ...toDisable.map(d =>
            queriesApi.update(projectId, d.query_id, { enabled: false }),
          ),
        ]);
      }

      await queryReviewJobs.reviewed(projectId, job.id, { action: 'applied' });

      onQueriesChanged?.();
      onHandled?.();
      onClose?.();
    } catch (e) {
      error = e.message || 'Something went wrong. Please try again.';
    } finally {
      submitting = false;
    }
  }

  // --- Deny / Clear ---

  async function handleDeny() {
    if (submitting || !job) return;
    submitting = true;
    error = '';

    try {
      await queryReviewJobs.reviewed(projectId, job.id, { action: 'denied' });
      onHandled?.();
      onClose?.();
    } catch (e) {
      error = e.message || 'Something went wrong. Please try again.';
    } finally {
      submitting = false;
    }
  }

  // --- Derived counts for footer summary ---

  let applyCount = $derived(
    jobType === 'suggest'
      ? selectedSuggestions.size
      : selectedAdds.size + selectedDisables.size,
  );
</script>

<Modal open={job !== null} title={modalTitle} {onClose}>
  {#if isFailed}
    <div class="failed-body">
      <p class="failed-message">
        This query review job failed and could not produce suggestions.
        {#if job?.error}
          <span class="failed-detail">{job.error}</span>
        {/if}
      </p>
      <p class="failed-hint">You can clear this job to dismiss it.</p>
    </div>

  {:else if jobType === 'suggest'}
    {#if suggestions.length === 0}
      <p class="empty-notice">No suggestions were generated for this job.</p>
    {:else}
      <div class="section-header">
        <span class="section-label">
          Suggested queries
          <span class="count-badge">{suggestions.length}</span>
        </span>
        <div class="select-controls">
          <button class="link-btn" onclick={selectAllSuggestions} disabled={submitting}>All</button>
          <span class="divider">/</span>
          <button class="link-btn" onclick={deselectAllSuggestions} disabled={submitting}>None</button>
        </div>
      </div>

      <ul class="item-list">
        {#each suggestions as s, i (i)}
          <li class="item-row">
            <label class="item-label">
              <input
                type="checkbox"
                checked={selectedSuggestions.has(i)}
                onchange={() => toggleSuggestion(i)}
                disabled={submitting}
              />
              <div class="item-text">
                <span class="item-query">{s.query_url}</span>
                <span class="item-meta">
                  {s.platform}
                  {#if s.angle}<span class="item-angle">· {s.angle}</span>{/if}
                </span>
              </div>
            </label>
          </li>
        {/each}
      </ul>
    {/if}

  {:else if jobType === 'refine'}
    {#if addItems.length === 0 && disableItems.length === 0}
      <p class="empty-notice">No refinements were generated for this job.</p>
    {:else}
      {#if addItems.length > 0}
        <div class="section-header">
          <span class="section-label">
            Add queries
            <span class="count-badge">{addItems.length}</span>
          </span>
        </div>
        <ul class="item-list">
          {#each addItems as a, i (i)}
            <li class="item-row">
              <label class="item-label">
                <input
                  type="checkbox"
                  checked={selectedAdds.has(i)}
                  onchange={() => toggleAdd(i)}
                  disabled={submitting}
                />
                <div class="item-text">
                  <span class="item-query">{a.query_url}</span>
                  <span class="item-meta">
                    {a.platform}
                    {#if a.angle}<span class="item-angle">· {a.angle}</span>{/if}
                  </span>
                </div>
              </label>
            </li>
          {/each}
        </ul>
      {/if}

      {#if disableItems.length > 0}
        <div class="section-header" class:section-header-spaced={addItems.length > 0}>
          <span class="section-label">
            Disable queries
            <span class="count-badge">{disableItems.length}</span>
          </span>
        </div>
        <ul class="item-list">
          {#each disableItems as d, i (i)}
            {@const matchedQuery = queries.find(q => q.id === d.query_id)}
            <li class="item-row" class:item-row-stale={!matchedQuery?.enabled}>
              <label class="item-label">
                <input
                  type="checkbox"
                  checked={selectedDisables.has(i)}
                  onchange={() => toggleDisable(i)}
                  disabled={submitting}
                />
                <div class="item-text">
                  <span class="item-query">
                    {matchedQuery?.query_url ?? d.query_id}
                    {#if !matchedQuery?.enabled}
                      <span class="already-disabled">(already disabled)</span>
                    {/if}
                  </span>
                  <span class="item-meta">
                    {matchedQuery?.platform ?? ''}
                    {#if d.reason}<span class="item-angle">· {d.reason}</span>{/if}
                  </span>
                </div>
              </label>
            </li>
          {/each}
        </ul>
      {/if}
    {/if}
  {/if}

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#snippet footer()}
    {#if isFailed}
      <button class="btn-secondary" onclick={onClose} disabled={submitting}>Cancel</button>
      <button class="btn-danger" onclick={handleDeny} disabled={submitting}>
        {submitting ? 'Clearing…' : 'Clear Job'}
      </button>
    {:else}
      <button class="btn-secondary" onclick={handleDeny} disabled={submitting}>
        {submitting ? 'Dismissing…' : 'Dismiss'}
      </button>
      <button
        class="btn-primary"
        onclick={handleApply}
        disabled={submitting || applyCount === 0}
      >
        {submitting ? 'Applying…' : `Apply${applyCount > 0 ? ` (${applyCount})` : ''}`}
      </button>
    {/if}
  {/snippet}
</Modal>

<style>
  /* --- Body layouts --- */

  .failed-body {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .failed-message {
    margin: 0;
    color: #e2e2e8;
    font-size: 14px;
    line-height: 1.5;
  }

  .failed-detail {
    display: block;
    margin-top: 6px;
    color: #f87171;
    font-size: 13px;
    font-family: monospace;
    word-break: break-word;
  }

  .failed-hint {
    margin: 0;
    color: #777;
    font-size: 13px;
  }

  .empty-notice {
    margin: 0;
    color: #777;
    font-size: 14px;
  }

  /* --- Section headers --- */

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }

  .section-header-spaced {
    margin-top: 18px;
  }

  .section-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #888;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .count-badge {
    background: #2a2a3a;
    color: #aaa;
    font-size: 10px;
    font-weight: 500;
    padding: 1px 6px;
    border-radius: 10px;
  }

  .select-controls {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .link-btn {
    background: none;
    border: none;
    color: #7c6af5;
    font-size: 12px;
    cursor: pointer;
    padding: 2px 4px;
    border-radius: 3px;
    transition: opacity 0.15s;
  }

  .link-btn:hover:not(:disabled) {
    opacity: 0.75;
  }

  .link-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .divider {
    color: #555;
    font-size: 12px;
  }

  /* --- Item list --- */

  .item-list {
    list-style: none;
    margin: 0 0 4px;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .item-row {
    border-radius: 6px;
    background: #0f0f18;
    border: 1px solid #2a2a3a;
    transition: border-color 0.15s;
  }

  .item-row:has(input:checked) {
    border-color: #4a4070;
  }

  .item-row-stale {
    opacity: 0.6;
  }

  .item-label {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 9px 12px;
    cursor: pointer;
  }

  .item-label input[type='checkbox'] {
    margin-top: 2px;
    accent-color: #7c6af5;
    flex-shrink: 0;
    cursor: pointer;
  }

  .item-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .item-query {
    font-size: 13px;
    color: #e2e2e8;
    word-break: break-all;
  }

  .item-meta {
    font-size: 11px;
    color: #666;
  }

  .item-angle {
    color: #888;
  }

  .already-disabled {
    color: #f87171;
    font-size: 11px;
    margin-left: 4px;
  }

  /* --- Error --- */

  .error-msg {
    margin-top: 14px;
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  /* --- Footer buttons --- */

  .btn-secondary {
    padding: 8px 16px;
    background: none;
    border: 1px solid #3a3a4a;
    border-radius: 6px;
    color: #888;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .btn-secondary:hover:not(:disabled) {
    border-color: #555;
    color: #e2e2e8;
  }

  .btn-secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    padding: 8px 20px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .btn-primary:hover:not(:disabled) {
    opacity: 0.88;
  }

  .btn-primary:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .btn-danger {
    padding: 8px 20px;
    background: #c0392b;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .btn-danger:hover:not(:disabled) {
    opacity: 0.88;
  }

  .btn-danger:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
</style>
