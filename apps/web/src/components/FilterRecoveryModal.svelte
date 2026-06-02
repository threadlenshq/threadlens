<script>
  import { RESTORE_VISIBILITY_LABEL, RESTORE_AND_TRUST_LABEL } from '../lib/filterLabels.js';

  /** @type {{ trust_options?: Array<{ label: string, value: string }> } | null} */
  let { finding = null, onClose, onRecover } = $props();

  let selectedTrust = $state('');

  // Reset selection whenever the finding changes so stale choices can't carry over
  $effect(() => {
    finding; // track finding reference
    selectedTrust = '';
  });

  function handleRestoreVisibility() {
    onRecover?.({ mode: 'restore_visibility' });
  }

  function handleRestoreAndTrust() {
    if (!selectedTrust) return;
    onRecover?.({ mode: 'restore_and_trust', trust: selectedTrust });
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="modal-backdrop" onclick={onClose}>
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div class="modal" role="dialog" aria-modal="true" aria-labelledby="recovery-modal-title" onclick={(e) => e.stopPropagation()}>
    <h2 class="modal-title" id="recovery-modal-title">Restore filtered finding</h2>

    <div class="actions">
      <button class="btn-primary" onclick={handleRestoreVisibility}>
        {RESTORE_VISIBILITY_LABEL}
      </button>

      <div class="trust-section">
        <p class="trust-label">{RESTORE_AND_TRUST_LABEL}</p>

        {#if finding?.trust_options?.length}
          <ul class="trust-options">
            {#each finding.trust_options as option (option.value)}
              <li>
                <label class="trust-option">
                  <input
                    type="radio"
                    name="trust_option"
                    value={option.value}
                    bind:group={selectedTrust}
                  />
                  {option.label}
                </label>
              </li>
            {/each}
          </ul>
        {/if}

        <button
          class="btn-secondary"
          disabled={!selectedTrust}
          onclick={handleRestoreAndTrust}
        >
          {RESTORE_AND_TRUST_LABEL}
        </button>
      </div>
    </div>

    <button class="btn-close" onclick={onClose} aria-label="Close">✕</button>
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: var(--color-surface, #fff);
    border-radius: 8px;
    padding: 1.5rem;
    max-width: 480px;
    width: 100%;
    position: relative;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.15);
  }

  .modal-title {
    font-size: 1.125rem;
    font-weight: 600;
    margin: 0 0 1.25rem;
  }

  .actions {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .trust-section {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .trust-label {
    font-weight: 500;
    margin: 0;
  }

  .trust-options {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .trust-option {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
  }

  .btn-primary,
  .btn-secondary {
    padding: 0.5rem 1rem;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    font-size: 0.875rem;
    font-weight: 500;
  }

  .btn-primary {
    background: var(--color-accent, #4f46e5);
    color: #fff;
  }

  .btn-secondary {
    background: var(--color-surface-alt, #f3f4f6);
    color: var(--color-text, #111);
  }

  .btn-secondary:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .btn-close {
    position: absolute;
    top: 0.75rem;
    right: 0.75rem;
    background: none;
    border: none;
    cursor: pointer;
    font-size: 1rem;
    color: var(--color-text-muted, #6b7280);
    line-height: 1;
    padding: 0.25rem;
  }
</style>
