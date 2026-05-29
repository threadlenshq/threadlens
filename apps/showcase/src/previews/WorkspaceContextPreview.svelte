<script>
  import InsightPane from '$web/components/layout/InsightPane.svelte';
  import TopContextBar from '$web/components/layout/TopContextBar.svelte';
  import { mockActiveRuns, mockProject, mockReport, noop } from '../scenarios/mock-data.js';

  let { example } = $props();
  let selectedReport = $derived(example.id === 'insight-pane-active-report' ? mockReport : null);
</script>

{#if example.id === 'top-context-breadcrumb'}
  <div class="topbar-demo">
    <TopContextBar view="posts" projectName={mockProject.name}>
      <button type="button" onclick={noop}>Run scout</button>
      <button type="button" class="secondary" onclick={noop}>New report</button>
    </TopContextBar>
  </div>
{:else}
  <div class="insight-demo">
    <div class="fake-content">
      <span>Main findings list</span>
      <strong>{selectedReport ? 'Filtered by active report' : 'No report selected'}</strong>
    </div>
    <div class="insight-wrapper">
      <InsightPane
        project={mockProject}
        activeRuns={mockActiveRuns}
        enabledQueryCount={5}
        filterSummary="Showing: new · reddit"
        {selectedReport}
      />
    </div>
  </div>
{/if}

<style>
  .topbar-demo,
  .insight-demo {
    border: 1px solid var(--color-border);
    border-radius: 10px;
    overflow: hidden;
    background: var(--color-canvas);
  }

  .topbar-demo {
    padding: 14px 16px;
  }

  button {
    border: none;
    border-radius: 6px;
    background: var(--color-brand);
    color: white;
    padding: 7px 10px;
    cursor: default;
    font-size: 12px;
    font-weight: 650;
  }

  button.secondary {
    border: 1px solid var(--color-border);
    background: var(--color-surface-elevated);
    color: var(--color-text-primary);
  }

  .insight-demo {
    display: flex;
    min-height: 300px;
  }

  .fake-content {
    flex: 1;
    min-width: 180px;
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    color: var(--color-text-muted);
    background: var(--color-canvas);
  }

  .fake-content strong {
    color: var(--color-text-secondary);
    font-size: 13px;
  }

  .insight-wrapper :global(aside) {
    display: flex !important;
    position: static !important;
    height: auto !important;
  }
</style>
