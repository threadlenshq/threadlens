<script>
  import { queries as queriesApi, queryReviewJobs as queryReviewJobsApi } from '../lib/api.js';
  import Surface from './ui/Surface.svelte';
  import QueryJobReviewModal from './QueryJobReviewModal.svelte';

  let {
    projectId,
    reviewJob = null,
    onQueriesChanged,
    onQueryReviewJobStarted,
    onQueryReviewJobHandled,
    onQueryReviewModalClosed,
  } = $props();

  const MIN_RECOMMENDED_QUERIES = 8;
  const MIN_RECOMMENDED_ANGLES = 3;
  const PLATFORM_LABELS = { reddit: 'Reddit', bluesky: 'Bluesky', google: 'Google' };
  const QUALITY_LABEL_FALLBACK = 'No signal yet';
  const QUALITY_SUMMARY_FALLBACK = 'No completed social or Google reports yet.';
  const QUERY_VISIBILITY_OPTIONS = [
    { value: 'enabled', label: 'Enabled only' },
    { value: 'disabled', label: 'Disabled only' },
    { value: 'all', label: 'Both' },
  ];

  function filterQueryList(query, visibility) {
    if (visibility === 'enabled') return !!query.enabled;
    if (visibility === 'disabled') return !query.enabled;
    return true;
  }

  function filteredEmptyMessage(totalCount, visibility) {
    if (totalCount === 0) return 'Start with one narrow query, such as "meeting notes too time consuming", then inspect results before expanding.';
    if (visibility === 'enabled') return 'No enabled queries match this filter.';
    if (visibility === 'disabled') return 'No disabled queries match this filter.';
    return 'No queries match this filter.';
  }

  function formatQualityScore(query) {
    return Number.isFinite(query?.quality?.score) ? String(query.quality.score) : '--';
  }

  function qualityTone(query) {
    return query?.quality?.level || 'unknown';
  }

  let queryList = $state([]);
  let queryVisibility = $state('enabled');

  function extractKeyword(query) {
    if (!query) return '';
    if (query.platform === 'reddit') {
      const raw = String(query.query_url || '').trim();
      try {
        const parsed = new URL(raw);
        const q = parsed.searchParams.get('q');
        if (q) return q.trim();
      } catch {
        const match = raw.match(/[?&]q=([^&]+)/i);
        if (match?.[1]) {
          return decodeURIComponent(match[1].replace(/\+/g, ' ')).trim();
        }
      }
    }
    return String(query.query_url || '').trim();
  }

  let enabledQueries = $derived(queryList.filter(q => q.enabled));
  let enabledCount = $derived(enabledQueries.length);
  let visibleQueries = $derived(queryList.filter(query => filterQueryList(query, queryVisibility)));
  let visibleCount = $derived(visibleQueries.length);
  let emptyMessage = $derived(filteredEmptyMessage(queryList.length, queryVisibility));
  let uniqueAngles = $derived(new Set(enabledQueries.map(q => q.angle).filter(Boolean)));
  let angleCount = $derived(uniqueAngles.size);
  let showCountWarning = $derived(enabledCount < MIN_RECOMMENDED_QUERIES);
  let showAngleTip = $derived(!showCountWarning && angleCount < MIN_RECOMMENDED_ANGLES);
  let loading = $state(false);
  let redditQueries = $derived(
    visibleQueries
      .filter(q => q.platform === 'reddit')
      .sort((a, b) => `${extractKeyword(a)}:${a.angle || ''}:${a.id}`.localeCompare(`${extractKeyword(b)}:${b.angle || ''}:${b.id}`))
  );
  let blueskyQueries = $derived(
    visibleQueries
      .filter(q => q.platform === 'bluesky')
      .sort((a, b) => `${extractKeyword(a)}:${a.angle || ''}:${a.id}`.localeCompare(`${extractKeyword(b)}:${b.angle || ''}:${b.id}`))
  );
  let googleQueries = $derived(
    visibleQueries
      .filter(q => q.platform === 'google')
      .sort((a, b) => `${a.query_url}:${a.angle || ''}:${a.id}`.localeCompare(`${b.query_url}:${b.angle || ''}:${b.id}`))
  );
  let error = $state('');

  // Add form state
  let newPlatform = $state('reddit');
  let newUrl = $state('');
  let newAngle = $state('');
  let adding = $state(false);

  // Suggest / Refine confirm modal state
  let showSuggestConfirmModal = $state(false);
  let showRefineConfirmModal = $state(false);
  let suggestRefinement = $state('');
  let suggestError = $state('');
  let refineError = $state('');
  let startingJob = $state(false);

  async function loadQueries() {
    loading = true;
    error = '';
    try {
      queryList = await queriesApi.list(projectId);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function toggleEnabled(q) {
    try {
      const updated = await queriesApi.update(projectId, q.id, { enabled: !q.enabled });
      queryList = queryList.map(item => item.id === q.id ? { ...item, ...updated } : item);
      onQueriesChanged?.({ projectId });
    } catch (e) {
      error = e.message;
    }
  }

  async function deleteQuery(q) {
    if (!confirm(`Delete this query?`)) return;
    try {
      await queriesApi.delete(projectId, q.id);
      queryList = queryList.filter(item => item.id !== q.id);
      onQueriesChanged?.({ projectId });
    } catch (e) {
      error = e.message;
    }
  }

  async function addQuery() {
    if (!newUrl.trim()) return;
    adding = true;
    error = '';
    try {
      const created = await queriesApi.create(projectId, {
        platform: newPlatform,
        query_url: newUrl.trim(),
        angle: newAngle.trim() || null,
      });
      queryList = [...queryList, created];
      onQueriesChanged?.({ projectId });
      newUrl = '';
      newAngle = '';
    } catch (e) {
      error = e.message;
    } finally {
      adding = false;
    }
  }

  function openSuggestConfirm() {
    suggestError = '';
    showSuggestConfirmModal = true;
  }

  function openRefineConfirm() {
    refineError = '';
    showRefineConfirmModal = true;
  }

  function closeSuggestConfirm() {
    showSuggestConfirmModal = false;
  }

  function closeRefineConfirm() {
    showRefineConfirmModal = false;
  }

  async function suggestQueries() {
    startingJob = true;
    suggestError = '';
    showSuggestConfirmModal = false;
    try {
      const job = await queryReviewJobsApi.create(projectId, {
        kind: 'suggest',
        refinement: suggestRefinement || undefined,
      });
      suggestRefinement = '';
      onQueryReviewJobStarted?.(job);
    } catch (e) {
      suggestError = e.message;
    } finally {
      startingJob = false;
    }
  }

  async function refineQueries() {
    startingJob = true;
    suggestError = '';
    showRefineConfirmModal = false;
    try {
      const job = await queryReviewJobsApi.create(projectId, {
        kind: 'refine',
        refinement: suggestRefinement || undefined,
      });
      suggestRefinement = '';
      onQueryReviewJobStarted?.(job);
    } catch (e) {
      suggestError = e.message;
    } finally {
      startingJob = false;
    }
  }

  function closeOnEscape(event, close) {
    if (event.key === 'Escape') {
      close();
    }
  }

  function truncate(str, len = 60) {
    if (!str) return '';
    return str.length > len ? str.slice(0, len) + '...' : str;
  }

  function webUrl(query) {
    if (query.platform === 'reddit') {
      return query.query_url.replace('.json', '');
    }
    if (query.platform === 'bluesky') {
      try {
        const url = new URL(query.query_url);
        const q = url.searchParams.get('q');
        return `https://bsky.app/search?q=${encodeURIComponent(q || query.query_url)}`;
      } catch {
        return `https://bsky.app/search?q=${encodeURIComponent(query.query_url)}`;
      }
    }
    if (query.platform === 'google') {
      return `https://www.google.com/search?q=${encodeURIComponent(query.query_url)}`;
    }
    return query.query_url;
  }

  let lastProjectId = null;
  $effect(() => {
    if (projectId && projectId !== lastProjectId) {
      lastProjectId = projectId;
      showSuggestConfirmModal = false;
      showRefineConfirmModal = false;
      suggestRefinement = '';
      onQueryReviewModalClosed?.();
      loadQueries();
    }
  });
</script>

<svelte:window onkeydown={(e) => {
  if (e.key !== 'Escape') return;
  if (showRefineConfirmModal) {
    closeRefineConfirm();
    return;
  }
  if (showSuggestConfirmModal) closeSuggestConfirm();
}} />

<div class="query-editor">
  <div class="section-header">
    <h3 class="section-title">
      Search Queries
      <a class="doc-link" href="https://docs.threadlens.dev/user-guide/scouting-sources/" target="_blank" rel="noopener" title="How scouting sources and queries work">?</a>
    </h3>
    <div class="header-actions">
      <div class="query-filter" role="group" aria-label="Query visibility filter">
        {#each QUERY_VISIBILITY_OPTIONS as option}
          <button
            type="button"
            class="query-filter-btn"
            class:active={queryVisibility === option.value}
            aria-pressed={queryVisibility === option.value}
            onclick={() => queryVisibility = option.value}
          >
            {option.label}
          </button>
        {/each}
      </div>
      <button class="suggest-btn" onclick={openRefineConfirm} disabled={startingJob}>
        {startingJob ? 'Starting...' : 'Refine Queries'}
      </button>
      <button class="suggest-btn" onclick={openSuggestConfirm} disabled={startingJob}>
        {startingJob ? 'Starting...' : 'Suggest Queries'}
      </button>
      <span class="count">{visibleCount} {visibleCount === 1 ? 'query' : 'queries'}</span>
      <a class="doc-link" href="https://docs.threadlens.dev/user-guide/scouting-sources/" target="_blank" rel="noopener" title="AI can help suggest or refine queries">?</a>
    </div>
  </div>

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#if suggestError}
    <div class="error-msg">{suggestError}</div>
  {/if}

  {#if showCountWarning}
    <div class="info-banner">
      One narrow query is enough for first value. After you review a completed scout, expand toward 8 enabled queries across at least 3 angles for stronger recurring signal.
    </div>
  {:else if showAngleTip}
    <div class="tip-banner">
      Tip: Your queries cover {angleCount} {angleCount === 1 ? 'angle' : 'angles'}. Try covering at least {MIN_RECOMMENDED_ANGLES} different angles for broader results.
    </div>
  {/if}

  {#if loading}
    <div class="loading">Loading queries...</div>
  {:else if visibleQueries.length === 0}
    <div class="empty-msg">{emptyMessage}</div>
  {:else}
    <div class="platform-sections">
      {#each [{ key: 'reddit', label: 'Reddit', items: redditQueries }, { key: 'bluesky', label: 'Bluesky', items: blueskyQueries }, { key: 'google', label: 'Google Search', items: googleQueries }] as platform (platform.key)}
        {#if platform.items.length > 0}
          <Surface elevation="base" padding="none">
            <div class="platform-section">
              <div class="platform-section-header">
                <span class="platform-section-title">{platform.label}</span>
                <span class="keyword-count">{platform.items.length}</span>
              </div>
              <div class="query-list">
                {#each platform.items as q (q.id)}
                  <div class="query-row">
                    <div class="query-row-primary">
                      <div class="query-row-main">
                        <span class="query-url" title={q.query_url}>{truncate(extractKeyword(q))}</span>
                        <a class="external-link" href={webUrl(q)} target="_blank" rel="noopener noreferrer" title="Open in browser">
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                            <polyline points="15 3 21 3 21 9"/>
                            <line x1="10" y1="14" x2="21" y2="3"/>
                          </svg>
                        </a>
                        {#if q.angle}
                          <span class="angle-tag">{q.angle}</span>
                        {/if}
                      </div>

                      <div class="query-row-actions">
                        <label class="toggle" title={q.enabled ? 'Disable' : 'Enable'}>
                          <input type="checkbox" checked={q.enabled} onchange={() => toggleEnabled(q)} />
                          <span class="toggle-slider"></span>
                        </label>
                        <button class="delete-btn" onclick={() => deleteQuery(q)} title="Delete query">&#x2715;</button>
                      </div>
                    </div>

                    <div class="query-row-secondary" title={q.quality?.summary || 'No quality signal yet'}>
                      <span class="quality-score">{formatQualityScore(q)}</span>
                      <span
                        class="quality-badge"
                        class:strong={qualityTone(q) === 'strong'}
                        class:mixed={qualityTone(q) === 'mixed'}
                        class:weak={qualityTone(q) === 'weak'}
                        class:unknown={qualityTone(q) === 'unknown'}
                      >
                        {q.quality?.label || QUALITY_LABEL_FALLBACK}
                      </span>
                      <span class="quality-summary">{q.quality?.summary || QUALITY_SUMMARY_FALLBACK}</span>
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          </Surface>
        {/if}
      {/each}
    </div>
  {/if}

  <div class="add-form">
    <div class="add-form-title">
      Add Query
      <a class="doc-link" href="https://docs.threadlens.dev/user-guide/scouting-sources/" target="_blank" rel="noopener" title="Add queries for each platform to scout">?</a>
    </div>
    <div class="form-row">
      <select bind:value={newPlatform} class="platform-select">
        <option value="reddit">Reddit</option>
        <option value="bluesky">Bluesky</option>
        <option value="google">Google</option>
      </select>
      <input
        class="angle-input"
        type="text"
        placeholder="Angle (optional)"
        bind:value={newAngle}
      />
    </div>
    <textarea
      class="url-textarea"
      placeholder={newPlatform === 'google' ? 'Root keyword (e.g., remote developer burnout)' : 'Query URL'}
      bind:value={newUrl}
      rows="2"
    ></textarea>
     <button class="add-btn" onclick={addQuery} disabled={adding || !newUrl.trim()}>
      {adding ? 'Adding...' : 'Add Query'}
    </button>
  </div>

  {#if showSuggestConfirmModal}
    <div class="modal-overlay" role="presentation" tabindex="-1" onclick={(e) => { if (e.target === e.currentTarget) closeSuggestConfirm(); }} onkeydown={(e) => closeOnEscape(e, closeSuggestConfirm)}>
      <div class="modal confirm-modal">
        <div class="modal-header">
          <h3 class="modal-title">Refine Query Suggestions</h3>
          <button class="modal-close" onclick={closeSuggestConfirm}>&#x2715;</button>
        </div>
        <div class="confirm-modal-body">
          <p class="confirm-modal-text">
            Optionally add context to steer the next suggestion run.
          </p>
          <textarea
            class="refinement-input"
            bind:value={suggestRefinement}
            rows="4"
            placeholder="Example: 51% of results are weak fit. Consider tighter root keywords or adding forum-biased queries."
          ></textarea>
        </div>
        <div class="modal-actions">
          <button class="add-btn" onclick={suggestQueries} disabled={startingJob}>
            {startingJob ? 'Starting...' : 'Generate Suggestions'}
          </button>
          <button class="cancel-btn" onclick={closeSuggestConfirm}>Cancel</button>
        </div>
      </div>
    </div>
  {/if}

  {#if showRefineConfirmModal}
    <div class="modal-overlay" role="presentation" tabindex="-1" onclick={(e) => { if (e.target === e.currentTarget) closeRefineConfirm(); }} onkeydown={(e) => closeOnEscape(e, closeRefineConfirm)}>
      <div class="modal confirm-modal">
        <div class="modal-header">
          <h3 class="modal-title">Refine Queries</h3>
          <button class="modal-close" onclick={closeRefineConfirm}>&#x2715;</button>
        </div>
        <div class="confirm-modal-body">
          <p class="confirm-modal-text">
            Analyze the current query set against the project’s latest reports and suggest what to turn off or add next.
          </p>
          <textarea
            class="refinement-input"
            bind:value={suggestRefinement}
            rows="4"
            placeholder="Optional: bias toward buyer-language keywords, trim broad discovery terms, or lean more on Google findings."
          ></textarea>
        </div>
        <div class="modal-actions">
          <button class="add-btn" onclick={refineQueries} disabled={startingJob}>
            {startingJob ? 'Starting...' : 'Generate Refinements'}
          </button>
          <button class="cancel-btn" onclick={closeRefineConfirm}>Cancel</button>
        </div>
      </div>
    </div>
  {/if}

  <QueryJobReviewModal
    {projectId}
    job={reviewJob}
    queries={queryList}
    onClose={onQueryReviewModalClosed}
    onHandled={onQueryReviewJobHandled}
    onQueriesChanged={() => { loadQueries(); onQueriesChanged?.({ projectId }); }}
  />
</div>

<style>
  .query-editor {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .count {
    font-size: 12px;
    color: #666;
  }

  .query-filter {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 3px;
    background: #14141d;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
  }

  .query-filter-btn {
    padding: 5px 10px;
    background: transparent;
    border: none;
    border-radius: 6px;
    color: #8d8da1;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .query-filter-btn.active {
    background: #2a2a45;
    color: #f1efff;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .info-banner {
    padding: 10px 14px;
    background: #1a1a3a;
    border: 1px solid #3a3a6a;
    border-radius: 6px;
    color: #a0a0d0;
    font-size: 13px;
    line-height: 1.5;
  }

  .info-banner strong {
    color: #7c6af5;
  }

  .tip-banner {
    padding: 10px 14px;
    background: #1a2a1a;
    border: 1px solid #2a4a2a;
    border-radius: 6px;
    color: #80c080;
    font-size: 13px;
    line-height: 1.5;
  }

  .loading,
  .empty-msg {
    color: #666;
    font-size: 14px;
    text-align: center;
    padding: 20px 0;
  }

  .platform-sections {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .platform-section {
    display: flex;
    flex-direction: column;
  }

  .platform-section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px 8px;
    border-bottom: 1px solid #2a2a3a;
  }

  .platform-section-title {
    font-size: 12px;
    font-weight: 700;
    color: #c9c9dc;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }

  .query-list {
    display: flex;
    flex-direction: column;
    padding: 8px 0;
  }

  .keyword-count {
    min-width: 20px;
    padding: 1px 7px;
    border-radius: 999px;
    background: #2a2a45;
    border: 1px solid #3c3c60;
    color: #a99af7;
    font-size: 11px;
    text-align: center;
  }

  .query-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    column-gap: 12px;
    row-gap: 6px;
    padding: 10px 16px;
    border-bottom: 1px solid #1e1e2c;
  }

  .query-row:last-child {
    border-bottom: none;
  }

  .query-row-primary {
    display: contents;
  }

  .query-row-main {
    grid-column: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .query-row-actions {
    grid-column: 2;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .query-row-secondary {
    grid-column: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    color: #9fa3b8;
    font-size: 12px;
  }

  .platform-badge {
    flex-shrink: 0;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .platform-badge.reddit {
    background: #ff4500;
    color: #fff;
  }

  .platform-badge.bluesky {
    background: #0085ff;
    color: #fff;
  }

  .platform-badge.google {
    background: #34a853;
    color: #fff;
  }

  .query-url {
    flex: 1;
    min-width: 0;
    font-size: 13px;
    color: #c0c0d0;
    font-family: monospace;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .external-link {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    color: #666;
    border-radius: 4px;
    transition: all 0.15s;
    text-decoration: none;
  }

  .external-link:hover {
    color: #7c6af5;
    background: #7c6af520;
  }

  .angle-tag {
    flex-shrink: 0;
    padding: 2px 8px;
    background: #2a2a45;
    border: 1px solid #7c6af5;
    border-radius: 4px;
    font-size: 11px;
    color: #a99af7;
  }

  .quality-score {
    min-width: 32px;
    font-weight: 700;
    color: #f5f7ff;
    text-align: right;
  }

  .quality-badge {
    flex-shrink: 0;
    padding: 2px 8px;
    border-radius: 999px;
    font-size: 11px;
    font-weight: 600;
  }

  .quality-badge.strong {
    background: #17301f;
    color: #7ee787;
    border: 1px solid #2d6a3d;
  }

  .quality-badge.mixed {
    background: #302914;
    color: #f2cc60;
    border: 1px solid #6f5a22;
  }

  .quality-badge.weak {
    background: #341b1b;
    color: #ff9b9b;
    border: 1px solid #7a3131;
  }

  .quality-badge.unknown {
    background: #232734;
    color: #aab3c5;
    border: 1px solid #3a4154;
  }

  .quality-summary {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: #8a8f9e;
  }

  .toggle {
    flex-shrink: 0;
    position: relative;
    width: 36px;
    height: 20px;
    cursor: pointer;
  }

  .toggle input {
    opacity: 0;
    width: 0;
    height: 0;
    position: absolute;
  }

  .toggle-slider {
    position: absolute;
    inset: 0;
    background: #333;
    border-radius: 20px;
    transition: background 0.2s;
  }

  .toggle-slider::after {
    content: '';
    position: absolute;
    width: 14px;
    height: 14px;
    left: 3px;
    top: 3px;
    background: #fff;
    border-radius: 50%;
    transition: transform 0.2s;
  }

  .toggle input:checked + .toggle-slider {
    background: #7c6af5;
  }

  .toggle input:checked + .toggle-slider::after {
    transform: translateX(16px);
  }

  .delete-btn {
    flex-shrink: 0;
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid #352c35;
    border-radius: 6px;
    color: #7f8696;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .delete-btn:hover {
    background: #3a1a1a;
    border-color: #f87171;
    color: #f87171;
  }

  @media (max-width: 720px) {
    .section-header,
    .header-actions {
      align-items: flex-start;
      flex-wrap: wrap;
    }

    .query-row-main {
      flex-wrap: wrap;
    }

    .query-row-actions {
      align-self: flex-start;
    }

    .query-row-secondary {
      flex-wrap: wrap;
    }
  }

  .add-form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
  }

  .add-form-title {
    font-size: 13px;
    font-weight: 600;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .form-row {
    display: flex;
    gap: 10px;
  }

  .platform-select {
    padding: 7px 10px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
    cursor: pointer;
  }

  .angle-input {
    flex: 1;
    padding: 7px 10px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
  }

  .angle-input::placeholder,
  .url-textarea::placeholder {
    color: #555;
  }

  .url-textarea {
    width: 100%;
    padding: 8px 10px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
    font-family: monospace;
    resize: vertical;
  }

  .url-textarea:focus,
  .angle-input:focus,
  .platform-select:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .add-btn {
    align-self: flex-start;
    padding: 7px 16px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .add-btn:hover:not(:disabled) {
    background: #6a58e3;
  }

  .add-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .suggest-btn {
    padding: 5px 12px;
    background: transparent;
    border: 1px solid #7c6af5;
    border-radius: 6px;
    color: #7c6af5;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .suggest-btn:hover:not(:disabled) {
    background: #7c6af520;
  }

  .suggest-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal-overlay--blocking {
    background: rgba(3, 6, 14, 0.86);
    backdrop-filter: blur(8px);
  }

  .modal {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 12px;
    width: 90%;
    max-width: 700px;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .confirm-modal {
    max-width: 640px;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid #2a2a3a;
  }

  .modal-title {
    font-size: 15px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .modal-close {
    background: none;
    border: none;
    color: #888;
    font-size: 16px;
    cursor: pointer;
    padding: 4px;
  }

  .modal-close:hover {
    color: #e2e2e8;
  }

  .suggestion-list {
    flex: 1;
    overflow-y: auto;
    padding: 12px 20px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .confirm-modal-body {
    padding: 16px 20px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .confirm-modal-text {
    margin: 0;
    color: #a5a5b5;
    font-size: 13px;
    line-height: 1.5;
  }

  .refinement-input {
    width: 100%;
    padding: 10px 12px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
    resize: vertical;
  }

  .refinement-input::placeholder {
    color: #555;
  }

  .refinement-input:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .suggestion-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    cursor: pointer;
    transition: border-color 0.15s;
  }

  .suggestion-row:hover {
    border-color: #3a3a5a;
  }

  .suggestion-row.selected {
    border-color: #7c6af5;
  }

  .suggestion-row input[type="checkbox"] {
    accent-color: #7c6af5;
  }

  .suggestion-url {
    flex: 1;
    font-size: 13px;
    color: #c0c0d0;
    font-family: monospace;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .modal-actions {
    display: flex;
    gap: 10px;
    padding: 16px 20px;
    border-top: 1px solid #2a2a3a;
  }

  .refine-loading-shell {
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    border-bottom: 1px solid #2a2a3a;
    background: linear-gradient(180deg, rgba(124, 106, 245, 0.08), rgba(124, 106, 245, 0.02));
  }

  .refine-loading-intro {
    display: flex;
    align-items: flex-start;
    gap: 14px;
  }

  .refine-loading-spinner {
    width: 18px;
    height: 18px;
    margin-top: 2px;
    border: 2px solid rgba(169, 154, 247, 0.22);
    border-top-color: #a99af7;
    border-radius: 50%;
    animation: spin 0.9s linear infinite;
  }

  .refine-loading-title {
    font-size: 16px;
    font-weight: 600;
    color: #f1efff;
  }

  .refine-loading-text,
  .refine-loading-note {
    margin: 0;
    color: #b9b7ca;
    font-size: 13px;
    line-height: 1.55;
  }

  .refine-briefing {
    padding: 14px 16px;
    border-radius: 10px;
    background: rgba(15, 15, 19, 0.9);
    border: 1px solid #2d2d42;
  }

  .refine-briefing-label {
    display: inline-block;
    margin-bottom: 6px;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: #a99af7;
  }

  .refine-briefing-text {
    min-height: 21px;
    margin: 0;
    color: #e2e2e8;
    font-size: 14px;
    line-height: 1.5;
  }

  .typing-caret {
    display: inline-block;
    width: 8px;
    height: 1em;
    margin-left: 2px;
    vertical-align: text-bottom;
    border-right: 2px solid #a99af7;
    animation: blink 1s steps(1) infinite;
  }

  .suggestion-list--loading {
    padding-top: 18px;
  }

  .refine-skeleton {
    cursor: default;
    pointer-events: none;
  }

  .skeleton {
    position: relative;
    overflow: hidden;
    background: #202031;
    border-radius: 999px;
  }

  .skeleton::after {
    content: '';
    position: absolute;
    inset: 0;
    transform: translateX(-100%);
    background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.16), transparent);
    animation: shimmer 1.3s infinite;
  }

  .skeleton-checkbox {
    width: 16px;
    height: 16px;
    border-radius: 4px;
  }

  .skeleton-badge {
    width: 58px;
    height: 24px;
  }

  .skeleton-platform {
    width: 70px;
    height: 22px;
  }

  .skeleton-angle {
    width: 92px;
    height: 24px;
  }

  .skeleton-line {
    height: 10px;
    border-radius: 999px;
  }

  .skeleton-line--wide {
    width: 100%;
    max-width: 360px;
  }

  .skeleton-line--mid {
    width: 68%;
  }

  .skeleton-line--short {
    width: 48%;
  }

  .refine-summary-block {
    border-bottom: 1px solid #2a2a3a;
  }

  .refine-context-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    font-size: 12px;
    color: #8d8da1;
  }

  .refine-context-row span {
    padding: 3px 8px;
    border-radius: 999px;
    background: #11111a;
    border: 1px solid #2a2a3a;
  }

  .refine-row {
    align-items: flex-start;
  }

  .refine-action-badge {
    flex-shrink: 0;
    min-width: 62px;
    text-align: center;
    padding: 2px 8px;
    border-radius: 999px;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.02em;
  }

  .refine-action-badge.add {
    background: #1f3a26;
    border: 1px solid #356d43;
    color: #8be0a2;
  }

  .refine-action-badge.disable {
    background: #3a201f;
    border: 1px solid #6e3c38;
    color: #f0a39b;
  }

  .refine-copy {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .refine-meta {
    font-size: 11px;
    color: #8d8da1;
  }

  .refine-reason {
    font-size: 12px;
    color: #c4c4d4;
    line-height: 1.45;
  }

  .cancel-btn {
    padding: 7px 16px;
    background: transparent;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #888;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .cancel-btn:hover {
    border-color: #555;
    color: #e2e2e8;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @keyframes shimmer {
    100% {
      transform: translateX(100%);
    }
  }

  @keyframes blink {
    50% {
      opacity: 0;
    }
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
