<script>
  import GoogleLockedNotice from '$web/components/GoogleLockedNotice.svelte';
  import Modal from '$web/components/ui/Modal.svelte';
  import { noop } from '../scenarios/mock-data.js';

  let { example } = $props();
</script>

<div class="modal-preview">
  {#if example.id === 'confirmation-modal'}
    <Modal open={true} title="Archive finding" onClose={noop}>
      <p>This finding will move out of the active review queue. You can still find it in excluded findings.</p>

      {#snippet footer()}
        <button type="button" class="secondary" onclick={noop}>Cancel</button>
        <button type="button" class="danger" onclick={noop}>Archive finding</button>
      {/snippet}
    </Modal>
  {:else if example.id === 'credential-guidance-modal'}
    <GoogleLockedNotice
      open={true}
      showViewExistingReports={true}
      onClose={noop}
      onViewExistingReports={noop}
    />
  {/if}
</div>

<style>
  .modal-preview {
    min-height: 360px;
    position: relative;
    border: 1px solid var(--color-border);
    border-radius: 12px;
    overflow: hidden;
    background: repeating-linear-gradient(135deg, #101018, #101018 12px, #12121b 12px, #12121b 24px);
  }

  .modal-preview :global(.modal-overlay) {
    position: absolute !important;
    inset: 0 !important;
    padding: 20px !important;
    background: rgba(0, 0, 0, 0.48) !important;
  }

  .modal-preview :global(.modal-content) {
    max-width: 460px !important;
  }

  p {
    margin: 0;
    color: #c0c0d0;
    font-size: 14px;
    line-height: 1.55;
  }

  button {
    border-radius: 6px;
    padding: 8px 12px;
    font-size: 13px;
    font-weight: 650;
    cursor: default;
  }

  .secondary {
    border: 1px solid #3a3a4a;
    background: #23233a;
    color: #e2e2e8;
  }

  .danger {
    border: none;
    background: #e06c75;
    color: #fff;
  }
</style>
