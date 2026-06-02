<script>
  import { onMount, onDestroy } from 'svelte';
  import { slide } from 'svelte/transition';
  import { parseApiTimestamp } from '../lib/format.js';

  let {
    runningJobs = [],
    completedJobs = [],
    failedJobs = [],
    onReview,
  } = $props();

  let now = $state(Date.now());
  let ticker = $state(null);

  function formatElapsed(startedAt) {
    const startMs = parseApiTimestamp(startedAt);
    if (Number.isNaN(startMs)) return '';
    const seconds = Math.floor((now - startMs) / 1000);
    if (seconds < 0) return '0s';
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
    return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
  }

  onMount(() => {
    ticker = setInterval(() => { now = Date.now(); }, 1000);
  });

  onDestroy(() => {
    clearInterval(ticker);
  });
</script>

{#if runningJobs.length > 0 || completedJobs.length > 0 || failedJobs.length > 0}
  <div class="banner" transition:slide={{ duration: 200 }}>
    {#each runningJobs as job (job.id)}
      <div class="job-row">
        <span class="spinner"></span>
        <span class="label">{job.label || 'Query Review'}</span>
        <span class="step">{job.step || 'Running...'}</span>
        <span class="elapsed">{formatElapsed(job.started_at)}</span>
      </div>
    {/each}
    {#each completedJobs as job (job.id)}
      <div class="job-row completed">
        <span class="success-icon">&#10003;</span>
        <span class="label completed">{job.label || 'Query Review'}</span>
        <span class="step completed">
          {job.step != null
            ? job.step
            : job.result_count != null
              ? `${job.result_count} result${job.result_count !== 1 ? 's' : ''}`
              : 'Complete'}
        </span>
        {#if onReview && !job.hideReview}
          <button class="review-btn" onclick={() => onReview?.(job)}>Review</button>
        {/if}
      </div>
    {/each}
    {#each failedJobs as job (job.id)}
      <div class="job-row failed">
        <span class="fail-icon">&#10007;</span>
        <span class="label failed">{job.label || 'Query Review'}</span>
        <span class="step failed">{job.error || job.step || 'Job failed'}</span>
        {#if onReview && !job.hideReview}
          <button class="review-btn" onclick={() => onReview?.(job)}>Review</button>
        {/if}
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

  .job-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 20px;
    border-left: 3px solid #7c6af5;
  }

  .job-row + .job-row {
    border-top: 1px solid #2a2a3a;
  }

  .label {
    font-size: 13px;
    font-weight: 600;
    color: #7c6af5;
    min-width: 90px;
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

  .job-row.completed {
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

  .label.completed {
    color: #4ade80;
  }

  .step.completed {
    color: #a0d8b0;
  }

  .job-row.failed {
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

  .label.failed {
    color: #f87171;
  }

  .step.failed {
    color: #f87171;
    opacity: 0.85;
  }

  .review-btn {
    background: none;
    border: 1px solid #7c6af5;
    color: #7c6af5;
    font-size: 12px;
    cursor: pointer;
    padding: 2px 10px;
    border-radius: 4px;
    transition: all 0.15s;
    flex-shrink: 0;
  }

  .review-btn:hover {
    background: #7c6af5;
    color: #fff;
  }

  .job-row.completed .review-btn {
    border-color: #4ade80;
    color: #4ade80;
  }

  .job-row.completed .review-btn:hover {
    background: #4ade80;
    color: #1e1a2e;
  }

  .job-row.failed .review-btn {
    border-color: #f87171;
    color: #f87171;
  }

  .job-row.failed .review-btn:hover {
    background: #f87171;
    color: #fff;
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
