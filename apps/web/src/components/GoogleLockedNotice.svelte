<script>
  import Modal from './ui/Modal.svelte';

  let {
    open = false,
    showViewExistingReports = false,
    onClose = () => {},
    onViewExistingReports = () => {},
  } = $props();

  function viewExistingReports() {
    onViewExistingReports?.();
    onClose?.();
  }
</script>

<Modal {open} title="Google scouting is locked" {onClose}>
  {#snippet children()}
  <div class="google-locked-copy">
    <p>
      Scout uses Parallel.ai Search for Google scouting. This local open-core runtime does not have
      <code>PARALLEL_API_KEY</code> configured, so new Google scout runs are unavailable.
    </p>

    <ol>
      <li>Create or copy an API key from <a href="https://platform.parallel.ai/" target="_blank" rel="noreferrer">https://platform.parallel.ai/</a>.</li>
      <li>Add <code>PARALLEL_API_KEY=&lt;your key&gt;</code> to the environment used by the Scout API, such as your .env file or shell environment.</li>
      <li>Restart the Scout API and refresh this page.</li>
    </ol>

    <p class="reassurance">Existing Google reports remain viewable because they use saved results.</p>
  </div>
  {/snippet}

  {#snippet footer()}
    <a class="external-link" href="https://platform.parallel.ai/" target="_blank" rel="noreferrer">Open Parallel.ai</a>
    {#if showViewExistingReports}
      <button class="secondary-btn" type="button" onclick={viewExistingReports}>View existing Google reports</button>
    {/if}
    <button class="primary-btn" type="button" onclick={onClose}>Close</button>
  {/snippet}
</Modal>

<style>
  .google-locked-copy {
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

  code {
    padding: 1px 5px;
    border-radius: 4px;
    background: #101018;
    color: #e2e2e8;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 12px;
  }

  a {
    color: #9f8cff;
  }

  .reassurance {
    color: #98c379;
  }

  .external-link,
  .secondary-btn,
  .primary-btn {
    border-radius: 6px;
    padding: 8px 12px;
    font-size: 13px;
    font-weight: 600;
    text-decoration: none;
    cursor: pointer;
  }

  .external-link,
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
