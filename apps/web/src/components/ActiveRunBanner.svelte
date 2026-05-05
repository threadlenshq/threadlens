<script>
  import { onMount, onDestroy } from 'svelte';
  import { slide } from 'svelte/transition';

  let {
    runs = [],
    failedRuns = [],
    completedRuns = [],
    onCancel,
    onDismissFailed,
    onDismissCompleted,
  } = $props();

  let now = $state(Date.now());
  let ticker = $state(null);
  let confirmingCancelId = $state(null);

  function handleCancel(runId) {
    if (confirmingCancelId === runId) {
      confirmingCancelId = null;
      onCancel?.(runId);
    } else {
      confirmingCancelId = runId;
    }
  }

  function formatElapsed(startedAt) {
    const seconds = Math.floor((now - new Date(startedAt + 'Z').getTime()) / 1000);
    if (seconds < 0) return '0s';
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
    return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
  }

  function capitalize(s) {
    return s.charAt(0).toUpperCase() + s.slice(1);
  }

  onMount(() => {
    ticker = setInterval(() => { now = Date.now(); }, 1000);
  });

  onDestroy(() => {
    clearInterval(ticker);
  });
</script>

{#if runs.length > 0 || failedRuns.length > 0 || completedRuns.length > 0}
  <div class="banner" transition:slide={{ duration: 200 }}>
    {#each runs as run (run.id)}
      <div class="run-row">
        <span class="spinner"></span>
        <span class="platform">{capitalize(run.platform)}</span>
        <span class="step">{run.step || 'Starting...'}</span>
        <span class="elapsed">{formatElapsed(run.started_at)}</span>
        <button
          class="cancel-btn"
          class:confirming={confirmingCancelId === run.id}
          onclick={() => handleCancel(run.id)}
          onblur={() => { if (confirmingCancelId === run.id) confirmingCancelId = null; }}
          title={confirmingCancelId === run.id ? 'Click again to confirm' : 'Cancel run'}
        >
          {confirmingCancelId === run.id ? 'Cancel?' : '\u00d7'}
        </button>
      </div>
    {/each}
    {#each completedRuns as run (run.id)}
      <div class="run-row completed">
        <span class="success-icon">&#10003;</span>
        <span class="platform completed">{capitalize(run.platform)}</span>
        <span class="step completed">
          Found {run.posts_found} post{run.posts_found !== 1 ? 's' : ''} from {run.posts_checked} checked{#if run.warnings} &middot; {run.warnings}{/if}
        </span>
        <button class="dismiss" onclick={() => onDismissCompleted?.(run.id)} title="Dismiss">&times;</button>
      </div>
    {/each}
    {#each failedRuns as run (run.id)}
      <div class="run-row failed">
        <span class="fail-icon">&#10007;</span>
        <span class="platform failed">{capitalize(run.platform)}</span>
        <span class="step failed">{run.error || 'Run failed'}</span>
        <button class="dismiss" onclick={() => onDismissFailed?.(run.id)} title="Dismiss">&times;</button>
      </div>
    {/each}
  </div>
{/if}

<style>
  .banner {
    display: flex;
    flex-direction: column;
    gap: 0;
    background: #1e1a2e;
    border-bottom: 1px solid #2a2a3a;
    flex-shrink: 0;
  }

  .run-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 20px;
    border-left: 3px solid #7c6af5;
  }

  .run-row + .run-row {
    border-top: 1px solid #2a2a3a;
  }

  .platform {
    font-size: 13px;
    font-weight: 600;
    color: #7c6af5;
    min-width: 60px;
  }

  .step {
    font-size: 13px;
    color: #c0c0d0;
    flex: 1;
  }

  .elapsed {
    font-size: 12px;
    color: #666;
    font-variant-numeric: tabular-nums;
  }

  .run-row.completed {
    border-left-color: #4ade80;
  }

  .success-icon {
    color: #4ade80;
    font-size: 14px;
    font-weight: 700;
    flex-shrink: 0;
    width: 12px;
    text-align: center;
  }

  .platform.completed {
    color: #4ade80;
  }

  .step.completed {
    color: #a0d8b0;
  }

  .run-row.failed {
    border-left-color: #f87171;
  }

  .fail-icon {
    color: #f87171;
    font-size: 14px;
    font-weight: 700;
    flex-shrink: 0;
    width: 12px;
    text-align: center;
  }

  .platform.failed {
    color: #f87171;
  }

  .step.failed {
    color: #f87171;
    opacity: 0.85;
  }

  .dismiss {
    background: none;
    border: none;
    color: #666;
    font-size: 18px;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
  }

  .dismiss:hover {
    color: #f87171;
  }

  .run-row.completed .dismiss:hover {
    color: #4ade80;
  }

  .cancel-btn {
    background: none;
    border: 1px solid transparent;
    color: #666;
    font-size: 16px;
    cursor: pointer;
    padding: 0 6px;
    line-height: 1;
    border-radius: 4px;
    transition: all 0.15s;
  }

  .cancel-btn:hover {
    color: #f87171;
  }

  .cancel-btn.confirming {
    font-size: 11px;
    color: #f87171;
    border-color: #f87171;
    padding: 2px 8px;
  }

  .spinner {
    width: 12px;
    height: 12px;
    border: 2px solid rgba(124, 106, 245, 0.3);
    border-top-color: #7c6af5;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
    flex-shrink: 0;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
