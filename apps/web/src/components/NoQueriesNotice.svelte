<script>
  import Modal from './ui/Modal.svelte';

  let {
    open = false,
    platformLabel = null,
    onClose = () => {},
    onAddQueries = () => {},
  } = $props();

  function addQueries() {
    onAddQueries?.();
    onClose?.();
  }
</script>

<Modal {open} title={platformLabel ? `No ${platformLabel} queries yet` : 'No search queries yet'} {onClose}>
  {#snippet children()}
  <div class="no-queries-copy">
    {#if platformLabel}
      <p>
        ThreadLens needs at least one enabled <strong>{platformLabel}</strong> query to scout it.
        There are no enabled {platformLabel} queries for this project yet.
      </p>
    {:else}
      <p>
        ThreadLens needs at least one enabled query to scout. None of the available platforms have an
        enabled query for this project yet.
      </p>
    {/if}

    <ol>
      <li>Open <strong>Sources</strong> and use the <strong>+ Add Query</strong> panel.</li>
      <li>Pick the platform{#if platformLabel} (<strong>{platformLabel}</strong>){/if} and add a query.</li>
      <li>Run ThreadLens again.</li>
    </ol>
  </div>
  {/snippet}

  {#snippet footer()}
    <button class="secondary-btn" type="button" onclick={onClose}>Close</button>
    <button class="primary-btn" type="button" onclick={addQueries}>Go to Sources</button>
  {/snippet}
</Modal>

<style>
  .no-queries-copy {
    display: flex;
    flex-direction: column;
    gap: 14px;
    color: #c0c0d0;
    font-size: 14px;
    line-height: 1.55;
  }

  p {
    margin: 0;
  }

  ol {
    margin: 0;
    padding-left: 22px;
  }

  li + li {
    margin-top: 8px;
  }

  strong {
    color: #e2e2e8;
  }

  .secondary-btn,
  .primary-btn {
    border-radius: 6px;
    padding: 8px 12px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
  }

  .secondary-btn {
    border: 1px solid #3a3a4a;
    background: #23233a;
    color: #e2e2e8;
  }

  .primary-btn {
    border: none;
    background: #7c6af5;
    color: #fff;
  }
</style>
