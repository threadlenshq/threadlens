<script>
  import { manualScout, manualScoutCommit, posts } from '../lib/api.js';
  import PostCard from './PostCard.svelte';
  import { scoreColor } from '../lib/format.js';

  let { projectId, onNavigateToPost } = $props();

  let url = $state('');
  let platform = $state('reddit');
  let loading = $state(false);
  let result = $state(null);
  let error = $state('');
  let decisionLoading = $state(false);
  let postLoading = $state(false);
  let fetchedPost = $state(null);

  let canSubmit = $derived(url.trim().length > 0 && !loading);

  async function fetchPostDetails(postId) {
    if (!postId) return;
    postLoading = true;
    try {
      fetchedPost = await posts.get(projectId, postId);
    } catch {
      fetchedPost = null;
    } finally {
      postLoading = false;
    }
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!canSubmit) return;

    loading = true;
    error = '';
    result = null;
    fetchedPost = null;

    try {
      result = await manualScout(projectId, url.trim(), platform);
      if (result.post_id) {
        await fetchPostDetails(result.post_id);
      }
    } catch (e) {
      error = e.message;
      result = null;
    } finally {
      loading = false;
    }
  }

  async function handleDecision(decision) {
    if (!result?.post || decisionLoading) return;

    decisionLoading = true;
    error = '';

    try {
      const commitResult = await manualScoutCommit(projectId, decision, result.post);
      result = commitResult;
      fetchedPost = null;
      if (commitResult.post_id && commitResult.status === 'saved') {
        await fetchPostDetails(commitResult.post_id);
      }
    } catch (e) {
      error = e.message;
    } finally {
      decisionLoading = false;
    }
  }

  function handleReset() {
    url = '';
    platform = 'reddit';
    result = null;
    error = '';
    loading = false;
    decisionLoading = false;
    postLoading = false;
    fetchedPost = null;
  }
</script>

