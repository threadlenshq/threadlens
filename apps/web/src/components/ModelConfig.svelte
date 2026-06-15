<script>
  import { onMount, onDestroy } from 'svelte';
  import { isOverkill, isUnderkill } from '@scout/shared/catalog';
  import { models as modelsApi } from '../lib/api.js';
  import Surface from './ui/Surface.svelte';
  import EmptyState from './ui/EmptyState.svelte';

  const KNOWN_TIERS = new Set(['low', 'medium', 'high', 'reasoning']);

  let catalogModels = [];
  let tasks = [];
  let config = {};
  let loading = false;
  let error = '';
  let managedProvider = null;
  let savingTaskId = null;
  let savedTaskId = null;
  let savedTimer = null;
  let initialized = false;
  let hasLoadedCatalog = false;
  let currentProvider = 'sdk';

  $: modelGroups = Object.entries(
    catalogModels.reduce((acc, model) => {
      const provider = model.provider || 'other';
      if (!acc[provider]) acc[provider] = [];
      acc[provider].push(model);
      return acc;
    }, {})
  );

  function getModel(id) {
    return catalogModels.find((model) => model.id === id) || null;
  }

  function getMismatch(task, modelId) {
    const model = getModel(modelId);
    if (!model) return null;
    if (!KNOWN_TIERS.has(model.tier) || !KNOWN_TIERS.has(task.complexity)) return null;
    if (isOverkill(model.tier, task.complexity)) return 'overkill';
    if (isUnderkill(model.tier, task.complexity)) return 'underkill';
    return null;
  }

  function clearSavedBadgeSoon(taskId) {
    savedTaskId = taskId;
    clearTimeout(savedTimer);
    savedTimer = setTimeout(() => {
      if (savedTaskId === taskId) savedTaskId = null;
    }, 1400);
  }

  async function load() {
    loading = true;
    error = '';
    try {
      const [catalog, currentConfig] = await Promise.all([
        modelsApi.catalog(),
        modelsApi.config(),
      ]);
      catalogModels = Array.isArray(catalog.models) ? catalog.models : [];
      tasks = Array.isArray(catalog.tasks) ? catalog.tasks : [];
      managedProvider = catalog.managedProvider || { available: false };
      currentProvider = catalog.currentProvider || 'sdk';
      config = currentConfig || {};
      hasLoadedCatalog = true;
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function initialize() {
    if (initialized) return;
    initialized = true;
    load();
  }

  async function setTaskModel(task, modelId) {
    savingTaskId = task.id;
    error = '';
    try {
      const updated = await modelsApi.setTask(task.id, modelId);
      config = { ...config, [task.id]: updated };
      clearSavedBadgeSoon(task.id);
    } catch (e) {
      error = e.message;
    } finally {
      savingTaskId = null;
    }
  }

  async function resetTaskModel(task) {
    savingTaskId = task.id;
    error = '';
    try {
      await modelsApi.resetTask(task.id);
      config = {
        ...config,
        [task.id]: { modelId: task.default, source: 'default' }
      };
      clearSavedBadgeSoon(task.id);
    } catch (e) {
      error = e.message;
    } finally {
      savingTaskId = null;
    }
  }

  onMount(initialize);

  $: if (!initialized && typeof window !== 'undefined') {
    initialize();
  }

  onDestroy(() => {
    clearTimeout(savedTimer);
  });
</script>

<div class="model-config">
  <div class="dashboard-header">
    <h2 class="dashboard-title">AI Models &amp; Runtimes</h2>
    <p class="dashboard-sub">Configure the inference engines used for scoring and report generation.</p>
  </div>

  {#if managedProvider !== null}
    <div class="provider-surface-wrap {managedProvider.available ? 'provider-surface-wrap--available' : 'provider-surface-wrap--local'}">
      <Surface elevation={managedProvider.available ? 'elevated' : 'base'} padding="dense" class="">
        <div class="provider-row">
          <span class="provider-label">
            {managedProvider.available
              ? 'Managed AI provider routing is available for this server session.'
              : 'Managed AI is not enabled. Self-hosted provider configuration and local fallbacks remain available.'}
          </span>
          <span class="readiness-chip {managedProvider.available ? 'readiness-chip--ready' : 'readiness-chip--missing'}">
            {managedProvider.available ? 'Ready' : 'Local'}
          </span>
        </div>
      </Surface>
    </div>
  {/if}

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#if loading}
    <div class="task-list">
      {#each Array(3) as _}
        <div class="task-skeleton"></div>
      {/each}
    </div>
  {:else if hasLoadedCatalog && tasks.length === 0}
    <EmptyState
      title="No Tasks Configured"
      description="No AI task definitions were found. Check your model catalog configuration."
      icon="settings"
    />
  {:else}
    <div class="task-list">
      {#each tasks as task (task.id)}
        {@const current = config[task.id] || { modelId: task.default, source: 'default' }}
        {@const mismatch = getMismatch(task, current.modelId)}
        {@const selectedModel = getModel(current.modelId)}
        <div class="task-surface-wrap {mismatch ? 'task-surface-wrap--mismatch' : ''}">
          <Surface elevation="elevated" padding="none" class="">
          <div class="task-row">
            <div class="task-meta">
              <div class="task-label-row">
                <span class="task-label">{task.label}</span>
                <span class="readiness-chip {mismatch ? 'readiness-chip--warn' : 'readiness-chip--ready'}">
                  {mismatch ? (mismatch === 'overkill' ? 'Overkill' : 'Underkill') : 'Matched'}
                </span>
              </div>
              <div class="task-desc">{task.description}</div>
              <div class="task-stats">{task.complexity} complexity · {task.volume}</div>
            </div>

            <div class="task-controls">
              <select
                class="model-select"
                data-task-id={task.id}
                value={current.modelId}
                disabled={savingTaskId === task.id}
                on:change={(e) => setTaskModel(task, e.target.value)}
              >
                {#each modelGroups as [provider, models]}
                  <optgroup label={provider}>
                    {#each models as model (model.id)}
                      {@const recommended = model.provider === currentProvider}
                      <option value={model.id}>{model.label} — {model.cost}{recommended ? ' ✦ recommended' : ''}</option>
                    {/each}
                  </optgroup>
                {/each}
              </select>

              <div class="task-feedback">
                {#if mismatch === 'overkill'}
                  <span class="hint-badge warn">Overkill — task is {task.complexity}, model is {selectedModel?.tier}</span>
                {:else if mismatch === 'underkill'}
                  <span class="hint-badge warn">Underkill — model may be too weak for this task</span>
                {/if}

                {#if current.source === 'user'}
                  <button
                    type="button"
                    class="reset-link"
                    disabled={savingTaskId === task.id}
                    on:click={() => resetTaskModel(task)}
                  >
                    Reset to default
                  </button>
                {/if}

                {#if savingTaskId === task.id}
                  <span class="save-state">Saving...</span>
                {:else if savedTaskId === task.id}
                  <span class="save-state saved">Saved</span>
                {/if}
              </div>
            </div>
          </div>
          </Surface>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .model-config {
    display: flex;
    flex-direction: column;
    gap: 16px;
    max-width: 900px;
    margin: 0 auto;
    padding: 24px;
  }

  .dashboard-header {
    margin-bottom: 8px;
  }

  .dashboard-title {
    font-size: 20px;
    font-weight: 700;
    color: #e2e2e8;
    margin: 0 0 6px;
  }

  .dashboard-sub {
    font-size: 13px;
    color: #888;
    margin: 0;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .provider-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .provider-label {
    font-size: 13px;
    flex: 1;
  }

  /* Colorize Surface wrapper for provider availability state */
  .provider-surface-wrap--available :global(div) {
    color: #86efac;
    background: #0f2417;
    border-color: #1d4a2e;
  }

  .provider-surface-wrap--local :global(div) {
    color: #c4b5fd;
    background: #181528;
    border-color: #2d2a4a;
  }

  .readiness-chip {
    display: inline-flex;
    align-items: center;
    font-size: 11px;
    font-weight: 600;
    border-radius: 999px;
    padding: 3px 10px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .readiness-chip--ready {
    background: #0f2a1a;
    color: #86efac;
    border: 1px solid #1d4a2e;
  }

  .readiness-chip--missing {
    background: #2a1f0a;
    color: #fbbf24;
    border: 1px solid #4a3a0a;
  }

  .readiness-chip--warn {
    background: #2f2817;
    color: #eecb6a;
    border: 1px solid #665726;
  }

  .task-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .task-skeleton {
    height: 80px;
    border-radius: 8px;
    background: linear-gradient(90deg, #1a1a24 25%, #1e1e2a 50%, #1a1a24 75%);
    background-size: 200% 100%;
    animation: shimmer 1.4s ease-in-out infinite;
    border: 1px solid #2a2a3a;
  }

  @keyframes shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
  }

  @media (prefers-reduced-motion: reduce) {
    .task-skeleton { animation: none; opacity: 0.5; }
  }

  /* Wrap Surface for task rows to allow overflow:hidden and mismatch border */
  .task-surface-wrap {
    overflow: hidden;
    border-radius: 8px;
  }

  .task-surface-wrap--mismatch :global(div.rounded-lg) {
    border-left: 3px solid #665726;
  }

  .task-row {
    padding: 14px 16px;
    display: flex;
    gap: 14px;
    align-items: flex-start;
    justify-content: space-between;
  }

  .task-meta {
    flex: 1;
    min-width: 0;
  }

  .task-label-row {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .task-label {
    font-size: 14px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .task-desc {
    margin-top: 4px;
    font-size: 12px;
    color: #a8a8ba;
    line-height: 1.4;
  }

  .task-stats {
    margin-top: 5px;
    font-size: 11px;
    color: #777;
    text-transform: capitalize;
  }

  .task-controls {
    width: 360px;
    max-width: 100%;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .model-select {
    width: 100%;
    padding: 8px 10px;
    border-radius: 6px;
    border: 1px solid #2a2a3a;
    background: #1a1a24;
    color: #e2e2e8;
    font-size: 12px;
  }

  .model-select:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .task-feedback {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .hint-badge {
    font-size: 11px;
    border-radius: 999px;
    padding: 2px 8px;
    border: 1px solid #665726;
    color: #eecb6a;
    background: #2f2817;
  }

  .reset-link {
    border: none;
    background: none;
    color: #8ea2ff;
    font-size: 11px;
    cursor: pointer;
    text-decoration: underline;
    padding: 0;
  }

  .reset-link:disabled {
    opacity: 0.6;
    cursor: default;
    text-decoration: none;
  }

  .save-state {
    font-size: 11px;
    color: #7e7e8f;
  }

  .save-state.saved {
    color: #98c379;
  }
</style>
