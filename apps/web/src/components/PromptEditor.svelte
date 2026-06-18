<script>
  import { onMount } from 'svelte';
  import { prompts as promptsApi } from '../lib/api.js';

  let { projectId } = $props();

  let promptList = $state([]);
  let loading = $state(false);
  let error = $state('');
  let savingId = $state(null);
  let expandedId = $state(null);
  let suggestingFor = $state(null);
  let suggestError = $state('');
  let aiSuggestions = $state({});

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
    suggestError = '';
  }

  async function loadSuggestions(prompt) {
    suggestingFor = prompt.id;
    suggestError = '';
    try {
      const resp = await promptsApi.suggest(projectId, { platform: prompt.platform, type: prompt.type });
      aiSuggestions = { ...aiSuggestions, [prompt.id]: resp.suggestions || [] };
      if (resp.notice) {
        suggestError = resp.notice;
      }
      if (resp.suggestions?.length > 0) {
        try { localStorage.setItem(storageKey(prompt.platform, prompt.type), JSON.stringify({ suggestions: resp.suggestions })); } catch {}
      }
    } catch (e) {
      suggestError = e.message;
      aiSuggestions = { ...aiSuggestions, [prompt.id]: [] };
    } finally {
      suggestingFor = null;
    }
  }

  function storageKey(platform, type) {
    return `scout_prompt_suggest_${projectId}_${platform}_${type}`;
  }

  function loadCachedSuggestions() {
    for (const pt of PROMPT_TYPES) {
      const cached = localStorage.getItem(storageKey(pt.platform, pt.type));
      if (cached) {
        try {
          const data = JSON.parse(cached);
          const prompt = findPrompt(pt.platform, pt.type);
          if (prompt && data.suggestions?.length > 0) {
            aiSuggestions = { ...aiSuggestions, [prompt.id]: data.suggestions };
          }
        } catch {}
      }
    }
  }

  let reviewModal = $state(null);

  function openReviewModal(prompt, index) {
    const suggestions = aiSuggestions[prompt.id];
    reviewModal = {
      prompt,
      selectedIndex: index,
      editedText: suggestions[index]?.text || '',
      suggestions,
    };
  }

  function closeReviewModal() {
    reviewModal = null;
  }

  async function approveReview() {
    const { prompt, editedText } = reviewModal;
    promptList = promptList.map(p =>
      p.id === prompt.id ? { ...p, prompt_text: editedText } : p
    );
    await savePrompt(prompt, editedText);
    localStorage.removeItem(storageKey(prompt.platform, prompt.type));
    aiSuggestions = { ...aiSuggestions, [prompt.id]: undefined };
    reviewModal = null;
  }

  function switchReviewSuggestion(index) {
    const suggestions = reviewModal.suggestions;
    reviewModal = {
      ...reviewModal,
      selectedIndex: index,
      editedText: suggestions[index]?.text || '',
    };
  }

  function discardSuggestions(prompt) {
    localStorage.removeItem(storageKey(prompt.platform, prompt.type));
    aiSuggestions = { ...aiSuggestions, [prompt.id]: undefined };
  }

  $effect(() => {
    if (suggestingFor !== null) {
      window.__scoutPromptSuggesting = true;
      const handler = (e) => { e.preventDefault(); };
      window.addEventListener('beforeunload', handler);
      return () => {
        window.removeEventListener('beforeunload', handler);
        window.__scoutPromptSuggesting = false;
      };
    }
  });

  onMount(async () => {
    await loadPrompts();
    loadCachedSuggestions();
  });
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
                <div class="suggest-row">
                  <button
                    class="suggest-btn"
                    disabled={suggestingFor === prompt.id}
                    onclick={() => loadSuggestions(prompt)}
                  >
                    {suggestingFor === prompt.id ? 'Suggesting...' : 'Suggest with AI'}
                  </button>
                </div>
                {#if suggestError && aiSuggestions[prompt.id]?.length === 0}
                  <div class="suggest-notice">{suggestError}</div>
                {/if}
                {#if aiSuggestions[prompt.id]?.length > 0}
                  <div class="chip-row">
                    {#each aiSuggestions[prompt.id] as suggestion, i}
                      <button
                        class="suggestion-chip"
                        onclick={() => openReviewModal(prompt, i)}
                        title={suggestion.text}
                      >
                        {suggestion.label}
                      </button>
                    {/each}
                    <button
                      class="suggestion-dismiss"
                      onclick={() => discardSuggestions(prompt)}
                      title="Discard suggestions"
                    >&times;</button>
                  </div>
                {/if}
                <textarea
                  class="prompt-textarea"
                  value={prompt.prompt_text || ''}
                  rows="8"
                  placeholder="Enter prompt content..."
                  onblur={(e) => savePrompt(prompt, e.target.value)}
                ></textarea>
                <div class="prompt-hint">Changes are saved automatically when you click away.</div>

                {#if reviewModal?.prompt.id === prompt.id}
                  <div class="review-overlay" onclick={closeReviewModal} role="dialog" aria-modal="true">
                    <div class="review-modal" onclick={(e) => e.stopPropagation()}>
                      <h4 class="review-title">Review Suggestion</h4>

                      <label class="review-label">Original</label>
                      <textarea class="review-original" rows="4" readonly value={prompt.prompt_text || '(empty)'}></textarea>

                      <label class="review-label">
                        Suggestion ({reviewModal.suggestions[reviewModal.selectedIndex]?.label || ''})
                      </label>
                      <textarea
                        class="review-suggestion"
                        rows="10"
                        value={reviewModal.editedText}
                        oninput={(e) => reviewModal = { ...reviewModal, editedText: e.target.value }}
                      ></textarea>

                      <div class="review-chips">
                        {#each reviewModal.suggestions as s, i}
                          <button
                            class="suggestion-chip"
                            class:active={i === reviewModal.selectedIndex}
                            onclick={() => switchReviewSuggestion(i)}
                          >
                            {s.label}
                          </button>
                        {/each}
                      </div>

                      <div class="review-actions">
                        <button class="review-cancel" onclick={closeReviewModal}>Cancel</button>
                        <button class="review-approve" onclick={approveReview}>Approve</button>
                      </div>
                    </div>
                  </div>
                {/if}
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
    position: relative;
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

  .suggest-row {
    display: flex;
    align-items: center;
    justify-content: flex-end;
  }

  .suggest-btn {
    padding: 5px 12px;
    background: none;
    border: 1px solid #7c6af5;
    border-radius: 5px;
    color: #7c6af5;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .suggest-btn:hover:not(:disabled) {
    background: #2a2a45;
  }

  .suggest-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .suggest-notice {
    padding: 8px 10px;
    background: #3a2a1a;
    border: 1px solid #6a4a2a;
    border-radius: 6px;
    color: #f5a623;
    font-size: 12px;
  }

  .chip-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .suggestion-chip {
    padding: 4px 10px;
    background: #2a2a45;
    border: 1px solid #3a3a5a;
    border-radius: 14px;
    color: #c8c8e0;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .suggestion-chip:hover {
    background: #3a3a5a;
    border-color: #7c6af5;
    color: #e2e2e8;
  }

  .suggestion-dismiss {
    padding: 0 6px;
    background: none;
    border: none;
    color: #555;
    font-size: 14px;
    cursor: pointer;
    line-height: 1;
    transition: color 0.15s;
  }

  .suggestion-dismiss:hover {
    color: #f87171;
  }

  .review-overlay {
    position: absolute;
    inset: 0;
    background: #000000dd;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 60px;
    z-index: 10;
    border-radius: 0 0 7px 7px;
  }

  .review-modal {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 20px;
    width: 90%;
    max-width: 600px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    max-height: calc(100vh - 140px);
    overflow-y: auto;
  }

  .review-title {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .review-label {
    font-size: 11px;
    color: #777;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .review-original {
    width: 100%;
    padding: 8px;
    background: #12121a;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #777;
    font-size: 12px;
    font-family: monospace;
    resize: none;
    line-height: 1.5;
  }

  .review-suggestion {
    width: 100%;
    padding: 8px;
    background: #12121a;
    border: 1px solid #3a3a5a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
    font-family: monospace;
    resize: vertical;
    line-height: 1.5;
  }

  .review-suggestion:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .review-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .suggestion-chip.active {
    background: #3a3a5a;
    border-color: #7c6af5;
    color: #e2e2e8;
  }

  .review-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 4px;
  }

  .review-cancel {
    padding: 6px 16px;
    background: none;
    border: 1px solid #3a3a5a;
    border-radius: 5px;
    color: #888;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .review-cancel:hover {
    color: #e2e2e8;
    border-color: #555;
  }

  .review-approve {
    padding: 6px 16px;
    background: #7c6af5;
    border: none;
    border-radius: 5px;
    color: #fff;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s;
  }

  .review-approve:hover {
    background: #6a58e0;
  }
</style>
