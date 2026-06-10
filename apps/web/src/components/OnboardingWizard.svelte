<script>
  import { onboarding as onboardingApi } from '../lib/api.js';
  import { untrack } from 'svelte';

  let { status, onStatusReload = async () => {} } = $props();
  let currentStep = $state(status?.currentRequiredStep || 'welcome');
  let aiProvider = $state(status?.context?.aiProviderPath || 'anthropic');
  let providerSecret = $state('');
  let saving = $state(false);
  let error = $state('');
  let testState = $state('idle'); // 'idle' | 'testing' | 'connected' | 'failed'
  let testError = $state('');

  const stepOrder = ['welcome', 'ai_provider', 'app_database', 'review'];
  const providerSecretKeys = { anthropic: 'ANTHROPIC_API_KEY', gemini: 'GEMINI_API_KEY' };

  const AI_PROVIDERS = [
    {
      value: 'anthropic',
      label: 'Anthropic API (Claude)',
      description: 'Uses your ANTHROPIC_API_KEY. Recommended fastest path for Docker self-hosting.',
    },
    {
      value: 'gemini',
      label: 'Google Gemini',
      description: 'Uses your GEMINI_API_KEY.',
    },
    {
      value: 'copilot',
      label: 'GitHub Copilot',
      description: 'Advanced local path for people who already have Copilot CLI authenticated inside the runtime.',
    },
    {
      value: 'claude-cli',
      label: 'Claude CLI',
      description: 'Advanced local path for people who already have Claude CLI authenticated inside the runtime.',
    },
  ];

  let providerConfigured = $derived(
    !!status?.capabilities?.providers?.find((provider) => provider.id === aiProvider && provider.configured)
  );
  let requiresSecret = $derived(['anthropic', 'gemini'].includes(aiProvider) && !providerConfigured);
  let providerDescription = $derived(AI_PROVIDERS.find((p) => p.value === aiProvider)?.description || '');
  let canContinue = $derived(
    currentStep !== 'ai_provider' || (aiProvider && (!requiresSecret || providerSecret.trim().length > 0))
  );

  function goToStep(step) {
    if (stepOrder.includes(step)) currentStep = step;
  }

  function valuesForCurrentStep() {
    const values = { AI_PROVIDER: aiProvider };
    const secretKey = providerSecretKeys[aiProvider];
    if (secretKey && providerSecret.trim()) values[secretKey] = providerSecret.trim();
    return values;
  }

  async function saveCurrentStep() {
    if (!canContinue || saving) return;
    saving = true;
    error = '';
    try {
      const next = await onboardingApi.requiredStep({ step: currentStep, values: valuesForCurrentStep() });
      providerSecret = '';
      currentStep = next.currentRequiredStep || stepOrder[Math.min(stepOrder.indexOf(currentStep) + 1, stepOrder.length - 1)];
      await onStatusReload();
    } catch (e) {
      error = e.message || 'Failed to save this setup step.';
    } finally {
      saving = false;
    }
  }

  async function saveFinalSetup() {
    if (!canContinue || saving) return;
    saving = true;
    error = '';
    try {
      await onboardingApi.save({ values: valuesForCurrentStep() });
      providerSecret = '';
      document.dispatchEvent(new CustomEvent('onboarding-complete'));
      await onStatusReload();
    } catch (e) {
      error = e.message || 'Failed to finish setup.';
    } finally {
      saving = false;
    }
  }

  async function testConnection() {
    if (testState === 'testing') return;
    testState = 'testing';
    testError = '';
    try {
      const key = providerSecret.trim();
      const result = await onboardingApi.testAI({ provider: aiProvider, key });
      if (result.connected) {
        testState = 'connected';
      } else {
        testState = 'failed';
        testError = result.error || 'Connection failed.';
      }
    } catch (e) {
      testState = 'failed';
      testError = e.message || 'Test request failed.';
    }
  }

  // Clear test result when the secret or provider changes.
  $effect(() => {
    providerSecret;
    aiProvider;
    untrack(() => {
      testState = 'idle';
      testError = '';
    });
  });
