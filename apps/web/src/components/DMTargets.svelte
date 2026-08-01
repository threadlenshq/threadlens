<script>
  import { dmTargets } from '../lib/api.js';
  import { parseApiTimestamp } from '../lib/format.js';

  let { projectId = null, projectMode = 'marketing' } = $props();

  const TABS = [
    { id: '', label: 'All' },
    { id: 'new', label: 'New' },
    { id: 'sent', label: 'Sent' },
    { id: 'replied', label: 'Replied' },
    { id: 'ignored', label: 'Ignored' },
  ];

  let items = $state([]);
  let counts = $state({ new: 0, sent: 0, replied: 0, ignored: 0 });
  let activeTab = $state('');
  let loading = $state(false);
  let error = $state(null);
  let selectedIds = $state(new Set());
  let bulkBusy = $state(false);
  let fetchEpoch = 0;

  let total = $derived((counts.new || 0) + (counts.sent || 0) + (counts.replied || 0) + (counts.ignored || 0));
  let headerChecked = $derived(items.length > 0 && selectedIds.size === items.length);

  async function refresh() {
    if (!projectId) return;
    const epoch = ++fetchEpoch;
    loading = true;
    error = null;
    try {
      const response = await dmTargets.list(projectId, activeTab);
      if (epoch !== fetchEpoch) return; // stale
      items = Array.isArray(response?.items) ? response.items : [];
      counts = response?.counts && typeof response.counts === 'object'
        ? { new: response.counts.new || 0, sent: response.counts.sent || 0, replied: response.counts.replied || 0, ignored: response.counts.ignored || 0 }
        : { new: 0, sent: 0, replied: 0, ignored: 0 };
    } catch (e) {
      if (epoch !== fetchEpoch) return;
      error = e.message || 'Failed to load DM targets';
      items = [];
    } finally {
      if (epoch === fetchEpoch) loading = false;
    }
  }

  $effect(() => {
    // Re-fetch whenever the project changes.
    if (projectId) {
      refresh();
    }
  });

  function selectTab(tabId) {
    if (activeTab === tabId) return;
    activeTab = tabId;
    selectedIds = new Set();
  }

  function toggleRow(item, event) {
    event.stopPropagation();
    const next = new Set(selectedIds);
    if (next.has(item.id)) {
      next.delete(item.id);
    } else {
      next.add(item.id);
    }
    selectedIds = next;
  }

  function toggleHeader() {
    if (headerChecked) {
      selectedIds = new Set();
    } else {
      selectedIds = new Set(items.map((it) => it.id));
    }
  }

  async function bulkSet(status) {
    if (selectedIds.size === 0) return;
    bulkBusy = true;
    error = null;
    try {
      await dmTargets.bulkUpdate(projectId, { ids: [...selectedIds], dm_status: status });
      selectedIds = new Set();
      await refresh();
    } catch (e) {
      error = e.message || 'Bulk update failed';
    } finally {
      bulkBusy = false;
    }
  }

  function statusPillClass(status) {
    return `status-pill status-${status}`;
  }

  function formatRelative(value) {
    if (!value) return '';
    const ms = parseApiTimestamp(value);
    if (Number.isNaN(ms)) return value;
    const seconds = Math.floor((Date.now() - ms) / 1000);
    if (seconds < 60) return 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}d ago`;
    return new Date(ms).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' });
  }

  function postSubredditLabel(item) {
    if (item.post_subreddit) return `r/${item.post_subreddit}`;
    return item.post_platform || '';
  }
</script>

{#if projectMode !== 'marketing'}
  <div class="dm-empty">
    <p>DM Targets is only available in marketing-mode projects.</p>
  </div>
{:else}
  <div class="dm-view">
    <div class="dm-header">
      <h2 class="dm-title">DM Targets</h2>
      <div class="dm-total">{total} total</div>
    </div>

    <div class="dm-tabs">
      {#each TABS as tab}
        <button
          type="button"
          class="dm-tab"
          class:active={activeTab === tab.id}
          onclick={() => selectTab(tab.id)}
        >
          <span class="dm-tab-label">{tab.label}</span>
          <span class="dm-tab-count">
            {tab.id === '' ? total : (counts[tab.id] || 0)}
          </span>
        </button>
      {/each}
    </div>

    {#if selectedIds.size > 0}
      <div class="bulk-bar">
        <span class="bulk-count">{selectedIds.size} selected</span>
        <div class="bulk-actions">
          <button class="bulk-btn" disabled={bulkBusy} onclick={() => bulkSet('sent')}>Mark sent</button>
          <button class="bulk-btn" disabled={bulkBusy} onclick={() => bulkSet('replied')}>Mark replied</button>
          <button class="bulk-btn" disabled={bulkBusy} onclick={() => bulkSet('ignored')}>Ignore</button>
        </div>
        <button class="bulk-cancel" onclick={() => { selectedIds = new Set(); }}>Cancel</button>
      </div>
    {/if}

    {#if loading}
      <div class="dm-loading">Loading DM targets...</div>
    {:else if error}
      <div class="dm-error">{error}</div>
    {:else if items.length === 0}
      <div class="dm-empty">
        <p>No DM targets for this project yet. Run a scout to generate targets.</p>
      </div>
    {:else}
      <div class="dm-table-wrap">
        <table class="dm-table">
          <thead>
            <tr>
              <th scope="col" class="col-check">
                <input
                  type="checkbox"
                  checked={headerChecked}
                  onchange={toggleHeader}
                  aria-label="Select all rows"
                />
              </th>
              <th scope="col">Target</th>
              <th scope="col">Post</th>
              <th scope="col">Signal</th>
              <th scope="col">DM sent</th>
              <th scope="col">Status</th>
            </tr>
          </thead>
          <tbody>
            {#each items as item (item.id)}
              <tr>
                <td class="col-check">
                  <input
                    type="checkbox"
                    checked={selectedIds.has(item.id)}
                    onchange={(e) => toggleRow(item, e)}
                    aria-label="Select row"
                  />
                </td>
                <td class="col-target">
                  <a
                    class="dm-username"
                    href={`https://www.reddit.com/user/${item.username}`}
                    target="_blank"
                    rel="noopener noreferrer"
                  >u/{item.username}</a>
                  <div class="dm-intent-row">
                    <span class="dm-intent">Intent {item.intent_score}</span>
                    <a
                      class="dm-message"
                      href={`https://www.reddit.com/message/compose/?to=${item.username}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >Message ↗</a>
                  </div>
                </td>
                <td class="col-post">
                  <a
                    class="dm-post-title"
                    href={item.post_url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >{item.post_title || '(untitled)'}</a>
                  <div class="dm-post-meta">
                    <span>{postSubredditLabel(item)}</span>
                    {#if item.post_created_at}
                      <span>· {formatRelative(item.post_created_at)}</span>
                    {/if}
                  </div>
                </td>
                <td class="col-signal">
                  <span class="dm-signal">{item.signal || ''}</span>
                </td>
                <td class="col-dm">
                  {#if item.dm_status === 'new'}
                    <span class="dm-none">no DM sent yet</span>
                  {:else if item.draft_dm}
                    <p class="dm-draft">{item.draft_dm}</p>
                    <a
                      class="dm-message"
                      href={`https://www.reddit.com/message/compose/?to=${item.username}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >Message ↗</a>
                    <div class="dm-updated">Updated {formatRelative(item.dm_status_updated_at)}</div>
                  {:else}
                    <span class="dm-none">no draft</span>
                  {/if}
                </td>
                <td class="col-status">
                  <span class={statusPillClass(item.dm_status)}>{item.dm_status}</span>
                  <div class="dm-updated">Updated {formatRelative(item.dm_status_updated_at)}</div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
{/if}

<style>
  .dm-view { display: flex; flex-direction: column; gap: 16px; }
  .dm-header { display: flex; align-items: baseline; justify-content: space-between; }
  .dm-title { font-size: 18px; font-weight: 600; color: #d0d0e8; }
  .dm-total { font-size: 12px; color: #6b6b80; }

  .dm-tabs { display: flex; gap: 6px; flex-wrap: wrap; }
  .dm-tab {
    display: inline-flex; align-items: center; gap: 6px;
    padding: 5px 12px; border-radius: 6px;
    background: #1a1a24; color: #888;
    border: 1px solid #2a2a3a; cursor: pointer;
    font-size: 13px; transition: all 0.15s;
  }
  .dm-tab:hover { background: #23233a; color: #c0c0d8; }
  .dm-tab.active { background: #7c6af520; color: #a090ff; border-color: #7c6af560; }
  .dm-tab-count {
    font-size: 11px; font-weight: 700; padding: 1px 6px; border-radius: 8px;
    background: #2a2a3a; color: #9090a8;
  }
  .dm-tab.active .dm-tab-count { background: #7c6af530; color: #c0b0ff; }

  .bulk-bar {
    display: flex; align-items: center; gap: 12px;
    padding: 8px 12px; border-radius: 6px;
    background: #7c6af515; border: 1px solid #7c6af540;
  }
  .bulk-count { font-size: 13px; font-weight: 600; color: #d0d0e8; }
  .bulk-actions { display: flex; gap: 6px; }
  .bulk-btn {
    padding: 4px 12px; font-size: 12px; font-weight: 500;
    border-radius: 5px; cursor: pointer;
    background: #23233a; color: #c0c0d8;
    border: 1px solid #3a3a50;
  }
  .bulk-btn:hover:not(:disabled) { background: #2a2a45; }
  .bulk-btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .bulk-cancel { margin-left: auto; padding: 4px 10px; font-size: 12px; background: transparent; color: #9090a8; border: 1px solid #3a3a50; border-radius: 5px; cursor: pointer; }

  .dm-loading, .dm-empty, .dm-error {
    padding: 24px; text-align: center; color: #9090a8; font-size: 13px;
  }
  .dm-error { color: #e06c75; background: #e06c7515; border: 1px solid #e06c7530; border-radius: 6px; }

  .dm-table-wrap { overflow-x: auto; border: 1px solid #2a2a3a; border-radius: 6px; }
  .dm-table { width: 100%; border-collapse: collapse; font-size: 13px; }
  .dm-table th {
    text-align: left; padding: 10px 12px;
    background: #13131a; color: #6b6b80;
    font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em;
    font-size: 11px; border-bottom: 1px solid #2a2a3a;
  }
  .dm-table td {
    padding: 10px 12px; border-bottom: 1px solid #1a1a24; vertical-align: top; color: #c0c0d8;
  }
  .dm-table tbody tr:hover { background: #13131a; }

  .col-check { width: 32px; }
  .col-check input { accent-color: #7c6af5; cursor: pointer; }

  .dm-username { color: #7c6af5; font-weight: 600; text-decoration: none; }
  .dm-username:hover { text-decoration: underline; }
  .dm-intent-row { display: flex; gap: 8px; align-items: center; margin-top: 4px; }
  .dm-intent { font-size: 11px; padding: 1px 6px; border-radius: 3px; background: #2a2a3a; color: #9090a8; }
  .dm-message { font-size: 11px; color: #56b6c2; text-decoration: none; }
  .dm-message:hover { text-decoration: underline; }

  .dm-post-title { color: #c0c0d8; text-decoration: none; }
  .dm-post-title:hover { text-decoration: underline; }
  .dm-post-meta { font-size: 11px; color: #6b6b80; margin-top: 4px; }

  .dm-signal { color: #9090a8; font-style: italic; }

  .dm-draft {
    margin: 0 0 4px 0;
    color: #c0c0d8;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    line-clamp: 2;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .dm-none { color: #6b6b80; font-style: italic; font-size: 12px; }
  .dm-updated { font-size: 10px; color: #4a4a60; margin-top: 4px; }

  .status-pill {
    display: inline-block; font-size: 11px; font-weight: 700;
    padding: 2px 8px; border-radius: 4px; text-transform: uppercase; letter-spacing: 0.04em;
  }
  .status-new { background: #2a2a3a; color: #9090a8; }
  .status-sent { background: #7c6af520; color: #a090ff; }
  .status-replied { background: #98c37920; color: #98c379; }
  .status-ignored { background: #3a3a50; color: #6b6b80; }
</style>
