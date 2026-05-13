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

  async function createStarterProject() {
    busy = true;
    error = '';
    try {
      const result = await onboardingApi.starterProject({
        projectId: starterProjectId,
        projectName: starterProjectName,
        query: starterQuery,
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
    <button class="close-btn" aria-label="Close checklist" onclick={onClose}>✕</button>
  </header>

  <div class="drawer-body">
    <ul>
      {#each status.items as item}
        <li class:item-complete={item.state === 'completed'} class:item-skipped={item.state === 'skipped'}>
          <span class="item-label">{item.label}</span>
          <span class="item-state">{item.state}</span>
          <div class="item-actions">
            <button onclick={() => completeItem(item.id)}>Mark done</button>
            <button onclick={() => skipItem(item.id)}>Skip</button>
            <button onclick={() => navigateForItem(item.id)}>Show me</button>
          </div>
        </li>
      {/each}
    </ul>

    <div class="starter-card">
      <h3>Create a starter research project</h3>
      <label>Project ID <input bind:value={starterProjectId} /></label>
      <label>Project name <input bind:value={starterProjectName} /></label>
      <label>Starter query <input bind:value={starterQuery} /></label>
      <label>Platform
        <select bind:value={starterPlatform}>
          <option value="reddit">Reddit</option>
          <option value="google">Google</option>
          <option value="bluesky">Bluesky</option>
        </select>
      </label>
      <button data-testid="create-starter-project" disabled={busy} onclick={createStarterProject}>
        Create starter project and query
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
    background: rgba(0, 0, 0, 0.4);
    z-index: 199;
  }

  /* Drawer */
  .exploration-checklist {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    width: 380px;
    max-width: 92vw;
    background: #151520;
    border-left: 1px solid #2a2a3a;
    color: #e2e2e8;
    z-index: 200;
    display: flex;
    flex-direction: column;
    transform: translateX(100%);
    transition: transform 0.25s ease;
    box-shadow: -4px 0 24px rgba(0, 0, 0, 0.5);
  }

  .exploration-checklist.is-open {
    transform: translateX(0);
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 12px;
    padding: 16px 16px 12px;
    border-bottom: 1px solid #2a2a3a;
    flex-shrink: 0;
  }

  .eyebrow {
    color: #8f8faf;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin: 0 0 4px;
  }

  h2 {
    margin: 0;
    font-size: 15px;
  }

  h3 {
    margin: 0 0 10px;
    font-size: 13px;
    color: #c0c0d8;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: #8f8faf;
    font-size: 16px;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
    line-height: 1;
  }

  .close-btn:hover {
    color: #e2e2e8;
    background: #2a2a3a;
  }

  .drawer-body {
    flex: 1;
    overflow-y: auto;
    padding: 12px 16px 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  ul {
    display: grid;
    gap: 8px;
    list-style: none;
    padding: 0;
    margin: 0;
  }

  li {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 8px 10px;
  }

  .item-label {
    flex: 1;
    min-width: 0;
    font-size: 13px;
  }

  .item-state {
    color: #aaaac0;
    font-size: 11px;
  }

  .item-actions {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }

  .item-complete {
    border-color: #2e7d32;
  }

  .item-skipped {
    opacity: 0.7;
  }

  button {
    border: 1px solid #4a4a68;
    background: #23233a;
    color: #e2e2e8;
    border-radius: 6px;
    padding: 5px 9px;
    cursor: pointer;
    font-size: 12px;
  }

  button:hover {
    background: #2e2e4a;
  }

  .ghost-btn {
    background: transparent;
    border-color: transparent;
    color: #8f8faf;
    font-size: 12px;
  }

  .ghost-btn:hover {
    color: #e2e2e8;
    background: #1a1a24;
    border-color: #2a2a3a;
  }

  .starter-card {
    display: flex;
    flex-direction: column;
    gap: 8px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 12px;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
    color: #aaaac0;
  }

  input,
  select {
    background: #101018;
    border: 1px solid #2a2a3a;
    color: #e2e2e8;
    border-radius: 6px;
    padding: 6px 8px;
    font-size: 13px;
  }

  .checklist-error {
    color: #f87171;
    margin: 0;
    font-size: 13px;
  }

  .dismiss-row {
    text-align: center;
    padding-top: 4px;
  }
</style>
