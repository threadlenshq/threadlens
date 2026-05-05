<script>
  import { onDestroy } from 'svelte';
  import { queries as queriesApi } from '../lib/api.js';

  let { projectId, onQueriesChanged } = $props();

  const MIN_RECOMMENDED_QUERIES = 8;
  const MIN_RECOMMENDED_ANGLES = 3;
  const PLATFORM_LABELS = { reddit: 'Reddit', bluesky: 'Bluesky', google: 'Google' };
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
    if (totalCount === 0) return 'No queries yet. Add one below.';
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
  let groupedQueries = $derived((() => {
    const groups = new Map();
    for (const query of visibleQueries) {
      const keyword = extractKeyword(query) || '(empty)';
      const key = keyword.toLowerCase();
      if (!groups.has(key)) {
        groups.set(key, { keyword, items: [] });
      }
      groups.get(key).items.push(query);
    }
    return [...groups.values()]
      .map(group => ({
        ...group,
        items: [...group.items].sort((a, b) =>
          `${a.platform}:${a.angle || ''}:${a.id}`.localeCompare(`${b.platform}:${b.angle || ''}:${b.id}`)
        )
      }))
      .sort((a, b) => a.keyword.localeCompare(b.keyword));
  })());
  let loading = $state(false);
  let error = $state('');

  // Add form state
  let newPlatform = $state('reddit');
  let newUrl = $state('');
  let newAngle = $state('');
  let adding = $state(false);

  // Suggest feature state
  let suggesting = $state(false);
  let suggestions = $state([]);
  let selected = $state(new Set());
  let showSuggestConfirmModal = $state(false);
  let showSuggestModal = $state(false);
  let refining = $state(false);
  let refineRecommendations = $state([]);
  let refineSelected = $state(new Set());
  let refineSummary = $state('');
  let refineContext = $state(null);
  let showRefineConfirmModal = $state(false);
  let showRefineModal = $state(false);
  let refineError = $state('');
  let applyingRefine = $state(false);
  let suggestRefinement = $state('');
  let suggestError = $state('');
  let addingSelected = $state(false);
  const REFINE_LOADING_QUOTES = [
    'Reviewing active queries against your latest research signals.',
    'Looking for broad terms to trim and tighter buyer-language angles to add.',
    'Cross-checking social and Google reports for stronger query replacements.',
    'Building a reviewable refinement plan before anything changes in your project.'
  ];
  const REFINE_LOADING_ROW_COUNT = 4;

  let refineBriefingIndex = $state(0);
  let refineBriefingText = $state('');
  let refineTypingTimer = null;
  let refineQuoteTimer = null;
  let refineUnloadHandler = null;
  let wasRefining = false;

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
    suggesting = true;
    suggestError = '';
    showSuggestConfirmModal = false;
    try {
      const data = await queriesApi.suggest(projectId, { refinement: suggestRefinement });
      suggestions = data.suggestions || [];
      selected = new Set(suggestions.map((_, i) => i));
      suggestRefinement = '';
      showSuggestModal = true;
    } catch (e) {
      suggestError = e.message;
    } finally {
      suggesting = false;
    }
  }

  async function refineQueries() {
    refining = true;
    refineError = '';
    showRefineConfirmModal = false;
    showRefineModal = true;
    refineRecommendations = [];
    refineSelected = new Set();
    refineSummary = '';
    refineContext = null;
    try {
      const data = await queriesApi.refine(projectId, { refinement: suggestRefinement });
      refineRecommendations = data.recommendations || [];
      refineSelected = new Set(
        refineRecommendations
          .map((item, index) => item.type === 'add' ? index : null)
          .filter((value) => value !== null)
      );
      refineSummary = data.summary || '';
      refineContext = data.context || null;
      suggestRefinement = '';
      showRefineModal = true;
    } catch (e) {
      refineError = e.message;
      showRefineModal = false;
    } finally {
      refining = false;
    }
  }

  function toggleSuggestion(index) {
    if (selected.has(index)) {
      selected.delete(index);
    } else {
      selected.add(index);
    }
    selected = new Set(selected); // trigger reactivity
  }

  function toggleRefineRecommendation(index) {
    if (refineSelected.has(index)) {
      refineSelected.delete(index);
    } else {
      refineSelected.add(index);
    }
    refineSelected = new Set(refineSelected);
  }

  async function addSelectedSuggestions() {
    addingSelected = true;
    const results = await Promise.allSettled(
      [...selected].map(index => {
        const s = suggestions[index];
        return queriesApi.create(projectId, {
          platform: s.platform,
          query_url: s.query_url,
          angle: s.angle || null,
        });
      })
    );
    const created = results.filter(r => r.status === 'fulfilled').map(r => r.value);
    const failures = results.filter(r => r.status === 'rejected').length;
    queryList = [...queryList, ...created];
    if (created.length > 0) {
      onQueriesChanged?.({ projectId });
    }
    addingSelected = false;
    showSuggestModal = false;
    suggestions = [];
    selected = new Set();
    if (failures > 0) {
      error = `Added ${created.length} queries, ${failures} failed`;
    }
  }

  async function applySelectedRefinements() {
    applyingRefine = true;
    const chosen = [...refineSelected].map((index) => refineRecommendations[index]).filter(Boolean);
    const results = await Promise.allSettled(
      chosen.map((item) => {
        if (item.type === 'disable') {
          return queriesApi.update(projectId, item.query.id, { enabled: false });
        }
        return queriesApi.create(projectId, {
          platform: item.query.platform,
          query_url: item.query.query_url,
          angle: item.query.angle || null,
        });
      })
    );

    const failures = results.filter((result) => result.status === 'rejected').length;
    const changed = chosen.filter((_, index) => results[index].status === 'fulfilled');
    if (changed.length > 0) {
      await loadQueries();
      onQueriesChanged?.({ projectId });
    }
    applyingRefine = false;
    closeRefineModal();
    if (failures > 0) {
      error = `Applied ${changed.length} refinements, ${failures} failed`;
    }
  }

  function closeSuggestModal() {
    showSuggestModal = false;
    suggestions = [];
    selected = new Set();
    suggestError = '';
  }

  function closeRefineModal() {
    if (refining) return;
    showRefineModal = false;
    refineRecommendations = [];
    refineSelected = new Set();
    refineSummary = '';
    refineContext = null;
    refineError = '';
  }

  function recommendationLabel(item) {
    return item.type === 'disable' ? 'Turn off' : 'Add';
  }

  function recommendationMeta(item) {
    if (item.type === 'disable') return 'Current query';
    if (item.replaces_query_id) return `Suggested replacement for #${item.replaces_query_id}`;
    return 'Suggested addition';
  }

  function closeOnEscape(event, close) {
    if (event.key === 'Escape') {
      close();
    }
  }

  function startRefineBriefing() {
    stopRefineBriefing();
    refineBriefingIndex = 0;
    typeRefineBriefing(REFINE_LOADING_QUOTES[0]);
    refineQuoteTimer = setInterval(() => {
      refineBriefingIndex = (refineBriefingIndex + 1) % REFINE_LOADING_QUOTES.length;
      typeRefineBriefing(REFINE_LOADING_QUOTES[refineBriefingIndex]);
    }, 4200);
  }

  function typeRefineBriefing(text) {
    if (refineTypingTimer) clearInterval(refineTypingTimer);
    refineBriefingText = '';
    let index = 0;
    refineTypingTimer = setInterval(() => {
      index += 1;
      refineBriefingText = text.slice(0, index);
      if (index >= text.length) {
        clearInterval(refineTypingTimer);
        refineTypingTimer = null;
      }
    }, 24);
  }

  function stopRefineBriefing() {
    if (refineTypingTimer) {
      clearInterval(refineTypingTimer);
      refineTypingTimer = null;
    }
    if (refineQuoteTimer) {
      clearInterval(refineQuoteTimer);
      refineQuoteTimer = null;
    }
  }

  function setRefineUnloadProtection(active) {
    if (typeof window === 'undefined') return;
    if (active && !refineUnloadHandler) {
      refineUnloadHandler = (event) => {
        event.preventDefault();
        event.returnValue = '';
      };
      window.addEventListener('beforeunload', refineUnloadHandler);
      document.body.style.overflow = 'hidden';
      return;
    }
    if (!active && refineUnloadHandler) {
      window.removeEventListener('beforeunload', refineUnloadHandler);
      refineUnloadHandler = null;
      document.body.style.overflow = '';
    }
  }

  $effect(() => {
    if (refining && !wasRefining) {
      wasRefining = true;
      startRefineBriefing();
      setRefineUnloadProtection(true);
    } else if (!refining && wasRefining) {
      wasRefining = false;
      stopRefineBriefing();
      setRefineUnloadProtection(false);
    }
  });

  onDestroy(() => {
    stopRefineBriefing();
    setRefineUnloadProtection(false);
  });

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
      showSuggestModal = false;
      showRefineConfirmModal = false;
      showRefineModal = false;
      refining = false;
      suggestRefinement = '';
      loadQueries();
    }
  });
