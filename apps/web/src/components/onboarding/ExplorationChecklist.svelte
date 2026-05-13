<script>
  import { onboarding as onboardingApi } from '../../lib/api.js';

  let {
    status,
    selectedProjectId = null,
    onStatusReload = async () => {},
    onNavigate = () => {},
    onProjectReady = async () => {},
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

<section class="exploration-checklist" data-testid="exploration-checklist">
  <header>
    <div>
      <p class="eyebrow">First workspace tour</p>
      <h2>Explore ThreadLens at your pace</h2>
    </div>
    <button data-testid="dismiss-exploration" class="ghost-btn" onclick={dismissAll}>Dismiss checklist</button>
  </header>
  <ul>
    {#each status.items as item}
      <li class:item-complete={item.state === 'completed'} class:item-skipped={item.state === 'skipped'}>
        <span>{item.label}</span>
        <span class="item-state">{item.state}</span>
        <button onclick={() => completeItem(item.id)}>Mark done</button>
        <button onclick={() => skipItem(item.id)}>Skip</button>
        <button onclick={() => navigateForItem(item.id)}>Show me</button>
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
    <button data-testid="create-starter-project" disabled={busy} onclick={createStarterProject}>Create starter project and query</button>
  </div>
  {#if error}<p class="checklist-error">{error}</p>{/if}
</section>

<style>
  .exploration-checklist { padding: 14px 20px; background: #151520; border-bottom: 1px solid #2a2a3a; color: #e2e2e8; }
  header { display: flex; justify-content: space-between; gap: 12px; align-items: flex-start; }
  .eyebrow { color: #8f8faf; font-size: 12px; text-transform: uppercase; letter-spacing: 0.06em; margin: 0 0 4px; }
  h2, h3 { margin: 0; }
  ul { display: grid; gap: 8px; list-style: none; padding: 12px 0 0; margin: 0; }
  li { display: flex; align-items: center; gap: 8px; background: #1a1a24; border: 1px solid #2a2a3a; border-radius: 8px; padding: 8px; }
  li span:first-child { flex: 1; }
  .item-state { color: #aaaac0; font-size: 12px; }
  .item-complete { border-color: #2e7d32; }
  .item-skipped { opacity: 0.7; }
  button { border: 1px solid #4a4a68; background: #23233a; color: #e2e2e8; border-radius: 6px; padding: 6px 10px; cursor: pointer; }
  .ghost-btn { background: transparent; }
  .starter-card { margin-top: 12px; display: flex; flex-wrap: wrap; gap: 8px; align-items: end; }
  label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: #aaaac0; }
  input, select { background: #101018; border: 1px solid #2a2a3a; color: #e2e2e8; border-radius: 6px; padding: 6px; }
  .checklist-error { color: #f87171; margin: 8px 0 0; }
</style>
