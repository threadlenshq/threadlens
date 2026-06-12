<script>
  import { onMount } from 'svelte';
  import { telemetry as telemetryApi } from '../lib/api.js';
  import { updateTelemetryStatus } from '../lib/telemetry.js';

  let { onDismiss = () => {} } = $props();

  let visible = $state(false);
  let loading = $state(false);

  onMount(async () => {
    try {
      const status = await telemetryApi.status();
      if (
        status.env_opt_in === true &&
        status.ui_consent === 'unset' &&
        !status.popup_dismissed_at
      ) {
        visible = true;
      }
    } catch {
      // If we can't fetch status, don't show the toast.
    }
  });

  async function handleChoice(choice) {
    loading = true;
    try {
      await telemetryApi.consent(choice);
      updateTelemetryStatus({ ui_consent: choice });
      await telemetryApi.popupDismissed();
      updateTelemetryStatus({ popup_dismissed_at: new Date().toISOString() });
      visible = false;
      onDismiss();
    } catch (e) {
      console.error('Failed to save telemetry consent:', e);
    } finally {
      loading = false;
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape' && visible) {
      handleChoice('declined');
    }
  }

  onMount(() => {
    if (typeof window !== 'undefined') {
      window.addEventListener('keydown', handleKeydown);
      return () => window.removeEventListener('keydown', handleKeydown);
    }
  });
</script>

{#if visible}
  <div class="telemetry-toast" data-testid="telemetry-consent-toast" role="dialog" aria-label="Telemetry consent">
    <p class="telemetry-toast-text">
      <strong>Help improve ThreadLens</strong><br />
      Share anonymous usage events (feature usage, errors, version) with the ThreadLens team.
      No personal data, content, or credentials are ever sent.
    </p>
    <div class="telemetry-toast-actions">
      <button
        class="telemetry-btn accept"
        disabled={loading}
        onclick={() => handleChoice('granted')}
        data-testid="telemetry-accept"
      >Accept</button>
      <button
        class="telemetry-btn decline"
        disabled={loading}
        onclick={() => handleChoice('declined')}
        data-testid="telemetry-decline"
      >Decline</button>
      <a
        href="https://docs.threadlens.dev/reference/telemetry/"
        target="_blank"
        rel="noopener"
        class="telemetry-btn learn-more"
        data-testid="telemetry-learn-more"
      >Learn More</a>
    </div>
  </div>
{/if}

<style>
  .telemetry-toast {
    position: fixed;
    bottom: 1.5rem;
    left: 1.5rem;
    max-width: 380px;
    background: var(--surface-bg, #1e1e2e);
    border: 1px solid var(--border-color, #333);
    border-radius: 8px;
    padding: 1rem 1.25rem;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
    z-index: 9999;
    font-size: 0.875rem;
    line-height: 1.5;
    color: var(--text-primary, #e0e0e0);
  }
  .telemetry-toast-text {
    margin-bottom: 0.75rem;
  }
  .telemetry-toast-actions {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  .telemetry-btn {
    padding: 0.375rem 0.75rem;
    border-radius: 4px;
    border: 1px solid var(--border-color, #444);
    background: transparent;
    color: var(--text-primary, #e0e0e0);
    cursor: pointer;
    font-size: 0.8125rem;
    text-decoration: none;
  }
  .telemetry-btn.accept {
    background: var(--accent-bg, #185a54);
    border-color: var(--accent-bg, #185a54);
    color: #fff;
  }
  .telemetry-btn:hover {
    opacity: 0.85;
  }
  .telemetry-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
