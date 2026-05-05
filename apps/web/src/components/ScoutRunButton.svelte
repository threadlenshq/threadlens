<script>
  import { scout as scoutApi } from '../lib/api.js';

  let {
    projectId,
    externalRunning = false,
    lastRunLabel = '',
    enabledQueryCount = null,
    onScoutComplete,
  } = $props();

  let running = $state(false);
  let toastMessage = $state('');
  let toastTimeout = $state(null);
  let showDropdown = $state(false);
  let selectedPlatform = $state('all');
  let disabled = $derived(running || externalRunning || !projectId);

  const platforms = [
    { value: 'all', label: 'All Platforms' },
    { value: 'reddit', label: 'Reddit' },
    { value: 'bluesky', label: 'Bluesky' },
    { value: 'google', label: 'Google' },
  ];

  function selectedPlatformLabel() {
    return platforms.find((p) => p.value === selectedPlatform)?.label || selectedPlatform;
  }

  function showToast(msg) {
    toastMessage = msg;
    clearTimeout(toastTimeout);
    toastTimeout = setTimeout(() => { toastMessage = ''; }, 4000);
  }

  const MIN_RECOMMENDED_QUERIES = 8;

  async function run() {
    if (disabled) return;
    if (enabledQueryCount !== null && enabledQueryCount < MIN_RECOMMENDED_QUERIES) {
      const proceed = confirm(
        `You have ${enabledQueryCount} search ${enabledQueryCount === 1 ? 'query' : 'queries'}. For best results, we recommend at least ${MIN_RECOMMENDED_QUERIES}. Run anyway?`
      );
      if (!proceed) return;
    }
    running = true;
    toastMessage = '';
    showDropdown = false;
    try {
      let results;
      if (selectedPlatform === 'all') {
        results = await Promise.all([
          scoutApi.run(projectId, 'reddit'),
          scoutApi.run(projectId, 'bluesky'),
          scoutApi.run(projectId, 'google'),
        ]);
      } else {
        results = [await scoutApi.run(projectId, selectedPlatform)];
      }
      const runIds = results.map(r => r.runId).filter(Boolean);
      onScoutComplete?.({ platform: selectedPlatform, runIds });
    } catch (e) {
      showToast(e.message);
    } finally {
      running = false;
    }
  }

  function toggleDropdown() {
    showDropdown = !showDropdown;
  }

  function selectPlatform(val) {
    selectedPlatform = val;
    showDropdown = false;
    run();
  }

  function handleOutsideClick(e) {
    if (!e.target.closest('.scout-run-btn')) {
      showDropdown = false;
    }
  }
</script>

<svelte:window onclick={handleOutsideClick} />

<div class="scout-run-btn">
  <div class="btn-group" class:disabled>
    <button
      class="run-btn"
      {disabled}
      onclick={run}
    >
      {#if running || externalRunning}
        <span class="spinner"></span>
        Running...
      {:else}
        Run Scout
        {#if selectedPlatform !== 'all'}
          <span class="platform-pill">{selectedPlatformLabel()}</span>
        {/if}
      {/if}
    </button>
    <button
      class="dropdown-toggle"
      {disabled}
      onclick={(event) => { event.stopPropagation(); toggleDropdown(); }}
      title="Select platform"
    >
      <span class="caret" class:open={showDropdown}>&#9662;</span>
    </button>
  </div>

  {#if lastRunLabel && !running && !externalRunning}
    <span class="last-run">Last run: {lastRunLabel}</span>
  {/if}

  {#if showDropdown}
    <div class="dropdown-menu">
      {#each platforms as p}
        <button
          class="dropdown-item"
          class:selected={selectedPlatform === p.value}
          onclick={() => selectPlatform(p.value)}
        >
          {p.label}
        </button>
      {/each}
    </div>
  {/if}

  {#if toastMessage}
    <div class="toast-error" onclick={() => (toastMessage = '')}>
      {toastMessage}
    </div>
  {/if}
</div>

<style>
  .scout-run-btn {
    position: relative;
    display: inline-flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 4px;
  }

  .btn-group {
    display: flex;
    align-items: stretch;
    border-radius: 6px;
    overflow: visible;
  }

  .run-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 7px 14px;
    background: #7c6af5;
    border: none;
    border-radius: 6px 0 0 6px;
    color: #fff;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
    white-space: nowrap;
  }

  .run-btn:hover:not(:disabled) {
    background: #6a58e3;
  }

  .run-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .dropdown-toggle {
    padding: 7px 8px;
    background: #6a58e3;
    border: none;
    border-left: 1px solid rgba(255,255,255,0.2);
    border-radius: 0 6px 6px 0;
    color: #fff;
    cursor: pointer;
    transition: background 0.15s;
    display: flex;
    align-items: center;
  }

  .dropdown-toggle:hover:not(:disabled) {
    background: #5a48d3;
  }

  .dropdown-toggle:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .caret {
    font-size: 16px;
    line-height: 1;
    transition: transform 0.2s;
    display: block;
  }

  .caret.open {
    transform: rotate(180deg);
  }

  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid rgba(255,255,255,0.3);
    border-top-color: #fff;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
    flex-shrink: 0;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .platform-pill {
    padding: 1px 6px;
    background: rgba(255,255,255,0.2);
    border-radius: 4px;
    font-size: 11px;
  }

  .dropdown-menu {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    overflow: hidden;
    z-index: 100;
    min-width: 150px;
    box-shadow: 0 4px 16px rgba(0,0,0,0.4);
  }

  .dropdown-item {
    display: block;
    width: 100%;
    padding: 9px 14px;
    background: none;
    border: none;
    text-align: left;
    font-size: 13px;
    color: #c0c0d0;
    cursor: pointer;
    transition: background 0.1s;
  }

  .dropdown-item:hover {
    background: #2a2a3a;
    color: #e2e2e8;
  }

  .dropdown-item.selected {
    color: #7c6af5;
    background: #2a2a45;
  }

  .toast-error {
    position: fixed;
    top: 64px;
    right: 20px;
    padding: 10px 16px;
    background: #3a1a1a;
    border: 1px solid #f87171;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
    z-index: 1000;
    cursor: pointer;
    box-shadow: 0 4px 16px rgba(0,0,0,0.5);
    animation: toast-in 0.2s ease-out;
  }

  @keyframes toast-in {
    from { opacity: 0; transform: translateY(-8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .last-run {
    position: absolute;
    right: 100%;
    top: 50%;
    transform: translateY(-50%);
    margin-right: 10px;
    font-size: 11px;
    color: #666;
    white-space: nowrap;
  }
</style>