</script>

<svelte:window onkeydown={(e) => {
  if (e.key !== 'Escape') return;
  if (showSuggestModal) {
    closeSuggestModal();
    return;
  }
  if (showRefineModal) {
    closeRefineModal();
    return;
  }
  if (showRefineConfirmModal) {
    closeRefineConfirm();
    return;
  }
  if (showSuggestConfirmModal) closeSuggestConfirm();
}} />

<div class="query-editor">
  <div class="section-header">
    <h3 class="section-title">Search Queries</h3>
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
      <button class="suggest-btn" onclick={openRefineConfirm} disabled={refining || suggesting}>
        {refining ? 'Refining...' : 'Refine Queries'}
      </button>
      <button class="suggest-btn" onclick={openSuggestConfirm} disabled={suggesting}>
        {suggesting ? 'Suggesting...' : 'Suggest Queries'}
      </button>
      <span class="count">{visibleCount} {visibleCount === 1 ? 'query' : 'queries'}</span>
    </div>
  </div>

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#if suggestError}
    <div class="error-msg">{suggestError}</div>
  {/if}

  {#if refineError}
    <div class="error-msg">{refineError}</div>
  {/if}

  {#if showCountWarning}
    <div class="info-banner">
      For best results, we recommend at least {MIN_RECOMMENDED_QUERIES} search queries. You have {enabledCount} enabled. Use <strong>Suggest Queries</strong> to quickly add more.
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
    <div class="query-list">
      {#each groupedQueries as group (group.keyword)}
        <div class="query-group">
          <div class="query-group-header">
            <span class="keyword-label">{group.keyword}</span>
            <span class="keyword-count">{group.items.length}</span>
          </div>
          {#each group.items as q (q.id)}
            <div class="query-row">
              <div class="query-row-primary">
                <div class="query-row-main">
                  <span class="platform-badge" class:reddit={q.platform === 'reddit'} class:bluesky={q.platform === 'bluesky'} class:google={q.platform === 'google'}>
                    {PLATFORM_LABELS[q.platform] || q.platform}
                  </span>
                  <span class="query-url" title={q.query_url}>{truncate(q.query_url)}</span>
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
                  {q.quality?.label || 'No signal yet'}
                </span>
                <span class="quality-summary">{q.quality?.summary || 'No completed social or Google reports yet.'}</span>
              </div>
            </div>
          {/each}
        </div>
      {/each}
    </div>
  {/if}

  <div class="add-form">
    <div class="add-form-title">Add Query</div>
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
          <button class="add-btn" onclick={suggestQueries} disabled={suggesting}>
            {suggesting ? 'Suggesting...' : 'Generate Suggestions'}
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
          <button class="add-btn" onclick={refineQueries} disabled={refining}>
            {refining ? 'Refining...' : 'Generate Refinements'}
          </button>
          <button class="cancel-btn" onclick={closeRefineConfirm}>Cancel</button>
        </div>
      </div>
    </div>
  {/if}

  {#if showSuggestModal}
    <div class="modal-overlay" role="presentation" tabindex="-1" onclick={(e) => { if (e.target === e.currentTarget) closeSuggestModal(); }} onkeydown={(e) => closeOnEscape(e, closeSuggestModal)}>
      <div class="modal">
        <div class="modal-header">
          <h3 class="modal-title">Suggested Queries</h3>
          <button class="modal-close" onclick={closeSuggestModal}>&#x2715;</button>
        </div>
        {#if suggestions.length === 0}
          <div class="empty-msg">No suggestions generated. Try adding more project context.</div>
        {:else}
          <div class="suggestion-list">
            {#each suggestions as s, i}
              <label class="suggestion-row" class:selected={selected.has(i)}>
                <input type="checkbox" checked={selected.has(i)} onchange={() => toggleSuggestion(i)} />
                <span class="platform-badge" class:reddit={s.platform === 'reddit'} class:bluesky={s.platform === 'bluesky'} class:google={s.platform === 'google'}>
                  {PLATFORM_LABELS[s.platform] || s.platform}
                </span>
                <span class="suggestion-url" title={s.query_url}>{truncate(s.query_url, 50)}</span>
                <a class="external-link" href={webUrl(s)} target="_blank" rel="noopener noreferrer" title="Open in browser" onclick={(e) => e.stopPropagation()}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                    <polyline points="15 3 21 3 21 9"/>
                    <line x1="10" y1="14" x2="21" y2="3"/>
                  </svg>
                </a>
                {#if s.angle}
                  <span class="angle-tag">{s.angle}</span>
                {/if}
              </label>
            {/each}
          </div>
          <div class="modal-actions">
            <button class="add-btn" onclick={addSelectedSuggestions} disabled={addingSelected || selected.size === 0}>
              {addingSelected ? 'Adding...' : `Add ${selected.size} Selected`}
            </button>
            <button class="cancel-btn" onclick={closeSuggestModal}>Cancel</button>
          </div>
        {/if}
      </div>
    </div>
  {/if}

  {#if showRefineModal}
    <div class="modal-overlay" class:modal-overlay--blocking={refining} role="presentation" tabindex="-1" onclick={(e) => { if (e.target === e.currentTarget) closeRefineModal(); }} onkeydown={(e) => closeOnEscape(e, closeRefineModal)}>
      <div class="modal">
        <div class="modal-header">
          <h3 class="modal-title">Refine Query Recommendations</h3>
          {#if !refining}
            <button class="modal-close" onclick={closeRefineModal} aria-label="Close refine recommendations">&#x2715;</button>
          {/if}
        </div>
        {#if refining}
          <div class="refine-loading-shell">
            <div class="refine-loading-intro">
              <div class="refine-loading-spinner" aria-hidden="true"></div>
              <div>
                <div class="refine-loading-title">Refining your query set…</div>
                <p class="refine-loading-text">Keep this tab open while Scout reviews the current queries against your project context and latest reports.</p>
              </div>
            </div>
            <div class="refine-briefing" aria-live="polite">
              <span class="refine-briefing-label">AI briefing</span>
              <p class="refine-briefing-text">{refineBriefingText}<span class="typing-caret"></span></p>
            </div>
            <div class="refine-loading-note">This step can take a few minutes on larger report contexts. Navigation is temporarily locked so the request is not interrupted.</div>
          </div>
          <div class="suggestion-list suggestion-list--loading" aria-hidden="true">
            {#each Array.from({ length: REFINE_LOADING_ROW_COUNT }) as _, i}
              <div class="suggestion-row refine-row refine-skeleton" data-skeleton-index={i}>
                <div class="skeleton skeleton-checkbox"></div>
                <div class="skeleton skeleton-badge"></div>
                <div class="skeleton skeleton-platform"></div>
                <div class="refine-copy">
                  <div class="skeleton skeleton-line skeleton-line--wide"></div>
                  <div class="skeleton skeleton-line skeleton-line--mid"></div>
                  <div class="skeleton skeleton-line skeleton-line--short"></div>
                </div>
                <div class="skeleton skeleton-angle"></div>
              </div>
            {/each}
          </div>
        {:else if refineSummary || refineContext}
          <div class="confirm-modal-body refine-summary-block">
            {#if refineSummary}
              <p class="confirm-modal-text">{refineSummary}</p>
            {/if}
            {#if refineContext}
              <div class="refine-context-row">
                <span>{refineContext.enabled_query_count} enabled of {refineContext.query_count} total</span>
                {#if refineContext.social_report}
                  <span>Social report #{refineContext.social_report.id}</span>
                {/if}
                {#if refineContext.google_report}
                  <span>Google report #{refineContext.google_report.id}</span>
                {/if}
              </div>
            {/if}
          </div>
        {/if}
        {#if refineRecommendations.length === 0}
          <div class="empty-msg">No query refinements recommended right now.</div>
        {:else}
          <div class="suggestion-list">
            {#each refineRecommendations as item, i}
              <label class="suggestion-row refine-row" class:selected={refineSelected.has(i)}>
                <input type="checkbox" checked={refineSelected.has(i)} onchange={() => toggleRefineRecommendation(i)} />
                <span class="refine-action-badge" class:disable={item.type === 'disable'} class:add={item.type === 'add'}>
                  {recommendationLabel(item)}
                </span>
                <span class="platform-badge" class:reddit={item.query.platform === 'reddit'} class:bluesky={item.query.platform === 'bluesky'} class:google={item.query.platform === 'google'}>
                  {PLATFORM_LABELS[item.query.platform] || item.query.platform}
                </span>
                <div class="refine-copy">
                  <div class="suggestion-url" title={item.query.query_url}>{truncate(item.query.query_url, 50)}</div>
                  <div class="refine-meta">{recommendationMeta(item)}{item.angle ? ` · ${item.angle}` : ''}</div>
                  {#if item.reason}
                    <div class="refine-reason">{item.reason}</div>
                  {/if}
                </div>
                {#if item.query.angle}
                  <span class="angle-tag">{item.query.angle}</span>
                {/if}
              </label>
            {/each}
          </div>
          <div class="modal-actions">
            <button class="add-btn" onclick={applySelectedRefinements} disabled={applyingRefine || refineSelected.size === 0}>
              {applyingRefine ? 'Applying...' : `Apply ${refineSelected.size} Selected`}
            </button>
            <button class="cancel-btn" onclick={closeRefineModal}>Cancel</button>
          </div>
        {/if}
      </div>
    </div>
  {/if}
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

  .query-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .query-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .query-group-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 2px;
  }

  .keyword-label {
    font-size: 12px;
    font-weight: 600;
    color: #c9c9dc;
    letter-spacing: 0.01em;
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
    padding: 12px 14px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
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
</style>
