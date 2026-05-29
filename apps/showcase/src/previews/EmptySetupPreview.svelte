<script>
  import EmptyState from '$web/components/ui/EmptyState.svelte';
  import LoadingSkeleton from '$web/components/ui/LoadingSkeleton.svelte';
  import Surface from '$web/components/ui/Surface.svelte';
  import TourCallout from '$web/components/onboarding/TourCallout.svelte';
  import { noop } from '../scenarios/mock-data.js';

  let { example } = $props();
</script>

{#if example.id === 'first-run-empty-panel'}
  <Surface padding="comfortable" elevation="base">
    <EmptyState
      title="No findings yet"
      description="Run one focused scout to collect candidate posts before expanding query coverage."
      icon="search"
      actionLabel="Run scout"
      onaction={noop}
    />
  </Surface>
{:else if example.id === 'waiting-for-results'}
  <Surface padding="comfortable" elevation="elevated">
    <div class="loading-copy">
      <p class="label">Scout run in progress</p>
      <h4>Checking Reddit and Bluesky for fresh product signals.</h4>
    </div>
    <LoadingSkeleton type="card" count={1} />
  </Surface>
{:else if example.id === 'guided-first-value'}
  <div class="callout-frame">
    <TourCallout
      title="Reach first value"
      body="Create one project, add one narrow query, run one scout, then inspect the strongest findings."
      actionLabel="Open checklist"
      onDismiss={noop}
    />
  </div>
{/if}

<style>
  .loading-copy {
    margin-bottom: 16px;
  }

  .label {
    margin: 0 0 4px;
    color: var(--color-brand-hover);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  h4 {
    margin: 0;
    color: var(--color-text-primary);
    font-size: 15px;
  }

  .callout-frame {
    max-width: 420px;
  }
</style>