</script>

{#if !status?.completed}
<div data-testid="onboarding-wizard" class="onboarding-wizard">
  <div class="wizard-header">
    <h2 class="wizard-title">Welcome to ThreadLens</h2>
    <p class="wizard-sub">Required setup — complete the steps below before entering your workspace.</p>
  </div>

  <!-- Step indicators -->
  <nav class="step-nav" aria-label="Setup steps">
    {#each stepOrder as step, i}
      <div
        class="step-dot"
        class:step-active={currentStep === step}
        class:step-done={status?.steps?.find(s => s.id === step)?.completed}
        aria-label={step.replace('_', ' ')}
      ></div>
    {/each}
  </nav>

  <!-- Welcome step -->
  {#if currentStep === 'welcome'}
    <section data-testid="welcome-section" class="wizard-section">
      <h3 class="step-title">Welcome</h3>
      <p class="step-body">ThreadLens is your self-hosted workspace for finding product opportunities in real conversations. This setup takes you through the fastest useful path: choose an AI provider, confirm local app readiness, create one focused research project, add one narrow query, and run your first scout manually.</p>
    </section>
    <div class="wizard-actions">
      <button
        data-testid="required-next"
        class="primary-btn"
        disabled={saving || !canContinue}
        onclick={saveCurrentStep}
      >Continue</button>
    </div>
  {/if}

  <!-- AI Provider step -->
  {#if currentStep === 'ai_provider'}
    <section data-testid="ai-provider-section" class="wizard-section">
      <h3 class="step-title">Choose an AI Provider</h3>
      <p class="step-body">ThreadLens uses AI to score posts for pain points and generate research reports. Select the provider path you want to use.</p>
      <label class="field-label" for="ai-provider-select">AI Provider</label>
      <select
        id="ai-provider-select"
        data-testid="ai-provider-select"
        bind:value={aiProvider}
        class="field-select"
      >
        {#each AI_PROVIDERS as p}
          <option value={p.value}>{p.label}</option>
        {/each}
      </select>
      {#if providerDescription}
        <p class="field-hint provider-description">{providerDescription}</p>
      {/if}
      {#if requiresSecret}
        <label class="field-label" for="provider-secret-input">
          {providerSecretKeys[aiProvider] || 'API Key'}
        </label>
        <input
          id="provider-secret-input"
          data-testid="provider-secret-input"
          type="password"
          class="field-input"
          bind:value={providerSecret}
          placeholder="Paste your API key here"
          autocomplete="off"
        />
        <p class="field-hint">This key will be written to the configured env file and not stored in onboarding state.</p>
        <div class="test-connection-row">
          <button
            data-testid="test-connection"
            class="test-btn"
            disabled={testState === 'testing' || !providerSecret.trim()}
            onclick={testConnection}
            aria-label="Test AI provider connection"
          >
            {#if testState === 'testing'}
              <span class="spinner"></span> Testing connection…
            {:else if testState === 'connected'}
              ✓ Connected
            {:else}
              Test Connection
            {/if}
          </button>
          {#if testState === 'connected'}
            <span class="test-success">Connected to {aiProvider}</span>
          {:else if testState === 'failed'}
            <span class="test-failure">{testError} — you can still continue and try again after saving.</span>
          {/if}
        </div>
      {/if}
      {#if !requiresSecret && ['anthropic', 'gemini'].includes(aiProvider)}
        <div class="test-connection-row">
          <button
            data-testid="test-connection"
            class="test-btn"
            disabled={testState === 'testing'}
            onclick={testConnection}
            aria-label="Test AI provider connection"
          >
            {#if testState === 'testing'}
              <span class="spinner"></span> Testing connection…
            {:else if testState === 'connected'}
              ✓ Connected
            {:else}
              Test Connection
            {/if}
          </button>
          {#if testState === 'connected'}
            <span class="test-success">Connected to {aiProvider}</span>
          {:else if testState === 'failed'}
            <span class="test-failure">{testError} — you can still continue and try again after saving.</span>
          {/if}
        </div>
      {/if}
    </section>
    <div class="wizard-actions">
      <button class="ghost-btn" onclick={() => goToStep('welcome')}>Back</button>
      <button
        data-testid="required-next"
        class="primary-btn"
        disabled={saving || !canContinue}
        onclick={saveCurrentStep}
      >Continue</button>
    </div>
  {/if}

  <!-- App & Database step -->
  {#if currentStep === 'app_database'}
    <section data-testid="app-database-section" class="wizard-section">
      <h3 class="step-title">App &amp; Database Setup</h3>
      <p class="step-body">Review your local environment before finishing setup. This step is read-only — it confirms what the app can access.</p>
      <div data-testid="app-database-readiness" class="readiness-card">
        <div class="readiness-row">
          <span class="readiness-label">Database location</span>
          <span class="readiness-value">{status?.appDatabase?.databasePathLabel || 'Not set'}</span>
        </div>
        <div class="readiness-row">
          <span class="readiness-label">Runtime</span>
          <span class="readiness-value">{status?.appDatabase?.runtimeMode || '—'}</span>
        </div>
        <div class="readiness-row">
          <span class="readiness-label">Environment file</span>
          <span class="readiness-value">{status?.appDatabase?.envFileLabel || '—'}</span>
        </div>
        <div class="readiness-row">
          <span class="readiness-label">Can write to env file</span>
          <span class="readiness-value">{status?.appDatabase?.envWritable ? 'Yes' : 'No'}</span>
        </div>
      </div>
      <p class="restart-note">Environment changes may not take effect until you restart Docker or the API.</p>
    </section>
    <div class="wizard-actions">
      <button class="ghost-btn" onclick={() => goToStep('ai_provider')}>Back</button>
      <button
        data-testid="required-next"
        class="primary-btn"
        disabled={saving || !canContinue}
        onclick={saveCurrentStep}
      >Continue</button>
    </div>
  {/if}

  <!-- Review step -->
  {#if currentStep === 'review'}
    <section data-testid="review-section" class="wizard-section">
      <h3 class="step-title">Review &amp; Save</h3>
      <p class="step-body">Confirm your setup before entering ThreadLens. Once saved, the workspace will open.</p>
      <div class="review-summary">
        <div class="readiness-row">
          <span class="readiness-label">AI provider</span>
          <span class="readiness-value">{AI_PROVIDERS.find((p) => p.value === aiProvider)?.label || aiProvider}</span>
        </div>
        <div class="readiness-row">
          <span class="readiness-label">API key</span>
          <span class="readiness-value">{providerConfigured ? 'Already configured' : requiresSecret ? 'Provided' : 'Not required'}</span>
        </div>
        <div class="readiness-row">
          <span class="readiness-label">Database location</span>
          <span class="readiness-value">{status?.appDatabase?.databasePathLabel || '—'}</span>
        </div>
        <div class="readiness-row">
          <span class="readiness-label">Runtime</span>
          <span class="readiness-value">{status?.appDatabase?.runtimeMode || '—'}</span>
        </div>
        <div class="readiness-row">
          <span class="readiness-label">Environment file</span>
          <span class="readiness-value">{status?.appDatabase?.envFileLabel || '—'}</span>
        </div>
        <div class="readiness-row">
          <span class="readiness-label">Can write to env file</span>
          <span class="readiness-value">{status?.appDatabase?.envWritable ? 'Yes' : 'No'}</span>
        </div>
      </div>
    </section>
    <div class="wizard-actions">
      <button class="ghost-btn" onclick={() => goToStep('app_database')}>Back</button>
      <button
        data-testid="required-save"
        class="primary-btn"
        disabled={saving || !canContinue}
        onclick={saveFinalSetup}
      >Save setup &amp; enter ThreadLens</button>
    </div>
  {/if}

  {#if error}
    <div data-testid="onboarding-error" class="wizard-error">{error}</div>
  {/if}

  {#if saving}
    <div data-testid="onboarding-saving" class="wizard-saving">Saving...</div>
  {/if}
</div>
{/if}

<style>
  .onboarding-wizard {
    max-width: 520px;
    margin: 60px auto;
    padding: 32px;
    background: #151520;
    border: 1px solid #2a2a3a;
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .wizard-header {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .wizard-title {
    font-size: 20px;
    font-weight: 700;
    color: #e2e2e8;
    margin: 0;
  }

  .wizard-sub {
    font-size: 13px;
    color: #a8a8ba;
    margin: 0;
  }

  .step-nav {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .step-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #2a2a3a;
    border: 1px solid #3a3a5a;
    transition: background 0.2s;
  }

  .step-dot.step-active {
    background: #7c6af5;
    border-color: #7c6af5;
  }

  .step-dot.step-done {
    background: #2e7d32;
    border-color: #2e7d32;
  }

  .wizard-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .step-title {
    font-size: 15px;
    font-weight: 600;
    color: #d0d0e8;
    margin: 0;
  }

  .step-body {
    font-size: 13px;
    color: #9090aa;
    margin: 0;
    line-height: 1.5;
  }

  .field-label {
    font-size: 13px;
    font-weight: 600;
    color: #c4c4d4;
  }

  .field-select,
  .field-input {
    padding: 8px 10px;
    border-radius: 6px;
    border: 1px solid #2a2a3a;
    background: #1a1a24;
    color: #e2e2e8;
    font-size: 13px;
    width: 100%;
  }

  .field-select:focus,
  .field-input:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .field-hint {
    font-size: 12px;
    color: #8080a0;
    margin: 0;
  }

  .provider-description {
    color: #a0a0c0;
    margin-top: 2px;
  }

  .readiness-card,
  .review-summary {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .readiness-row {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    font-size: 13px;
  }

  .readiness-label {
    color: #9090aa;
    flex-shrink: 0;
  }

  .readiness-value {
    color: #d0d0e8;
    text-align: right;
    word-break: break-all;
  }

  .restart-note {
    font-size: 12px;
    color: #8080a0;
    margin: 0;
    padding: 10px;
    background: #1a1410;
    border: 1px solid #3a2a10;
    border-radius: 6px;
    line-height: 1.4;
  }

  .wizard-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }

  .primary-btn {
    padding: 9px 20px;
    border-radius: 7px;
    border: none;
    background: #7c6af5;
    color: #fff;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
  }

  .primary-btn:disabled {
    opacity: 0.45;
    cursor: default;
  }

  .primary-btn:not(:disabled):hover {
    background: #6a57e0;
  }

  .ghost-btn {
    padding: 9px 16px;
    border-radius: 7px;
    border: 1px solid #3a3a5a;
    background: transparent;
    color: #a0a0bc;
    font-size: 14px;
    cursor: pointer;
  }

  .ghost-btn:hover {
    border-color: #7c6af5;
    color: #e2e2e8;
  }

  .wizard-error {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .wizard-saving {
    font-size: 13px;
    color: #a8a8ba;
    text-align: center;
  }

  .test-connection-row {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .test-btn {
    padding: 6px 14px;
    border-radius: 6px;
    border: 1px solid #3a3a5a;
    background: transparent;
    color: #c4c4d4;
    font-size: 13px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .test-btn:disabled {
    opacity: 0.45;
    cursor: default;
  }

  .test-btn:not(:disabled):hover {
    border-color: #7c6af5;
    color: #e2e2e8;
  }

  .test-success {
    font-size: 12px;
    color: #4ade80;
  }

  .test-failure {
    font-size: 12px;
    color: #f87171;
  }

  .spinner {
    display: inline-block;
    width: 12px;
    height: 12px;
    border: 2px solid #3a3a5a;
    border-top-color: #7c6af5;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
