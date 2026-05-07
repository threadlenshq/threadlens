<script>
  import { runtimeLabel } from '../lib/capabilities.js';

  let { snapshot = null, error = '' } = $props();
</script>

<div class="runtime-banner" data-runtime-mode={snapshot?.runtimeMode || 'loading'}>
  <div class="runtime-main">
    <span class="runtime-pill">{runtimeLabel(snapshot)}</span>
    {#if snapshot?.edition}
      <span class="runtime-edition">{snapshot.edition}</span>
    {/if}
  </div>

  {#if error}
    <span class="runtime-message error">{error}</span>
  {:else if snapshot?.messages?.length}
    <span class="runtime-message {snapshot.messages[0].level}">{snapshot.messages[0].message}</span>
  {:else}
    <span class="runtime-message info">Capabilities are loaded from the server and are advisory for the UI.</span>
  {/if}
</div>

<style>
  .runtime-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 8px 16px;
    border-bottom: 1px solid #24242c;
    background: #111116;
    color: #b8b8c2;
    font-size: 12px;
  }

  .runtime-main {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .runtime-pill {
    color: #e2e2e8;
    font-weight: 600;
  }

  .runtime-edition {
    color: #8f8fa3;
  }

  .runtime-message {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .runtime-message.warning {
    color: #fbbf24;
  }

  .runtime-message.error {
    color: #f87171;
  }

  .runtime-message.info {
    color: #8f8fa3;
  }
</style>
