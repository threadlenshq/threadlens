<script>
  import Modal from './ui/Modal.svelte';

  let {
    open = false,
    showViewExistingReports = false,
    onClose,
    onViewExistingReports,
  } = $props();
</script>

<Modal {open} title="Google Search is not available" {onClose}>
  {#snippet children()}
    <p>
      Google Search scouting requires a <strong>Parallel.ai</strong> account and a valid
      <code>PARALLEL_API_KEY</code> configured on your Scout server.
    </p>
    <p>
      Sign up at <a href="https://platform.parallel.ai/" target="_blank" rel="noopener noreferrer">platform.parallel.ai</a>
      to obtain an API key, then add it to your server environment.
    </p>
    {#if showViewExistingReports}
      <p>
        Previously saved Google reports are still accessible — click
        <strong>View existing Google reports</strong> below to browse them.
      </p>
    {/if}
  {/snippet}

  {#snippet footer()}
    <a
      href="https://platform.parallel.ai/"
      target="_blank"
      rel="noopener noreferrer"
      class="btn btn-primary"
    >
      Get API key
    </a>
    {#if showViewExistingReports}
      <button class="btn btn-secondary" onclick={() => onViewExistingReports?.()}>
        View existing Google reports
      </button>
    {/if}
    <button class="btn btn-ghost" onclick={() => onClose?.()}>
      Close
    </button>
  {/snippet}
</Modal>

<style>
  p {
    margin: 0 0 12px;
    color: #b0b0c0;
    font-size: 14px;
    line-height: 1.55;
  }

  p:last-child {
    margin-bottom: 0;
  }

  code {
    background: #2a2a3a;
    border-radius: 3px;
    padding: 1px 5px;
    font-size: 13px;
    color: #e2e2e8;
  }

  a {
    color: #7b8fff;
    text-decoration: none;
  }

  a:hover {
    text-decoration: underline;
  }

  .btn {
    padding: 7px 14px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: background 0.15s, color 0.15s;
    text-decoration: none;
    display: inline-flex;
    align-items: center;
  }

  .btn-primary {
    background: #4f5bd5;
    color: #fff;
  }

  .btn-primary:hover {
    background: #3f4bc5;
  }

  .btn-secondary {
    background: #2a2a3a;
    color: #e2e2e8;
  }

  .btn-secondary:hover {
    background: #3a3a4a;
  }

  .btn-ghost {
    background: transparent;
    color: #888;
  }

  .btn-ghost:hover {
    color: #e2e2e8;
    background: #2a2a3a;
  }
</style>
