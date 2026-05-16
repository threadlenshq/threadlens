<script>
  import { onboarding as onboardingApi } from '../../lib/api.js';

  let {
    open = false,
    status,
    selectedProjectId = null,
    onStatusReload = async () => {},
    onNavigate = () => {},
    onProjectReady = async () => {},
    onClose = () => {},
  } = $props();

  let starterProjectId = $state('ai-note-taking');
  let starterProjectName = $state('AI Note Taking Research');
  let starterQuery = $state('meeting notes too time consuming');
  let starterPlatform = $state('reddit');
  let busy = $state(false);
  let error = $state('');

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
    busy = true;
    error = '';
    try {
      const result = await onboardingApi.starterProject({
        projectId: starterProjectId.trim(),
        projectName: starterProjectName.trim(),
        query: starterQuery.trim(),
        platform: starterPlatform,
      });
      await onProjectReady(result.project.id);
      await onStatusReload();
    } catch (e) {
      error = e.message || 'Failed to create starter project.';
    } finally {
      busy = false;
    }
  }

  function navigateForItem(item) {
    if (item === 'reports_intro') onNavigate('reports');
    else if (item === 'settings_intro') onNavigate('models');
    else onNavigate('posts');
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
      <p class="eyebrow">First workspace tour</p>
      <h2>Explore ThreadLens at your pace</h2>
    </div>
    <button class="close-btn" aria-label="Close checklist" onclick={onClose}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
    </button>
  </header>

  <div class="drawer-body">
    <p class="intro-text">Follow these steps to learn the core workflow.</p>

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
          
          <button class="task-label" onclick={() => navigateForItem(item.id)} title="Show me">
            {item.label}
          </button>

          {#if item.state === 'pending'}
            <button class="task-skip" aria-label="Skip" onclick={() => skipItem(item.id)}>
              Skip
            </button>
          {/if}
        </li>
      {/each}
    </ul>

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
          <span>Starter query</span>
          <input bind:value={starterQuery} placeholder="E.g., meeting notes too time consuming" />
        </label>
        <label class="full-width">
          <span>Platform</span>
          <select bind:value={starterPlatform}>
            <option value="reddit">Reddit</option>
            <option value="google">Google</option>
            <option value="bluesky">Bluesky</option>
          </select>
        </label>
        <p class="source-hint full-width">Reddit is the lowest-friction first source. Use Google after `PARALLEL_API_KEY` is configured and Bluesky after `BLUESKY_HANDLE` plus `BLUESKY_APP_PASSWORD` are configured.</p>
      </div>
      <button class="primary-btn" data-testid="create-starter-project" disabled={busy || !starterReady} onclick={createStarterProject}>
        Create project and first query
      </button>
    </div>

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
</style>