<div class="manual-tasker">
  {#if !result}
    <form class="scout-form" onsubmit={handleSubmit}>
      <div class="field-group">
        <label class="field-label" for="manual-scout-url">URL</label>
        <input
          id="manual-scout-url"
          class="field-input"
          type="text"
          placeholder="https://reddit.com/r/... or https://bsky.app/..."
          bind:value={url}
          disabled={loading}
        />
      </div>

      <div class="field-group">
        <label class="field-label">Platform</label>
        <div class="platform-options">
          <button
            type="button"
            class="platform-btn"
            class:active={platform === 'reddit'}
            onclick={() => platform = 'reddit'}
            disabled={loading}
          >
            Reddit
          </button>
          <button
            type="button"
            class="platform-btn"
            class:active={platform === 'bluesky'}
            onclick={() => platform = 'bluesky'}
            disabled={loading}
          >
            Bluesky
          </button>
        </div>
      </div>

      <button class="submit-btn" type="submit" disabled={!canSubmit}>
        {#if loading}
          <span class="spinner"></span>
          Scouting...
        {:else}
          Scout URL
        {/if}
      </button>
    </form>
  {/if}

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#if result}
    <div class="result-card">
      {#if result.status === 'saved'}
        <div class="result-header success">
          <span class="result-icon">&#10003;</span>
          <span class="result-title">Saved</span>
        </div>
        {#if postLoading}
          <div class="post-loading"><span class="spinner"></span> Loading post details...</div>
        {:else if fetchedPost}
          <PostCard post={fetchedPost} onSelect={() => onNavigateToPost?.(fetchedPost)} />
        {:else if result.post}
          <div class="post-info">
            <a class="post-link" href={result.post.url} target="_blank" rel="noopener">
              {result.post.title || result.post.body || 'Untitled post'}
            </a>
            {#if result.post.final_score !== undefined || result.post.post_score !== undefined}
              <span class="score-badge" style="color: {scoreColor(result.post.final_score ?? result.post.post_score)}">
                {(result.post.final_score ?? result.post.post_score).toFixed ? (result.post.final_score ?? result.post.post_score).toFixed(1) : (result.post.final_score ?? result.post.post_score)}
              </span>
            {/if}
          </div>
          <button type="button" class="view-in-inbox-btn" onclick={() => onNavigateToPost?.(result.post)}>
            View in Inbox
          </button>
        {/if}

      {:else if result.status === 'already_scouted'}
        <div class="result-header info">
          <span class="result-icon">&#9432;</span>
          <span class="result-title">Already Scouted</span>
        </div>
        {#if postLoading}
          <div class="post-loading"><span class="spinner"></span> Loading post details...</div>
        {:else if fetchedPost}
          <PostCard post={fetchedPost} onSelect={() => onNavigateToPost?.(fetchedPost)} />
        {:else if result.post}
          <div class="post-info">
            <a class="post-link" href={result.post.url} target="_blank" rel="noopener">
              {result.post.title || result.post.body || 'Untitled post'}
            </a>
            {#if result.post.final_score !== undefined || result.post.post_score !== undefined}
              <span class="score-badge" style="color: {scoreColor(result.post.final_score ?? result.post.post_score)}">
                {(result.post.final_score ?? result.post.post_score).toFixed ? (result.post.final_score ?? result.post.post_score).toFixed(1) : (result.post.final_score ?? result.post.post_score)}
              </span>
            {/if}
          </div>
          <button type="button" class="view-in-inbox-btn" onclick={() => onNavigateToPost?.(result.post)}>
            View in Inbox
          </button>
        {/if}

      {:else if result.status === 'needs_decision'}
        <div class="result-header warning">
          <span class="result-icon">&#9888;</span>
          <span class="result-title">Needs Decision</span>
        </div>
        {#if postLoading}
          <div class="post-loading"><span class="spinner"></span> Loading post details...</div>
        {:else if fetchedPost}
          <PostCard post={fetchedPost} onSelect={() => onNavigateToPost?.(fetchedPost)} />
        {:else if result.post}
          <div class="post-info">
            <a class="post-link" href={result.post.url} target="_blank" rel="noopener">
              {result.post.title || result.post.body || 'Untitled post'}
            </a>
            {#if result.post.final_score !== undefined || result.post.post_score !== undefined}
              <span class="score-badge" style="color: {scoreColor(result.post.final_score ?? result.post.post_score)}">
                {(result.post.final_score ?? result.post.post_score).toFixed ? (result.post.final_score ?? result.post.post_score).toFixed(1) : (result.post.final_score ?? result.post.post_score)}
              </span>
            {/if}
          </div>
          <button type="button" class="view-in-inbox-btn" onclick={() => onNavigateToPost?.(result.post)}>
            View in Inbox
          </button>
        {/if}
        <div class="decision-actions">
          <button
            class="decision-btn keep"
            onclick={() => handleDecision('keep')}
            disabled={decisionLoading}
          >
            {#if decisionLoading}
              <span class="spinner"></span>
            {:else}
              Keep
            {/if}
          </button>
          <button
            class="decision-btn exclude"
            onclick={() => handleDecision('exclude')}
            disabled={decisionLoading}
          >
            {#if decisionLoading}
              <span class="spinner"></span>
            {:else}
              Exclude
            {/if}
          </button>
        </div>

      {:else if result.status === 'filtered'}
        <div class="result-header filtered">
          <span class="result-icon">&#128683;</span>
          <span class="result-title">Filtered</span>
        </div>
        {#if result.reason}
          <p class="filter-reason">{result.reason}</p>
        {/if}
        {#if result.explanation}
          <p class="filter-explanation">{result.explanation}</p>
        {/if}

      {:else if result.status === 'excluded'}
        <div class="result-header filtered">
          <span class="result-icon">&#128683;</span>
          <span class="result-title">Excluded</span>
        </div>

      {:else if result.status === 'error'}
        <div class="result-header error">
          <span class="result-icon">&#10007;</span>
          <span class="result-title">Error</span>
        </div>
        {#if result.message}
          <p class="error-detail">{result.message}</p>
        {/if}
      {:else}
        <div class="result-header">
          <span class="result-title">Unexpected response: {result.status}</span>
        </div>
      {/if}

      <button type="button" class="reset-btn" onclick={handleReset}>
        Scout another URL
      </button>
    </div>
  {/if}
</div>

<style>
  .manual-tasker {
    display: flex;
    flex-direction: column;
    gap: 16px;
    max-width: 900px;
  }

  .scout-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
    width: 100%;
  }

  .field-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .field-label {
    font-size: 12px;
    font-weight: 600;
    color: #aaa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .field-input {
    width: 100%;
    padding: 9px 12px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 14px;
    transition: border-color 0.15s;
  }

  .field-input:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .field-input:disabled {
    opacity: 0.6;
  }

  .platform-options {
    display: flex;
    gap: 8px;
  }

  .platform-btn {
    padding: 7px 16px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #888;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .platform-btn:hover:not(:disabled) {
    border-color: #3a3a55;
    color: #e2e2e8;
  }

  .platform-btn.active {
    background: #7c6af520;
    border-color: #7c6af5;
    color: #7c6af5;
  }

  .platform-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .submit-btn {
    align-self: flex-start;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 9px 20px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .submit-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .submit-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: #fff;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .result-card {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
  }

  .result-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    font-weight: 600;
  }

  .result-header.success {
    color: #98c379;
  }

  .result-header.info {
    color: #61afef;
  }

  .result-header.warning {
    color: #e5c07b;
  }

  .result-header.filtered {
    color: #888;
  }

  .result-header.error {
    color: #f87171;
  }

  .result-icon {
    font-size: 16px;
  }

  .post-loading {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: #888;
  }

  .post-info {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .post-link {
    color: #7c6af5;
    font-size: 14px;
    text-decoration: none;
    word-break: break-all;
  }

  .post-link:hover {
    text-decoration: underline;
  }

  .score-badge {
    font-size: 16px;
    font-weight: 700;
    flex-shrink: 0;
  }

  .decision-actions {
    display: flex;
    gap: 10px;
  }

  .decision-btn {
    padding: 7px 16px;
    border: none;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .decision-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .decision-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .decision-btn.keep {
    background: #98c379;
    color: #1a1a24;
  }

  .decision-btn.exclude {
    background: #2a2a3a;
    border: 1px solid #444;
    color: #888;
  }

  .decision-btn.exclude:hover:not(:disabled) {
    border-color: #f87171;
    color: #f87171;
    background: #3a1a1a;
  }

  .filter-reason {
    font-size: 13px;
    color: #888;
    margin: 0;
  }

  .filter-explanation {
    font-size: 12px;
    color: #666;
    margin: 0;
  }

  .error-detail {
    font-size: 13px;
    color: #f87171;
    margin: 0;
  }

  .view-in-inbox-btn {
    align-self: flex-start;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    background: #7c6af518;
    border: 1px solid #7c6af530;
    border-radius: 4px;
    color: #7c6af5;
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .view-in-inbox-btn:hover {
    background: #7c6af528;
    border-color: #7c6af5;
  }

  .reset-btn {
    align-self: flex-start;
    padding: 6px 14px;
    background: none;
    border: 1px solid #444;
    border-radius: 6px;
    color: #888;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .reset-btn:hover {
    color: #e2e2e8;
    border-color: #666;
  }
</style>
