<script>
  import { onMount } from 'svelte';
  import { prompts as promptsApi } from '../lib/api.js';

  let { projectId } = $props();

  let promptList = $state([]);
  let loading = $state(false);
  let error = $state('');
  let savingId = $state(null);
  let expandedId = $state(null);

  const PROMPT_TYPES = [
    { platform: 'reddit', type: 'product', label: 'Reddit Product' },
    { platform: 'reddit', type: 'karma', label: 'Reddit Karma' },
    { platform: 'reddit', type: 'dm', label: 'Reddit DM' },
    { platform: 'bluesky', type: 'product', label: 'Bluesky Product' },
    { platform: 'bluesky', type: 'karma', label: 'Bluesky Karma' },
  ];

  function findPrompt(platform, type) {
    return promptList.find(p => p.platform === platform && p.type === type) || null;
  }

  async function loadPrompts() {
    loading = true;
    error = '';
    try {
      promptList = await promptsApi.list(projectId);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function createPrompt(platform, type) {
    error = '';
    try {
      const created = await promptsApi.create(projectId, {
        platform,
        type,
        prompt_text: '',
      });
      promptList = [...promptList, created];
      expandedId = created.id;
    } catch (e) {
      error = e.message;
    }
  }

  async function savePrompt(prompt, content) {
    if (savingId === prompt.id) return;
    savingId = prompt.id;
    try {
      const updated = await promptsApi.update(projectId, prompt.id, { prompt_text: content });
      promptList = promptList.map(p => p.id === prompt.id ? { ...p, ...updated } : p);
    } catch (e) {
      error = e.message;
    } finally {
      savingId = null;
    }
  }

  function toggleExpand(id) {
    expandedId = expandedId === id ? null : id;
  }

  onMount(loadPrompts);
</script>

<div class="prompt-editor">
  <div class="section-header">
    <h3 class="section-title">
      Prompts
      <a class="doc-link" href="https://docs.threadlens.dev/user-guide/prompts/" target="_blank" rel="noopener" title="How prompts work in marketing mode">?</a>
    </h3>
    <span class="section-sub">Click a header to expand and edit</span>
  </div>

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#if loading}
    <div class="loading">Loading prompts...</div>
  {:else}
    <div class="prompt-groups">
      {#each PROMPT_TYPES as pt (pt.platform + '_' + pt.type)}
        {@const prompt = findPrompt(pt.platform, pt.type)}
        <div class="prompt-item">
          {#if prompt}
            <button
              class="prompt-header"
              class:expanded={expandedId === prompt.id}
              onclick={() => toggleExpand(prompt.id)}
            >
              <span class="platform-badge" class:reddit={pt.platform === 'reddit'} class:bluesky={pt.platform === 'bluesky'}>
                {pt.platform === 'reddit' ? 'Reddit' : 'Bluesky'}
              </span>
              <span class="prompt-label">{pt.label}</span>
              {#if savingId === prompt.id}
                <span class="saving-indicator">Saving...</span>
              {/if}
              <span class="chevron" class:open={expandedId === prompt.id}>&#8250;</span>
            </button>
            {#if expandedId === prompt.id}
              <div class="prompt-body">
                <textarea
                  class="prompt-textarea"
                  value={prompt.prompt_text || ''}
                  rows="8"
                  placeholder="Enter prompt content..."
                  onblur={(e) => savePrompt(prompt, e.target.value)}
                ></textarea>
                <div class="prompt-hint">Changes are saved automatically when you click away.</div>
              </div>
            {/if}
          {:else}
            <div class="prompt-missing">
              <span class="platform-badge" class:reddit={pt.platform === 'reddit'} class:bluesky={pt.platform === 'bluesky'}>
                {pt.platform === 'reddit' ? 'Reddit' : 'Bluesky'}
              </span>
              <span class="prompt-label missing">{pt.label}</span>
              <span class="not-set">Not set</span>
              <button class="create-btn" onclick={() => createPrompt(pt.platform, pt.type)}>
                Add {pt.label} prompt
              </button>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .prompt-editor {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-header {
    display: flex;
    align-items: baseline;
    gap: 12px;
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .section-sub {
    font-size: 12px;
    color: #555;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .loading {
    color: #666;
    font-size: 14px;
    text-align: center;
    padding: 20px 0;
  }

  .prompt-groups {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .prompt-item {
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    overflow: hidden;
  }

  .prompt-header {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 14px;
    background: #1a1a24;
    border: none;
    cursor: pointer;
    text-align: left;
    color: #e2e2e8;
    transition: background 0.15s;
  }

  .prompt-header:hover {
    background: #21213a;
  }

  .prompt-header.expanded {
    background: #21213a;
    border-bottom: 1px solid #2a2a3a;
  }

  .platform-badge {
    flex-shrink: 0;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .platform-badge.reddit {
    background: #ff4500;
    color: #fff;
  }

  .platform-badge.bluesky {
    background: #0085ff;
    color: #fff;
  }

  .prompt-label {
    flex: 1;
    font-size: 14px;
    font-weight: 500;
  }

  .prompt-label.missing {
    color: #888;
  }

  .saving-indicator {
    font-size: 12px;
    color: #7c6af5;
  }

  .chevron {
    font-size: 18px;
    color: #666;
    transition: transform 0.2s;
    line-height: 1;
  }

  .chevron.open {
    transform: rotate(90deg);
  }

  .prompt-body {
    background: #0f0f13;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .prompt-textarea {
    width: 100%;
    padding: 10px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
    font-family: monospace;
    resize: vertical;
    line-height: 1.5;
  }

  .prompt-textarea:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .prompt-textarea::placeholder {
    color: #555;
  }

  .prompt-hint {
    font-size: 11px;
    color: #555;
  }

  .prompt-missing {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 14px;
    background: #1a1a24;
  }

  .not-set {
    font-size: 12px;
    color: #555;
    font-style: italic;
  }

  .create-btn {
    margin-left: auto;
    padding: 5px 12px;
    background: none;
    border: 1px solid #7c6af5;
    border-radius: 5px;
    color: #7c6af5;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .create-btn:hover {
    background: #2a2a45;
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
