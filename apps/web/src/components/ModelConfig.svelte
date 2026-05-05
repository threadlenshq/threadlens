<script>
  import { onMount, onDestroy } from 'svelte';
  import { isOverkill, isUnderkill } from '../lib/catalog.js';
  import { models as modelsApi } from '../lib/api.js';

  const KNOWN_TIERS = new Set(['low', 'medium', 'high', 'reasoning']);

  let catalogModels = [];
  let tasks = [];
  let config = {};
  let loading = false;
  let error = '';
  let savingTaskId = null;
  let savedTaskId = null;
  let savedTimer = null;
  let initialized = false;

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
      config = currentConfig || {};
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
  <div class="section-header">
    <h3 class="section-title">Models</h3>
    <span class="section-sub">Choose a model per task. Settings are global.</span>
  </div>

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#if loading}
    <div class="loading">Loading models...</div>
  {:else}
    <div class="task-list">
      {#each tasks as task (task.id)}
        {@const current = config[task.id] || { modelId: task.default, source: 'default' }}
        {@const mismatch = getMismatch(task, current.modelId)}
        {@const selectedModel = getModel(current.modelId)}
        <div class="task-row">
          <div class="task-meta">
            <div class="task-label">{task.label}</div>
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
                    <option value={model.id}>{model.label} — {model.cost}</option>
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
      {/each}
    </div>
  {/if}
</div>

<style>
  .model-config {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-header {
    display: flex;
    align-items: baseline;
    gap: 12px;
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .section-sub {
    font-size: 12px;
    color: #666;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .loading {
    color: #666;
    font-size: 14px;
    padding: 10px 0;
  }

  .task-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .task-row {
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    background: #151520;
    padding: 12px 14px;
    display: flex;
    gap: 14px;
    align-items: flex-start;
    justify-content: space-between;
  }

  .task-meta {
    flex: 1;
    min-width: 0;
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
