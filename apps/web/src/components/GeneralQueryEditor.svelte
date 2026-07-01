<script>
  import { generalQueries as generalQueriesApi } from '../lib/api.js';

  let { projectId, onQueriesChanged } = $props();

  // Hacker News is a first-class platform alongside the others.
  const ALL_PLATFORMS = ['reddit', 'bluesky', 'google', 'hackernews'];

  const PLATFORM_LABELS = {
    reddit: 'Reddit',
    bluesky: 'Bluesky',
    google: 'Google',
    hackernews: 'Hacker News',
  };

  let list = $state([]);
  let loading = $state(false);
  let error = $state('');

  let newText = $state('');
  let newAngle = $state('');
  let newPlatforms = $state(['reddit', 'bluesky', 'google', 'hackernews']);
  let adding = $state(false);
  let showAddForm = $state(false);

  async function load() {
    loading = true;
    error = '';
    try {
      list = await generalQueriesApi.list(projectId);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function togglePlatform(p) {
    newPlatforms = newPlatforms.includes(p)
      ? newPlatforms.filter((x) => x !== p)
      : [...newPlatforms, p];
  }

  async function add() {
    if (!newText.trim() || !newAngle.trim() || newPlatforms.length === 0) return;
    adding = true;
    error = '';
    try {
      const created = await generalQueriesApi.create(projectId, {
        query_text: newText.trim(),
        angle: newAngle.trim(),
        platforms: newPlatforms,
      });
      list = [...list, created];
      onQueriesChanged?.({ projectId });
      newText = '';
      newAngle = '';
      newPlatforms = ['reddit', 'bluesky', 'google', 'hackernews'];
      showAddForm = false;
    } catch (e) {
      error = e.message;
    } finally {
      adding = false;
    }
  }

  async function remove(gq) {
    if (!confirm('Delete this general query and all its platform queries?')) return;
    try {
      await generalQueriesApi.delete(projectId, gq.id);
      list = list.filter((item) => item.id !== gq.id);
      onQueriesChanged?.({ projectId });
    } catch (e) {
      error = e.message;
    }
  }

  $effect(() => {
    if (projectId) {
      showAddForm = false;
      load();
    }
  });
</script>

<div class="general-query-editor">
  <div class="section-header">
    <h3 class="section-title">General Queries</h3>
    <span class="count">{list.length} {list.length === 1 ? 'query' : 'queries'}</span>
  </div>

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  <div class="add-panel" class:open={showAddForm}>
    <div class="add-toggle">
      <button class="add-toggle-btn" type="button" aria-expanded={showAddForm} onclick={() => showAddForm = !showAddForm}>
        <span class="add-toggle-icon" aria-hidden="true">{showAddForm ? '✕' : '+'}</span>
        <span class="add-toggle-label">Add General Query</span>
      </button>
    </div>
    {#if showAddForm}
      <div class="add-form">
        <input
          class="text-input"
          type="text"
          placeholder="Query text, e.g. manual invoicing pain"
          bind:value={newText}
        />
        <input
          class="text-input"
          type="text"
          placeholder="Angle / intent"
          bind:value={newAngle}
        />
        <div class="platform-checkboxes">
          {#each ALL_PLATFORMS as p}
            <label class="platform-label">
              <input
                type="checkbox"
                checked={newPlatforms.includes(p)}
                onchange={() => togglePlatform(p)}
              />
              {PLATFORM_LABELS[p]}
            </label>
          {/each}
        </div>
        <button
          class="add-btn"
          onclick={add}
          disabled={adding || !newText.trim() || !newAngle.trim() || newPlatforms.length === 0}
        >
          {adding ? 'Adding...' : 'Add General Query'}
        </button>
      </div>
    {/if}
  </div>

  {#if loading}
    <div class="loading">Loading...</div>
  {:else if list.length > 0}
    <ul class="gq-list">
      {#each list as gq (gq.id)}
        <li class="gq-row">
          <div class="gq-main">
            <span class="gq-text">{gq.query_text}</span>
            {#if gq.angle}
              <span class="angle-tag">{gq.angle}</span>
            {/if}
            <span class="platforms-tag">{(gq.platforms || []).map((p) => PLATFORM_LABELS[p] || p).join(', ')}</span>
          </div>
          <button class="delete-btn" onclick={() => remove(gq)} title="Delete general query">&#x2715;</button>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .general-query-editor {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .count {
    font-size: 12px;
    color: #666;
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

  .add-panel {
    display: flex;
    flex-direction: column;
  }

  .add-toggle {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 0;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    transition: background 0.12s ease, border-color 0.12s ease;
  }

  .add-toggle:hover {
    background: #20202c;
    border-color: #3a3a4a;
  }

  .add-toggle-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    padding: 11px 16px;
    background: none;
    border: none;
    color: #e2e2e8;
    font-size: 13px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    cursor: pointer;
  }

  .add-panel.open .add-toggle {
    border-bottom-left-radius: 0;
    border-bottom-right-radius: 0;
    border-bottom-color: transparent;
  }

  .add-toggle-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    font-size: 14px;
    color: #888;
  }

  .add-toggle-label {
    flex: 1;
    text-align: left;
  }

  .add-form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-top: none;
    border-radius: 0 0 8px 8px;
  }

  .text-input {
    width: 100%;
    padding: 7px 10px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
  }

  .text-input::placeholder {
    color: #555;
  }

  .text-input:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .platform-checkboxes {
    display: flex;
    gap: 16px;
    flex-wrap: wrap;
  }

  .platform-label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: #c0c0d0;
    cursor: pointer;
  }

  .platform-label input[type="checkbox"] {
    accent-color: #7c6af5;
  }

  .add-btn {
    align-self: flex-start;
    padding: 7px 16px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .add-btn:hover:not(:disabled) {
    background: #6a58e3;
  }

  .add-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .gq-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .gq-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
  }

  .gq-main {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .gq-text {
    font-size: 13px;
    color: #c0c0d0;
    font-family: monospace;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .angle-tag {
    flex-shrink: 0;
    padding: 2px 8px;
    background: #2a2a45;
    border: 1px solid #7c6af5;
    border-radius: 4px;
    font-size: 11px;
    color: #a99af7;
  }

  .platforms-tag {
    flex-shrink: 0;
    padding: 2px 8px;
    background: #1e2a1e;
    border: 1px solid #3a5a3a;
    border-radius: 4px;
    font-size: 11px;
    color: #80c080;
  }

  .delete-btn {
    flex-shrink: 0;
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid #352c35;
    border-radius: 6px;
    color: #7f8696;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .delete-btn:hover {
    background: #3a1a1a;
    border-color: #f87171;
    color: #f87171;
  }
</style>
