<script>
  import PreviewCard from './PreviewCard.svelte';

  let { scenario = null, previewRegistry = {} } = $props();
</script>

<section class="detail-panel" aria-live="polite">
  {#if !scenario}
    <div class="developer-message">
      <h2>No scenario selected</h2>
      <p>The showcase could not select a scenario. Check `src/scenarios/manifest.js`.</p>
    </div>
  {:else}
    <header class="detail-header">
      <p class="eyebrow">{scenario.group}</p>
      <h2>{scenario.title}</h2>
      <p class="description">{scenario.description}</p>
      <div class="when-to-use">
        <strong>When to use:</strong>
        <span>{scenario.whenToUse}</span>
      </div>
    </header>

    <div class="examples-grid">
      {#each scenario.examples as example}
        <PreviewCard {example} Preview={previewRegistry[example.preview]} />
      {/each}
    </div>
  {/if}
</section>

<style>
  .detail-panel {
    flex: 1;
    min-width: 0;
    height: 100vh;
    overflow-y: auto;
    padding: 36px;
  }

  .detail-header {
    max-width: 920px;
    margin: 0 auto 28px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .eyebrow {
    margin: 0;
    color: var(--color-brand-hover);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  h2 {
    margin: 0;
    color: var(--color-text-primary);
    font-size: 30px;
    letter-spacing: -0.03em;
  }

  .description {
    margin: 0;
    color: var(--color-text-secondary);
    font-size: 15px;
    line-height: 1.6;
  }

  .when-to-use {
    display: flex;
    gap: 8px;
    align-items: baseline;
    padding: 12px 14px;
    border: 1px solid var(--color-border);
    border-radius: 10px;
    background: rgba(124, 106, 245, 0.06);
    color: var(--color-text-secondary);
    font-size: 13px;
    line-height: 1.5;
  }

  .when-to-use strong {
    color: var(--color-text-primary);
    white-space: nowrap;
  }

  .examples-grid {
    max-width: 920px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .developer-message {
    max-width: 720px;
    margin: 40px auto;
    padding: 20px;
    border: 1px dashed var(--color-warning);
    border-radius: 12px;
    background: rgba(229, 192, 123, 0.08);
  }

  .developer-message h2,
  .developer-message p {
    margin: 0;
  }

  .developer-message p {
    margin-top: 8px;
    color: var(--color-text-secondary);
  }

  @media (max-width: 820px) {
    .detail-panel {
      height: auto;
      padding: 24px 18px;
    }

    .when-to-use {
      flex-direction: column;
      gap: 4px;
    }
  }
</style>
