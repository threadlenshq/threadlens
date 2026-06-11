<script>
  import { onboarding as onboardingApi, queries } from '../../lib/api.js';

  let {
    open = false,
    status,
    selectedProjectId = null,
    onStatusReload = async () => {},
    onNavigate = () => {},
    onProjectReady = async () => {},
    onClose = () => {},
    googleLocked = false,
    onGoogleLocked = () => {},
    lastScoutZeroScored = false,
  } = $props();

  let starterProjectId = $state('ai-note-taking');
  let starterProjectName = $state('AI Note Taking Research');
  let starterQuery = $state('meeting notes too time consuming');
  let starterPlatform = $state('reddit');
  let busy = $state(false);
  let error = $state('');

  let starterDescription = $state('');

  let seededQueryCount = $derived(
    status?.items?.find(i => i.id === 'starter_query')?.seededQueryCount ?? 0
  );

  // Seeding state
  let seeding = $state(false);
  let suggestions = $state([]);
  let cardEdits = $state({});
  let selected = $state(new Set());
  let seedError = $state('');
  let seedBusy = $state(false);
  let createdProjectId = $state('');
  let hasAutoSeeded = $state(false);

  // When a project is created externally and linked to the onboarding context,
  // auto-trigger the seeding flow so the user can immediately add AI-suggested queries.
  $effect(() => {
    const ctxProjectId = status?.context?.starterProjectId;
    if (ctxProjectId && ctxProjectId !== createdProjectId && !hasAutoSeeded) {
      hasAutoSeeded = true;
      createdProjectId = ctxProjectId;
      seeding = true;
      loadSuggestions();
    }
  });

  async function completeItem(item) {
    await mutateExploration({ item, state: 'completed', selectedProjectId });
  }

  async function skipItem(item) {
    await mutateExploration({ item, state: 'skipped', selectedProjectId });
  }

  async function dismissAll() {
    await mutateExploration({ dismiss: true, selectedProjectId });
  }

  async function mutateExploration(payload) {
    busy = true;
    error = '';
    try {
      await onboardingApi.exploration(payload);
      await onStatusReload();
    } catch (e) {
      error = e.message || 'Failed to update checklist.';
    } finally {
      busy = false;
    }
  }

  let starterReady = $derived(
    starterProjectId.trim().length > 0 &&
    starterProjectName.trim().length > 0 &&
    starterQuery.trim().length > 0
  );

  async function createStarterProject() {
    if (starterPlatform === 'google' && googleLocked) {
      onGoogleLocked();
      return;
    }
    busy = true;
    error = '';
    try {
      const result = await onboardingApi.starterProject({
        projectId: starterProjectId.trim(),
        projectName: starterProjectName.trim(),
        query: starterQuery.trim(),
        platform: starterPlatform,
        description: starterDescription.trim(),
      });
      createdProjectId = result.project.id;
      await onProjectReady(result.project.id);
      await onStatusReload();
      // Transition to seeding panel
      seeding = true;
      await loadSuggestions();
    } catch (e) {
      error = e.message || 'Failed to create starter project.';
    } finally {
      busy = false;
    }
  }

  async function loadSuggestions() {
    seedError = '';
    seedBusy = true;
    try {
      const refinement = starterDescription.trim() || 'starter';
      const resp = await queries.suggest(createdProjectId, { refinement });
      suggestions = (resp.suggestions || []).map((s, i) => ({ ...s, _id: i }));
      cardEdits = {};
      selected = new Set(suggestions.map((s) => s._id));
      for (const s of suggestions) {
        cardEdits[s._id] = { platform: s.platform, query_url: s.query_url, angle: s.angle };
      }
    } catch (e) {
      seedError = e.message || 'Failed to load AI suggestions.';
      suggestions = [];
      cardEdits = {};
      selected = new Set();
    } finally {
      seedBusy = false;
    }
  }

  function toggleCard(id) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selected = next;
  }

  async function createSelectedQueries() {
    seedBusy = true;
    seedError = '';
    let created = 0;
    let failed = 0;
    for (const s of suggestions) {
      if (!selected.has(s._id)) continue;
      const edit = cardEdits[s._id] || s;
      try {
        await queries.create(createdProjectId, {
          platform: edit.platform,
          query_url: edit.query_url,
          angle: edit.angle || '',
        });
        created++;
        // Uncheck successfully created queries so retry only submits failures.
        selected.delete(s._id);
      } catch (e) {
        console.error('Failed to create query', s._id, e);
        failed++;
      }
    }
    if (failed > 0) {
      seedError = `Created ${created} of ${created + failed} queries. ${failed} failed.`;
      // Keep the panel open so the user can see the error and retry/skip.
      seedBusy = false;
      await onStatusReload();
      return;
    }
    seeding = false;
    seedBusy = false;
    await onStatusReload();
  }

  async function skipSeeding() {
    seeding = false;
    await onStatusReload();
  }

  async function handleTaskGo(item) {
    if (item === 'starter_project') {
      if (starterReady) {
        await createStarterProject();
      }
      return;
    }
    if (item === 'starter_query') {
      if (createdProjectId && !seeding) {
        seeding = true;
        await loadSuggestions();
      } else if (createdProjectId && seeding && selected.size > 0) {
        await createSelectedQueries();
      } else {
        onNavigate('posts');
      }
      return;
    }
    if (item === 'reports_intro') {
      onNavigate('reports');
      return;
    }
    if (item === 'settings_intro') {
      onNavigate('models');
      return;
    }
    // first_scout, review_results
    onNavigate('posts');
  }
