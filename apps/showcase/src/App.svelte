<script>
  import { scenarios } from './scenarios/manifest.js';
  import { previewRegistry } from './scenarios/previews.js';
  import { validateScenarios } from './scenarios/validate.js';
  import ScenarioSidebar from './components/ScenarioSidebar.svelte';
  import ScenarioDetail from './components/ScenarioDetail.svelte';

  let validationError = null;

  try {
    validateScenarios(scenarios, previewRegistry);
  } catch (err) {
    validationError = err instanceof Error ? err.message : String(err);
  }

  const safeScenarios = Array.isArray(scenarios) ? scenarios : [];

  let selectedId = $state(safeScenarios.length > 0 ? safeScenarios[0].id : null);

  let selectedScenario = $derived(safeScenarios.find((s) => s.id === selectedId) ?? null);

  function handleSelect(id) {
    selectedId = id;
  }
</script>

{#if validationError}
  <div class="dev-error" role="alert">
    <h1>Showcase manifest error</h1>
    <p>{validationError}</p>
    <p class="hint">Fix the error in <code>src/scenarios/manifest.js</code> or <code>src/scenarios/previews.js</code> and reload.</p>
  </div>
{:else}
  <div class="app-layout">
    <ScenarioSidebar scenarios={safeScenarios} selectedId={selectedId} onSelect={handleSelect} />
    <ScenarioDetail scenario={selectedScenario} {previewRegistry} />
  </div>
{/if}

<style>
  .app-layout {
    display: flex;
    min-height: 100vh;
  }

  @media (max-width: 820px) {
    .app-layout {
      flex-direction: column;
    }
  }

  .dev-error {
    max-width: 680px;
    margin: 60px auto;
    padding: 28px 32px;
    border: 1px dashed var(--color-warning);
    border-radius: 14px;
    background: rgba(229, 192, 123, 0.08);
  }

  .dev-error h1 {
    margin: 0 0 12px;
    font-size: 22px;
    color: var(--color-warning);
  }

  .dev-error p {
    margin: 0 0 8px;
    color: var(--color-text-secondary);
    font-size: 14px;
    line-height: 1.5;
  }

  .dev-error .hint {
    margin-top: 12px;
    font-size: 13px;
    color: var(--color-text-muted);
  }

  .dev-error code {
    font-family: monospace;
    background: rgba(255, 255, 255, 0.08);
    padding: 1px 5px;
    border-radius: 4px;
  }
</style>
