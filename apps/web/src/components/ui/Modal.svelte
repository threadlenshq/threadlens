<script>
  import { fly } from 'svelte/transition';

  let { open = false, title = '', onClose, children, footer } = $props();

  function handleOverlayClick(e) {
    if (e.target === e.currentTarget) onClose?.();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') onClose?.();
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div
    class="modal-overlay"
    role="dialog"
    aria-modal="true"
    aria-label={title}
    tabindex="-1"
    onclick={handleOverlayClick}
    onkeydown={handleKeydown}
  >
    <div
      class="modal-content"
      transition:fly={{ y: 12, duration: 200 }}
    >
      <div class="modal-header">
        <h2 class="modal-title">{title}</h2>
        <button class="modal-close" onclick={() => onClose?.()} aria-label="Close">&#x2715;</button>
      </div>

      <div class="modal-body">
        {@render children?.()}
      </div>

      {#if footer}
        <div class="modal-footer">
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 500;
    padding: 24px;
  }

  .modal-content {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 10px;
    width: 100%;
    max-width: 480px;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
    max-height: 90vh;
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 18px 20px 16px;
    border-bottom: 1px solid #2a2a3a;
    flex-shrink: 0;
  }

  .modal-title {
    font-size: 16px;
    font-weight: 600;
    color: #e2e2e8;
    margin: 0;
  }

  .modal-close {
    background: none;
    border: none;
    color: #666;
    font-size: 16px;
    cursor: pointer;
    padding: 4px 6px;
    border-radius: 4px;
    line-height: 1;
    transition: color 0.15s, background 0.15s;
  }

  .modal-close:hover {
    color: #e2e2e8;
    background: #2a2a3a;
  }

  .modal-body {
    padding: 20px;
    overflow-y: auto;
    flex: 1;
  }

  .modal-footer {
    padding: 14px 20px;
    border-top: 1px solid #2a2a3a;
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    flex-shrink: 0;
  }
</style>