</script>

<!-- Backdrop -->
{#if open}
  <div class="checklist-backdrop" onclick={onClose} aria-hidden="true"></div>
{/if}

<!-- Slide-in drawer -->
<aside
  class="exploration-checklist"
  class:is-open={open}
  data-testid="exploration-checklist"
  aria-label="Workspace exploration checklist"
  aria-hidden={!open}
>
  <header>
    <div>
      <p class="eyebrow">First value path</p>
      <h2>Reach first value with one focused scout</h2>
    </div>
    <button class="close-btn" aria-label="Close checklist" onclick={onClose}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
    </button>
  </header>

  {#if status?.items?.length > 0}
    {@const completedCount = status.items.filter(i => i.state === 'completed' || i.state === 'skipped').length}
    {@const totalCount = status.items.length}
    <div class="progress-bar-container">
      <progress value={completedCount} max={totalCount} aria-label="Onboarding progress: {completedCount} of {totalCount} items complete"></progress>
      <span class="progress-label">{completedCount} of {totalCount} items complete</span>
    </div>
  {/if}

  <div class="drawer-body">
    <p class="intro-text">Follow the shortest path to first value: create one project, add one narrow query, run one scout manually, then inspect the strongest findings before expanding.</p>

    <ul class="task-list">
      {#each status.items as item}
        <li class="task-item" class:is-completed={item.state === 'completed'} class:is-skipped={item.state === 'skipped'}>
          <button 
            class="task-check" 
            aria-label="Mark done" 
            onclick={() => completeItem(item.id)}
            disabled={item.state === 'completed'}
            title="Mark as done"
          >
            {#if item.state === 'completed'}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-success, #10b981)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/></svg>
            {:else if item.state === 'skipped'}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-muted, #6b7280)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M8 12h8"/></svg>
            {:else}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-border, #4b5563)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/></svg>
            {/if}
          </button>
          
          <button class="task-label" onclick={() => handleTaskGo(item.id)} title="Do it">
            {#if item.id === 'first_scout' && item.state === 'pending'}
              {#if seededQueryCount === 0}
                Run your first scout (add a query first)
              {:else if seededQueryCount === 1}
                Run your first scout
              {:else}
                Run your first scout ({seededQueryCount} queries ready — go!)
              {/if}
            {:else if item.id === 'starter_query' && item.state === 'completed' && item.seededQueryCount > 1}
              Add a starter query ({item.seededQueryCount} queries added)
            {:else if item.id === 'starter_query' && item.state === 'completed' && item.seededQueryCount === 1}
              Add a starter query (1 query added)
            {:else}
              {item.label}
            {/if}
          </button>

          {#if item.state === 'pending'}
            <button class="task-skip" aria-label="Skip" onclick={() => skipItem(item.id)}>
              Skip
            </button>
          {/if}

          {#if item.state === 'pending'}
            <button class="task-go" onclick={() => handleTaskGo(item.id)} disabled={item.id === 'starter_project' && !starterReady}>
              {item.id === 'starter_project' ? 'Create' : item.id === 'starter_query' ? 'Add' : 'Go'}
            </button>
          {/if}
        </li>
      {/each}
    </ul>

    {#if lastScoutZeroScored}
      <div class="post-scout-guidance">
        <p>No posts scored in your last scout — try widening the angle on one of your queries.</p>
        <button class="ghost-btn" onclick={() => onNavigate('posts')}>Edit queries</button>
      </div>
    {/if}

    <div class="starter-card">
      <div class="card-header">
        <h3>First value</h3>
        <p>Create normal project data with one narrow query, then run a scout manually when you are ready.</p>
      </div>
      <div class="form-grid">
        <label>
          <span>Project name</span>
          <input bind:value={starterProjectName} placeholder="E.g., AI Note Taking" />
        </label>
        <label>
          <span>Project ID</span>
          <input bind:value={starterProjectId} placeholder="E.g., ai-note-taking" />
        </label>
        <label class="full-width">
          <span>What problems are you researching?</span>
          <textarea bind:value={starterDescription} placeholder="E.g., teams waste hours rewriting meeting notes into project plans" rows="2"></textarea>
        </label>
        <label class="full-width">
          <span>Starter query</span>
          <input bind:value={starterQuery} placeholder="E.g., meeting notes too time consuming" />
        </label>
        <label class="full-width">
          <span>Platform</span>
          <select bind:value={starterPlatform}>
            <option value="reddit">Reddit</option>
            <option value="google">{googleLocked ? 'Google 🔒' : 'Google'}</option>
            <option value="bluesky">Bluesky</option>
          </select>
        </label>
        <p class="source-hint full-width">Reddit is the lowest-friction first source. Google stays visible but requires <code>PARALLEL_API_KEY</code> in the Scout API env. Bluesky needs <code>BLUESKY_HANDLE</code> + <code>BLUESKY_APP_PASSWORD</code>.</p>
      </div>
      <button class="primary-btn" data-testid="create-starter-project" disabled={busy || !starterReady} onclick={createStarterProject}>
        Create project and first query
      </button>
    </div>

    {#if seeding}
      <div class="seeding-panel">
        <div class="seeding-header">
          <h3>Choose your starter queries</h3>
          <p>AI suggested these based on your description. Edit, uncheck, or skip.</p>
        </div>

        {#if seedBusy && suggestions.length === 0}
          <div class="seeding-loading" role="status">
            <span class="spinner"></span> Generating suggestions…
          </div>
        {:else if suggestions.length === 0 && !seedBusy}
          <div class="seeding-empty">
            <p>{seedError || "We couldn't think of any starter queries for this topic — try writing one manually."}</p>
          </div>
        {:else}
          <div class="suggestion-cards">
            {#each suggestions as s}
              <div class="suggestion-card" class:is-unchecked={!selected.has(s._id)}>
                <label class="card-check">
                  <input
                    type="checkbox"
                    checked={selected.has(s._id)}
                    onchange={() => toggleCard(s._id)}
                    aria-label="Select query: {cardEdits[s._id]?.query_url || s.query_url}"
                  />
                </label>
                <div class="card-fields">
                  <label>
                    <span>Query</span>
                    <input
                      bind:value={cardEdits[s._id].query_url}
                      placeholder="Search query or URL pattern"
                    />
                  </label>
                  <label>
                    <span>Angle</span>
                    <input
                      bind:value={cardEdits[s._id].angle}
                      placeholder="Pain-point angle"
                    />
                  </label>
                  <label>
                    <span>Platform</span>
                    <select bind:value={cardEdits[s._id].platform}>
                      <option value="reddit">Reddit</option>
                      <option value="google">Google</option>
                      <option value="bluesky">Bluesky</option>
                    </select>
                  </label>
                </div>
              </div>
            {/each}
          </div>

          {#if seedError}
            <p class="seed-error">{seedError}</p>
          {/if}

          <div class="seeding-actions">
            <button
              class="primary-btn"
              disabled={seedBusy || selected.size === 0}
              onclick={createSelectedQueries}
            >
              {seedBusy ? 'Creating…' : `Create ${selected.size} selected ${selected.size === 1 ? 'query' : 'queries'}`}
            </button>
            <button class="ghost-btn" onclick={skipSeeding} disabled={seedBusy}>
              Skip, I'll add queries manually
            </button>
          </div>
        {/if}
      </div>
    {/if}

    {#if error}<p class="checklist-error">{error}</p>{/if}

    <div class="dismiss-row">
      <button data-testid="dismiss-exploration" class="ghost-btn" onclick={dismissAll}>
        Dismiss checklist permanently
      </button>
    </div>
  </div>
</aside>

<style>
  /* Backdrop */
  .checklist-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(2px);
    z-index: 199;
  }

  /* Drawer */
  .exploration-checklist {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    width: 400px;
    max-width: 90vw;
    background: #0f0f13;
    border-left: 1px solid #23232f;
    color: #e2e2e8;
    z-index: 200;
    display: flex;
    flex-direction: column;
    transform: translateX(100%);
    transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
    box-shadow: -8px 0 32px rgba(0, 0, 0, 0.6);
  }

  .exploration-checklist.is-open {
    transform: translateX(0);
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 16px;
    padding: 24px 20px 16px;
    border-bottom: 1px solid #23232f;
    flex-shrink: 0;
    background: #14141b;
  }

  .eyebrow {
    color: #8f8faf;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    margin: 0 0 6px;
    font-weight: 600;
  }

  h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: #ffffff;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: #8f8faf;
    cursor: pointer;
    padding: 6px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s ease;
  }

  .close-btn:hover {
    color: #ffffff;
    background: #23232f;
  }

  .drawer-body {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .intro-text {
    margin: 0;
    font-size: 14px;
    color: #a0a0b8;
    line-height: 1.5;
  }

  .task-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .task-item {
    display: flex;
    align-items: center;
    gap: 12px;
    background: #181822;
    border: 1px solid #23232f;
    border-radius: 10px;
    padding: 12px 14px;
    transition: all 0.2s ease;
  }

  .task-item:hover {
    border-color: #38384f;
    background: #1b1b26;
  }

  .task-item.is-completed {
    opacity: 0.6;
    background: transparent;
    border-color: transparent;
  }

  .task-item.is-skipped {
    opacity: 0.5;
    background: transparent;
    border-style: dashed;
  }

  .task-check {
    background: transparent;
    border: none;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    color: inherit;
    flex-shrink: 0;
    border-radius: 50%;
    transition: transform 0.1s ease;
  }

  .task-check:not(:disabled):hover {
    transform: scale(1.1);
  }

  .task-check:not(:disabled):hover svg circle {
    stroke: #6366f1;
  }

  .task-label {
    flex: 1;
    text-align: left;
    background: transparent;
    border: none;
    padding: 0;
    color: #e2e2e8;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: color 0.15s ease;
  }

  .task-item:not(.is-completed):not(.is-skipped) .task-label:hover {
    color: #818cf8;
    text-decoration: underline;
    text-underline-offset: 4px;
  }

  .task-item.is-completed .task-label,
  .task-item.is-skipped .task-label {
    text-decoration: line-through;
    color: #8f8faf;
    cursor: default;
  }

  .task-skip {
    background: transparent;
    border: 1px solid transparent;
    color: #8f8faf;
    font-size: 12px;
    padding: 4px 10px;
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.15s ease;
    opacity: 0;
  }

  .task-item:hover .task-skip {
    opacity: 1;
  }

  .task-skip:hover {
    background: #23232f;
    color: #e2e2e8;
  }

  .task-go {
    background: #4f46e5;
    border: none;
    color: #fff;
    font-size: 12px;
    padding: 4px 12px;
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.15s ease;
    font-weight: 500;
    opacity: 0;
  }

  .task-item:hover .task-go {
    opacity: 1;
  }

  .task-go:hover {
    background: #4338ca;
  }

  .starter-card {
    display: flex;
    flex-direction: column;
    gap: 16px;
    background: linear-gradient(145deg, #181822 0%, #13131a 100%);
    border: 1px solid #2d2d3f;
    border-radius: 12px;
    padding: 20px;
    position: relative;
    overflow: hidden;
  }

  .starter-card::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: linear-gradient(90deg, #6366f1, #a855f7);
  }

  .card-header h3 {
    margin: 0 0 4px;
    font-size: 15px;
    font-weight: 600;
    color: #ffffff;
  }

  .card-header p {
    margin: 0;
    font-size: 13px;
    color: #a0a0b8;
    line-height: 1.4;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .form-grid label {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .form-grid .full-width {
    grid-column: 1 / -1;
  }

  .form-grid span {
    font-size: 12px;
    font-weight: 500;
    color: #a0a0b8;
  }

  input,
  select {
    background: #0b0b0e;
    border: 1px solid #2d2d3f;
    color: #e2e2e8;
    border-radius: 8px;
    padding: 10px 12px;
    font-size: 13px;
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
  }

  input:focus,
  select:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  input::placeholder {
    color: #4b5563;
  }

  .primary-btn {
    background: #4f46e5;
    color: white;
    border: none;
    border-radius: 8px;
    padding: 10px 16px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s ease;
    width: 100%;
    margin-top: 4px;
  }

  .primary-btn:hover:not(:disabled) {
    background: #4338ca;
  }

  .primary-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .checklist-error {
    color: #f87171;
    margin: 0;
    font-size: 13px;
    background: rgba(248, 113, 113, 0.1);
    padding: 10px 12px;
    border-radius: 8px;
    border: 1px solid rgba(248, 113, 113, 0.2);
  }

  .dismiss-row {
    text-align: center;
    margin-top: auto;
    padding-top: 16px;
  }

  .ghost-btn {
    background: transparent;
    border: none;
    color: #8f8faf;
    font-size: 13px;
    padding: 8px 16px;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .ghost-btn:hover {
    color: #e2e2e8;
    background: #181822;
  }

  .source-hint {
    color: #a1a1b5;
    font-size: 12px;
    line-height: 1.5;
    margin: -4px 0 0;
  }

  .source-hint code {
    font-family: monospace;
    font-size: 11px;
    background: #0b0b0e;
    border: 1px solid #2d2d3f;
    border-radius: 3px;
    padding: 1px 4px;
    color: #a0a0c8;
  }

  textarea {
    background: #0b0b0e;
    border: 1px solid #2d2d3f;
    color: #e2e2e8;
    border-radius: 8px;
    padding: 10px 12px;
    font-size: 13px;
    font-family: inherit;
    resize: vertical;
    min-height: 48px;
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
  }

  textarea:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  textarea::placeholder {
    color: #4b5563;
  }

  .seeding-panel {
    display: flex;
    flex-direction: column;
    gap: 14px;
    background: linear-gradient(145deg, #181822 0%, #13131a 100%);
    border: 1px solid #2d2d3f;
    border-radius: 12px;
    padding: 20px;
    position: relative;
    overflow: hidden;
  }

  .seeding-panel::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: linear-gradient(90deg, #a855f7, #6366f1);
  }

  .seeding-header h3 {
    margin: 0 0 4px;
    font-size: 15px;
    font-weight: 600;
    color: #ffffff;
  }

  .seeding-header p {
    margin: 0;
    font-size: 13px;
    color: #a0a0b8;
    line-height: 1.4;
  }

  .seeding-loading {
    display: flex;
    align-items: center;
    gap: 8px;
    color: #a0a0b8;
    font-size: 13px;
    padding: 16px 0;
  }

  .seeding-loading .spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid #3a3a5a;
    border-top-color: #6366f1;
    border-radius: 50%;
    animation: seed-spin 0.6s linear infinite;
  }

  @keyframes seed-spin {
    to { transform: rotate(360deg); }
  }

  .seeding-empty p {
    color: #a0a0b8;
    font-size: 13px;
    margin: 0;
    padding: 12px;
    background: #0b0b0e;
    border: 1px solid #2d2d3f;
    border-radius: 8px;
  }

  .suggestion-cards {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .suggestion-card {
    display: flex;
    gap: 10px;
    background: #0b0b0e;
    border: 1px solid #2d2d3f;
    border-radius: 8px;
    padding: 12px;
    transition: opacity 0.15s ease;
  }

  .suggestion-card.is-unchecked {
    opacity: 0.45;
  }

  .card-check {
    display: flex;
    align-items: flex-start;
    padding-top: 2px;
    flex-shrink: 0;
  }

  .card-check input[type="checkbox"] {
    width: 16px;
    height: 16px;
    accent-color: #6366f1;
    cursor: pointer;
  }

  .card-fields {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .card-fields label {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .card-fields span {
    font-size: 11px;
    font-weight: 500;
    color: #8f8faf;
  }

  .card-fields input,
  .card-fields select {
    background: #14141b;
    border: 1px solid #2d2d3f;
    color: #e2e2e8;
    border-radius: 6px;
    padding: 6px 8px;
    font-size: 12px;
  }

  .card-fields input:focus,
  .card-fields select:focus {
    outline: none;
    border-color: #6366f1;
  }

  .seeding-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .seed-error {
    color: #f87171;
    margin: 0;
    font-size: 12px;
    background: rgba(248, 113, 113, 0.1);
    padding: 8px 10px;
    border-radius: 6px;
    border: 1px solid rgba(248, 113, 113, 0.2);
  }

  .progress-bar-container {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 20px;
    border-bottom: 1px solid #23232f;
    background: #14141b;
    flex-shrink: 0;
  }

  .progress-bar-container progress {
    flex: 1;
    height: 6px;
    border-radius: 3px;
    appearance: none;
    background: #23232f;
    border: none;
  }

  .progress-bar-container progress::-webkit-progress-bar {
    background: #23232f;
    border-radius: 3px;
  }

  .progress-bar-container progress::-webkit-progress-value {
    background: linear-gradient(90deg, #6366f1, #a855f7);
    border-radius: 3px;
  }

  .progress-bar-container progress::-moz-progress-bar {
    background: linear-gradient(90deg, #6366f1, #a855f7);
    border-radius: 3px;
  }

  .progress-label {
    font-size: 12px;
    color: #8f8faf;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .post-scout-guidance {
    background: #1a1410;
    border: 1px solid #3a2a10;
    border-radius: 8px;
    padding: 12px 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .post-scout-guidance p {
    margin: 0;
    font-size: 13px;
    color: #d4a050;
    line-height: 1.4;
  }

  .post-scout-guidance .ghost-btn {
    align-self: flex-start;
    font-size: 12px;
    padding: 4px 12px;
    border: 1px solid #3a2a10;
    color: #d4a050;
  }

  .post-scout-guidance .ghost-btn:hover {
    background: #2a1a10;
    color: #f0c070;
  }
</style>
