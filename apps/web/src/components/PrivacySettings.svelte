<script>
  import { onMount } from 'svelte';
  import { telemetry as telemetryApi } from '../lib/api.js';
  import { getTelemetryStatus, updateTelemetryStatus, onTelemetryChange } from '../lib/telemetry.js';

  let status = $state(null);
  let loading = $state(true);
  let saving = $state(false);
  let error = $state('');

  let envMode = $derived(status?.env_opt_in || 'disabled');
  let consentEnabled = $derived(envMode !== 'disabled');
  let consentForced = $derived(envMode === 'enabled');
  let consentGranted = $derived(consentForced || status?.ui_consent === 'granted');

  onMount(() => {
    fetchStatus();
    return onTelemetryChange((latest) => {
      if (latest) status = { ...status, ...latest };
    });
  });

  async function fetchStatus() {
    try {
      status = await telemetryApi.status();
    } catch (e) {
      error = e.message || 'Failed to load telemetry status.';
    } finally {
      loading = false;
    }
  }

  async function handleToggle() {
    if (!consentEnabled) return;
    saving = true;
    error = '';
    const nextChoice = consentGranted ? 'declined' : 'granted';
    try {
      await telemetryApi.consent(nextChoice);
      status = { ...status, ui_consent: nextChoice };
      updateTelemetryStatus({ ui_consent: nextChoice });
    } catch (e) {
      error = e.message || 'Failed to update consent.';
    } finally {
      saving = false;
    }
  }

  async function handleResetPrompt() {
    saving = true;
    error = '';
    try {
      await telemetryApi.resetConsent();
      status = { ...status, ui_consent: 'unset', popup_dismissed_at: '' };
      updateTelemetryStatus({ ui_consent: 'unset', popup_dismissed_at: '' });
    } catch (e) {
      error = e.message || 'Failed to reset consent prompt.';
    } finally {
      saving = false;
    }
  }
</script>

