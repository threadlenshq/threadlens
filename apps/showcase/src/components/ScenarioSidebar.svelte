<script>
  let { scenarios = [], selectedId = null, onSelect = () => {} } = $props();

  let groupedScenarios = $derived.by(() => {
    const groups = [];
    const byGroup = new Map();

    for (const scenario of scenarios) {
      if (!byGroup.has(scenario.group)) {
        const group = { name: scenario.group, scenarios: [] };
        byGroup.set(scenario.group, group);
        groups.push(group);
      }
      byGroup.get(scenario.group).scenarios.push(scenario);
    }

    return groups;
  });
</script>

<aside class="sidebar" aria-label="Showcase scenarios">
  <div class="brand-block">
    <p class="eyebrow">Dev-only</p>
    <h1>UI Showcase</h1>
    <p>Curated ThreadLens component scenarios using static local data.</p>
  </div>

  {#if scenarios.length === 0}
    <div class="developer-message">No scenarios are registered in the showcase manifest.</div>
  {:else}
    <nav class="scenario-nav" aria-label="Usage scenarios">
      {#each groupedScenarios as group}
        <section class="scenario-group">
          <h2>{group.name}</h2>
          <div class="scenario-list">
            {#each group.scenarios as scenario}
              <button
                type="button"
                class="scenario-item"
                class:is-selected={scenario.id === selectedId}
                aria-current={scenario.id === selectedId ? 'page' : undefined}
                onclick={() => onSelect(scenario.id)}
              >
                <span class="scenario-title">{scenario.title}</span>
                <span class="scenario-description">{scenario.description}</span>
              </button>
            {/each}
          </div>
        </section>
      {/each}
    </nav>
  {/if}
</aside>

<style>
  .sidebar {
    width: 320px;
    min-width: 280px;
    height: 100vh;
    overflow-y: auto;
    border-right: 1px solid var(--color-border);
    background: color-mix(in srgb, var(--color-surface) 82%, #000 18%);
    padding: 24px 16px;
  }

  .brand-block {
    padding: 0 8px 24px;
    border-bottom: 1px solid var(--color-border);
    margin-bottom: 24px;
  }

  .eyebrow {
    margin: 0 0 8px;
    color: var(--color-brand-hover);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  h1 {
    margin: 0 0 8px;
    font-size: 22px;
    line-height: 1.1;
  }

  p {
    margin: 0;
    color: var(--color-text-secondary);
    font-size: 13px;
    line-height: 1.5;
  }

  .scenario-nav,
  .scenario-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .scenario-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .scenario-group h2 {
    margin: 0 8px;
    color: var(--color-text-muted);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  .scenario-item {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px;
    border: 1px solid transparent;
    border-radius: 10px;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
  }

  .scenario-item:hover,
  .scenario-item.is-selected {
    background: var(--color-surface-elevated);
    border-color: var(--color-border-hover);
  }

  .scenario-item.is-selected {
    border-color: color-mix(in srgb, var(--color-brand) 55%, var(--color-border));
  }

  .scenario-title {
    color: var(--color-text-primary);
    font-size: 14px;
    font-weight: 650;
  }

  .scenario-description {
    color: var(--color-text-secondary);
    font-size: 12px;
    line-height: 1.45;
  }

  .developer-message {
    margin: 0 8px;
    padding: 12px;
    border: 1px dashed var(--color-warning);
    border-radius: 8px;
    color: var(--color-warning);
    font-size: 13px;
  }

  @media (max-width: 820px) {
    .sidebar {
      width: 100%;
      height: auto;
      max-height: 45vh;
      border-right: none;
      border-bottom: 1px solid var(--color-border);
    }
  }
</style>