<div class="privacy-settings" data-testid="privacy-settings">
  <h2 class="privacy-title">
    Privacy
    <a class="doc-link" href="https://docs.threadlens.dev/reference/telemetry/" target="_blank" rel="noopener" title="Learn about telemetry and privacy">?</a>
  </h2>
  <p class="privacy-description">
    ThreadLens can share anonymous usage events with the ThreadLens team to help improve the product.
    No personal data, query content, source content, credentials, hostnames, IP addresses, or stack traces are ever collected.
  </p>

  {#if loading}
    <p class="privacy-loading">Loading telemetry status...</p>
  {:else if error}
    <p class="privacy-error">{error}</p>
  {:else}
    <div class="privacy-status">
      <div class="status-row">
        <span class="status-label">Environment opt-in</span>
        <span class="status-value">{envMode}</span>
      </div>
      <div class="status-row">
        <span class="status-label">UI consent</span>
        <span class="status-value">{status?.ui_consent || 'unset'}</span>
      </div>
      <div class="status-row">
        <span class="status-label">Instance ID</span>
        <span class="status-value">{status?.instance_id || '—'}</span>
      </div>
    </div>

    <div class="privacy-toggle-row">
      <label class="privacy-toggle">
        <input
          type="checkbox"
          checked={consentGranted}
          disabled={!consentEnabled || consentForced || saving}
          onchange={handleToggle}
          data-testid="privacy-consent-toggle"
        />
        <span>Share anonymous usage events with the ThreadLens team</span>
      </label>
      {#if !consentEnabled}
        <p class="privacy-disabled-hint" data-testid="privacy-disabled-hint">
          Disabled by environment configuration. Set <code>SCOUT_TELEMETRY_OPT_IN=1</code> for always-on
          or leave unset for consent mode.
        </p>
      {:else if consentForced}
        <p class="privacy-disabled-hint" data-testid="privacy-forced-hint">
          Always enabled by environment configuration (<code>SCOUT_TELEMETRY_OPT_IN=1</code>).
        </p>
      {/if}
    </div>

    <div class="privacy-details">
      <h3>What is collected</h3>
      <ul>
        <li>Feature usage signals (scout runs, report creation, schedule creation)</li>
        <li>Coarse error area signals (not error messages or stack traces)</li>
        <li>Scout version, deployment type (docker/local), OS platform</li>
        <li>A random instance ID to count instances, not people</li>
      </ul>
      <h3>What is never collected</h3>
      <ul>
        <li>Personal data, usernames, or email addresses</li>
        <li>Query text, prompt content, post content, or report content</li>
        <li>API keys, environment variable values, or file paths</li>
        <li>Hostnames, IP addresses, MAC addresses, or container IDs</li>
        <li>Error messages, stack traces, or HTTP request bodies</li>
      </ul>
      <p>
        <a href="https://docs.threadlens.dev/reference/telemetry/" target="_blank" rel="noopener">
          Read the full telemetry documentation
        </a>
      </p>
    </div>

    <div class="privacy-actions">
      <button
        class="reset-btn"
        disabled={saving}
        onclick={handleResetPrompt}
        data-testid="privacy-reset-prompt"
      >Reset consent prompt</button>
      <p class="reset-hint">This clears your consent choice and shows the consent prompt again.</p>
    </div>
  {/if}
</div>

<style>
  .privacy-settings {
    max-width: 640px;
    padding: 2rem;
  }
  .privacy-title {
    font-size: 1.5rem;
    margin-bottom: 0.5rem;
  }
  .privacy-description {
    color: var(--text-secondary, #999);
    margin-bottom: 1.5rem;
    line-height: 1.6;
  }
  .privacy-loading, .privacy-error {
    color: var(--text-secondary, #999);
  }
  .privacy-error {
    color: var(--error-color, #e74c3c);
  }
  .privacy-status {
    background: var(--surface-bg-alt, #252535);
    border-radius: 6px;
    padding: 1rem;
    margin-bottom: 1.5rem;
  }
  .status-row {
    display: flex;
    justify-content: space-between;
    padding: 0.375rem 0;
  }
  .status-label {
    color: var(--text-secondary, #999);
  }
  .status-value {
    font-family: monospace;
    font-size: 0.875rem;
  }
  .privacy-toggle-row {
    margin-bottom: 1.5rem;
  }
  .privacy-toggle {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
  }
  .privacy-toggle input:disabled {
    cursor: not-allowed;
  }
  .privacy-disabled-hint {
    color: var(--text-secondary, #999);
    font-size: 0.8125rem;
    margin-top: 0.375rem;
  }
  .privacy-disabled-hint code {
    background: var(--surface-bg-alt, #252535);
    padding: 0.125rem 0.375rem;
    border-radius: 3px;
    font-size: 0.8125rem;
  }
  .privacy-details {
    margin-bottom: 1.5rem;
  }
  .privacy-details h3 {
    font-size: 1rem;
    margin: 1rem 0 0.5rem;
  }
  .privacy-details ul {
    padding-left: 1.25rem;
    line-height: 1.8;
  }
  .privacy-details a {
    color: var(--accent-color, #185a54);
  }
  .privacy-actions {
    border-top: 1px solid var(--border-color, #333);
    padding-top: 1rem;
  }
  .reset-btn {
    padding: 0.5rem 1rem;
    border-radius: 4px;
    border: 1px solid var(--border-color, #444);
    background: transparent;
    color: var(--text-primary, #e0e0e0);
    cursor: pointer;
    font-size: 0.875rem;
  }
  .reset-btn:hover {
    opacity: 0.85;
  }
  .reset-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .reset-hint {
    color: var(--text-secondary, #999);
    font-size: 0.8125rem;
    margin-top: 0.375rem;
  }

  .doc-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    font-size: 9px;
    font-weight: 700;
    color: #4a4a60;
    background: #2a2a3a;
    border-radius: 50%;
    text-decoration: none;
    margin-left: 5px;
    vertical-align: middle;
    transition: color 0.15s, background 0.15s;
    cursor: help;
  }
  .doc-link:hover {
    color: #61afef;
    background: #61afef20;
  }
</style>
