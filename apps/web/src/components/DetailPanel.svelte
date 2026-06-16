<script>
  export let post = null;
  export let generating = false;
  export let generateError = null;
  export let postingReply = false;
  export let postReplyResult = null; // { type: 'success' | 'error', message: string }
  export let projectMode = 'marketing';
  export let selectedProjectId = null;

  export let onGenerateDraft = null;
  export let onStatusChange = null;
  export let onDraftChange = null;
  export let onPostReply = null;

  const PLATFORM_COLORS = {
    reddit: '#FF4500',
    bluesky: '#0085FF',
  };

  let draftValue = '';
  let copied = false;

  // DM draft state
  let dmDraftValues = {};   // { username: draft text }
  let dmGenerating = {};    // { username: true/false }
  let dmCopied = {};        // { username: true/false }
  let dmErrors = {};        // { username: error message }

  // Sync draftValue when post or draft changes
  $: if (post) {
    draftValue = post.draft_comment || '';
    // Sync DM draft values from post data
    dmDraftValues = {};
    (post.dm_targets || []).forEach(t => {
      if (t.draft_dm) dmDraftValues[t.username] = t.draft_dm;
    });
  }

  function scoreColor(score) {
    if (score == null) return '#4a4a60';
    if (score >= 9) return '#e5c07b';
    if (score >= 7) return '#98c379';
    if (score >= 5) return '#61afef';
    if (score >= 3) return '#d19a66';
    return '#6b6b80';
  }

  function formatDate(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    return d.toLocaleDateString('en-GB', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  function generateDraft() {
    onGenerateDraft?.({ id: post.id, platform: post.platform });
  }

  function saveDraftChange() {
    onDraftChange?.({ id: post.id, platform: post.platform, draft_comment: draftValue });
  }

  function copyDraft() {
    if (!draftValue) return;
    navigator.clipboard.writeText(draftValue).then(() => {
      copied = true;
      setTimeout(() => (copied = false), 1500);
    });
  }

  function postReply() {
    if (!draftValue || postingReply) return;
    onPostReply?.({ id: post.id, platform: post.platform, text: draftValue });
  }

  async function generateDmDraft(username) {
    dmGenerating[username] = true;
    dmGenerating = dmGenerating;
    delete dmErrors[username];
    dmErrors = dmErrors;
    try {
      const res = await fetch(
        `/api/projects/${selectedProjectId}/posts/${encodeURIComponent(post.id)}/dm/${encodeURIComponent(username)}/generate-draft`,
        { method: 'POST' }
      );
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
      // Update local state from response
      const target = (data.dm_targets || []).find(t => t.username === username);
      if (target?.draft_dm) {
        dmDraftValues[username] = target.draft_dm;
        dmDraftValues = dmDraftValues;
      }
      onStatusChange?.({ id: post.id, platform: post.platform, status: post.status });
    } catch (e) {
      dmErrors[username] = e.message;
      dmErrors = dmErrors;
    } finally {
      dmGenerating[username] = false;
      dmGenerating = dmGenerating;
    }
  }

  async function saveDmDraft(username) {
    try {
      await fetch(
        `/api/projects/${selectedProjectId}/posts/${encodeURIComponent(post.id)}/dm/${encodeURIComponent(username)}`,
        {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ draft_dm: dmDraftValues[username] || '' }),
        }
      );
    } catch (e) {
      console.error('Failed to save DM draft:', e);
    }
  }

  function copyDmDraft(username) {
    const text = dmDraftValues[username];
    if (!text) return;
    navigator.clipboard.writeText(text).then(() => {
      dmCopied[username] = true;
      dmCopied = dmCopied;
      setTimeout(() => {
        dmCopied[username] = false;
        dmCopied = dmCopied;
      }, 1500);
    });
  }

  function openDmCompose(username) {
    window.open(`https://www.reddit.com/message/compose/?to=${username}`, '_blank', 'noopener,noreferrer');
  }

  function parseProfile(signalsJSON) {
    if (!signalsJSON) return null;
    try {
      return JSON.parse(signalsJSON);
    } catch {
      return null;
    }
  }

  function openInBrowser() {
    if (post.url) window.open(post.url, '_blank', 'noopener,noreferrer');
  }

  function setStatus(status) {
    onStatusChange?.({ id: post.id, platform: post.platform, status });
  }

  // Flat field access - Scout API returns flat structure
  $: platformColor = post ? (PLATFORM_COLORS[post.platform] || '#7c6af5') : '#7c6af5';
  $: postScore = post?.post_score ?? null;
  $: commentScore = post?.comment_score ?? null;
  $: finalScore = post?.final_score ?? null;
  $: angle = post?.angle ?? null;
  $: why = post?.why ?? null;
  $: upvotes = post?.reddit_score ?? post?.like_count ?? null;
  $: numComments = post?.num_comments ?? post?.reply_count ?? null;
  $: dmTargets = post?.dm_targets || [];
  $: signalType = post?.signal_type ?? null;

  function escapeRegex(s) {
    return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  }

  function highlightSegments(text, phrase) {
    if (!text) return [{ type: 'text', value: '' }];
    const trimmed = (phrase || '').trim();
    if (!trimmed) return [{ type: 'text', value: text }];

    const words = trimmed.split(/\s+/).filter(w => w.length >= 3);
    const parts = [];
    if (words.length > 1) {
      parts.push(`\\b${escapeRegex(trimmed)}\\b`);
    }
    for (const w of words) {
      parts.push(`\\b${escapeRegex(w)}\\b`);
    }
    if (parts.length === 0) return [{ type: 'text', value: text }];

    const re = new RegExp(`(${parts.join('|')})`, 'gi');
    const segments = [];
    let lastIndex = 0;
    let match;
    while ((match = re.exec(text)) !== null) {
      if (match.index > lastIndex) {
        segments.push({ type: 'text', value: text.slice(lastIndex, match.index) });
      }
      const matched = match[0];
      const isPhrase = words.length > 1 && matched.toLowerCase() === trimmed.toLowerCase();
      segments.push({ type: isPhrase ? 'full' : 'word', value: matched });
      lastIndex = match.index + matched.length;
    }
    if (lastIndex < text.length) {
      segments.push({ type: 'text', value: text.slice(lastIndex) });
    }
    return segments;
  }

  $: titleSegments = highlightSegments(post?.title || '', angle);
  $: bodySegments = highlightSegments(post?.body || '', angle);
</script>

{#if !post}
  <div class="empty-panel">
    <div class="empty-icon">&#128200;</div>
    <p>Select a post to review</p>
  </div>
{:else}
  <div class="panel">
    <div class="panel-header">
      <div class="post-meta">
        <span class="platform-label" style="color: {platformColor}">
          {post.platform === 'reddit' ? 'Reddit' : 'Bluesky'}
        </span>
        <span class="author">by {post.author}</span>
        <span class="date">{formatDate(post.created_at)}</span>
      </div>
    </div>

    {#if projectMode === 'research'}
      <div class="actions-section top-actions">
        <div class="action-row">
          <button class="action-btn open" on:click={openInBrowser} title="Open in browser">
            Open &#8599;
          </button>
          <span class="tooltip-wrap">
            <button
              class="action-btn reviewed"
              class:active={post.status === 'reviewed'}
              on:click={() => setStatus('reviewed')}
            >
              {post.status === 'reviewed' ? '\u2713 Reviewed' : 'Mark as Reviewed'}
            </button>
            <span class="tooltip">No effect on analysis</span>
          </span>
          <span class="tooltip-wrap">
            <button
              class="action-btn star"
              class:active={post.status === 'starred'}
              on:click={() => setStatus('starred')}
            >
              {post.status === 'starred' ? '\u2605 Starred' : '\u2606 Star'}
            </button>
            <span class="tooltip">Gets extra weight in analysis</span>
          </span>
          <span class="tooltip-wrap">
            <button
              class="action-btn exclude"
              class:active={post.status === 'excluded'}
              on:click={() => setStatus('excluded')}
            >
              {post.status === 'excluded' ? 'Excluded' : 'Exclude'}
            </button>
            <span class="tooltip">Removed from analysis</span>
          </span>
        </div>
      </div>
    {:else}
      <div class="actions-section top-actions">
        <div class="action-row">
          <button class="action-btn open" on:click={openInBrowser} title="Open in browser">
            Open &#8599;
          </button>
          <button
            class="action-btn comment"
            class:active={post.status === 'commented'}
            on:click={() => setStatus('commented')}
          >
            {post.status === 'commented' ? '\u2713 Commented' : 'Mark as Commented'}
          </button>
          <button
            class="action-btn skip"
            class:active={post.status === 'skipped'}
            on:click={() => setStatus('skipped')}
          >
            {post.status === 'skipped' ? 'Skipped' : 'Skip'}
          </button>
        </div>
      </div>
    {/if}

    {#if post.subreddit}
      <span class="subreddit">r/{post.subreddit}</span>
    {/if}

    {#if post.title}
      <h2 class="post-title">{#each titleSegments as seg}{#if seg.type === 'full'}<mark class="hl-full">{seg.value}</mark>{:else if seg.type === 'word'}<mark class="hl-word">{seg.value}</mark>{:else}{seg.value}{/if}{/each}</h2>
    {/if}

    <div class="scoring-section">
      <h3 class="section-label">Scoring</h3>
      <div class="score-grid">
        {#if postScore != null}
          <div class="score-item">
            <span class="score-label">Post</span>
            <span class="score-val" style="color: {scoreColor(postScore)}">{postScore}</span>
          </div>
        {/if}
        {#if commentScore != null}
          <div class="score-item">
            <span class="score-label">Comment</span>
            <span class="score-val" style="color: {scoreColor(commentScore)}">{commentScore}</span>
          </div>
        {/if}
        {#if finalScore != null}
          <div class="score-item">
            <span class="score-label">Final</span>
            <span class="score-val final" style="color: {scoreColor(finalScore)}">
              {typeof finalScore === 'number' ? finalScore.toFixed(1) : finalScore}
            </span>
          </div>
        {/if}
        {#if angle}
          <div class="score-item">
            <span class="score-label">Angle</span>
            <span class="score-val angle">{angle}</span>
          </div>
        {/if}
      </div>
      {#if why}
        <p class="score-why">{why}</p>
      {/if}
      {#if signalType}
        <div class="signal-type">
          <span class="signal-label">Signal</span>
          <span class="signal-value" class:frustration={signalType === 'frustration'} class:workaround={signalType === 'workaround'} class:seeking={signalType === 'seeking_solution'}>
            {signalType === 'seeking_solution' ? 'Seeking Solution' : signalType.charAt(0).toUpperCase() + signalType.slice(1)}
          </span>
        </div>
      {/if}
      <div class="engagement-metrics">
        {#if upvotes != null}
          <span class="eng-metric">
            {post.platform === 'reddit' ? '\u25B2' : '\u2661'} {upvotes}
          </span>
        {/if}
        {#if numComments != null}
          <span class="eng-metric">&#128172; {numComments}</span>
        {/if}
      </div>
    </div>

    {#if post.body && post.body !== post.title}
      <div class="post-body">
        <p>{#each bodySegments as seg}{#if seg.type === 'full'}<mark class="hl-full">{seg.value}</mark>{:else if seg.type === 'word'}<mark class="hl-word">{seg.value}</mark>{:else}{seg.value}{/if}{/each}</p>
      </div>
    {/if}

    {#if projectMode === 'marketing'}
      <div class="draft-section">
        <div class="draft-header">
          <h3 class="section-label">Draft Comment</h3>
        </div>

        {#if generating}
          <div class="generating-state">
            <span class="spinner"></span>
            Generating...
          </div>
        {:else}
          <button class="generate-btn" on:click={generateDraft}>
            {post.draft_comment ? 'Regenerate Draft' : 'Generate Draft'}
          </button>
        {/if}

        {#if generateError}
          <p class="draft-error">{generateError}</p>
        {/if}

        {#if post.draft_comment || draftValue}
          <div class="draft-area">
            {#if post.draft_provider}
              <span class="draft-provider">via {post.draft_provider}</span>
            {/if}
            <textarea
              class="draft-input"
              bind:value={draftValue}
              placeholder="Draft comment..."
              rows="6"
              on:blur={saveDraftChange}
            ></textarea>
            {#if post.platform === 'bluesky'}
              <span class="char-count" class:over={[...new Intl.Segmenter().segment(draftValue)].length > 300}>
                {[...new Intl.Segmenter().segment(draftValue)].length}/300
              </span>
            {/if}
            <div class="draft-actions">
              <button class="copy-btn" on:click={copyDraft}>
                {copied ? '\u2713 Copied' : '\u{1F4CB} Copy to Clipboard'}
              </button>
              {#if post.platform === 'bluesky' && post.status !== 'commented'}
                <button class="post-reply-btn" on:click={postReply} disabled={postingReply || [...new Intl.Segmenter().segment(draftValue)].length > 300}>
                  {postingReply ? 'Posting...' : 'Post Reply'}
                </button>
              {/if}
            </div>
            {#if postReplyResult}
              <p class="post-result" class:success={postReplyResult.type === 'success'} class:error={postReplyResult.type === 'error'}>
                {postReplyResult.message}
              </p>
            {/if}
          </div>
        {/if}
      </div>
    {/if}

    {#if projectMode === 'marketing' && dmTargets.length > 0}
      <div class="dm-section">
        <h3 class="section-label">DM Targets <span class="dm-count">{dmTargets.length}</span></h3>
        {#each dmTargets as target}
          <div class="dm-card">
            <div class="dm-header">
              <a
                class="dm-username"
                href="https://www.reddit.com/user/{target.username}"
                target="_blank"
                rel="noopener noreferrer"
              >
                u/{target.username}
              </a>
              <span class="dm-intent" class:high={target.intent_score >= 7} class:medium={target.intent_score >= 5 && target.intent_score < 7}>
                Intent: {target.intent_score}
              </span>
            </div>
            {#if target.signal}
              <p class="dm-signal">"{target.signal}"</p>
            {/if}
            {#if target.context}
              <p class="dm-context">{target.context}</p>
            {/if}
            {#if target.approach}
              <p class="dm-approach"><strong>Approach:</strong> {target.approach}</p>
            {/if}
            {#if target.profile_signals}
              {@const profile = parseProfile(target.profile_signals)}
              {#if profile}
                <details class="profile-signals">
                  <summary class="profile-signals-summary">Profile Signals {#if target.profile_score != null}<span class="profile-score" class:negative={target.profile_score < 0} class:positive={target.profile_score > 0}>{target.profile_score}</span>{/if}</summary>
                  <div class="profile-signals-grid">
                    <span class="ps-label">Account age</span>
                    <span class="ps-value">{profile.account_age_days} days</span>
                    <span class="ps-label">Comment karma</span>
                    <span class="ps-value">{profile.comment_karma}</span>
                    <span class="ps-label">Post karma</span>
                    <span class="ps-value">{profile.post_karma}</span>
                    <span class="ps-label">Verified email</span>
                    <span class="ps-value">{profile.has_verified_email ? '\u2713' : '\u2014'}</span>
                    <span class="ps-label">Gold</span>
                    <span class="ps-value">{profile.is_gold ? '\u2713' : '\u2014'}</span>
                    <span class="ps-label">NSFW</span>
                    <span class="ps-value">{profile.is_nsfw ? '\u2713' : '\u2014'}</span>
                    <span class="ps-label">Self-promo ratio</span>
                    <span class="ps-value">{(profile.self_promo_ratio * 100).toFixed(1)}%</span>
                    <span class="ps-label">Subreddits</span>
                    <span class="ps-value">{profile.subreddit_count}</span>
                  </div>
                </details>
              {/if}
            {/if}

            <div class="dm-draft-area">
              {#if dmGenerating[target.username]}
                <div class="generating-state">
                  <span class="spinner"></span>
                  Generating DM...
                </div>
              {:else}
                <button class="generate-dm-btn" on:click={() => generateDmDraft(target.username)}>
                  {dmDraftValues[target.username] ? 'Regenerate DM' : 'Generate DM Draft'}
                </button>
              {/if}

              {#if dmErrors[target.username]}
                <p class="draft-error">{dmErrors[target.username]}</p>
              {/if}

              {#if dmDraftValues[target.username]}
                <textarea
                  class="dm-draft-input"
                  bind:value={dmDraftValues[target.username]}
                  rows="4"
                  on:blur={() => saveDmDraft(target.username)}
                ></textarea>
                <div class="dm-actions">
                  <button class="copy-btn" on:click={() => copyDmDraft(target.username)}>
                    {dmCopied[target.username] ? '\u2713 Copied' : '\u{1F4CB} Copy'}
                  </button>
                  <button class="send-dm-btn" on:click={() => { copyDmDraft(target.username); openDmCompose(target.username); }}>
                    Send DM &#8599;
                  </button>
                </div>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  .empty-panel {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    gap: 12px;
    color: #3a3a50;
  }

  .empty-icon {
    font-size: 40px;
    opacity: 0.4;
  }

  .empty-panel p {
    font-size: 15px;
  }

  .panel {
    display: flex;
    flex-direction: column;
    gap: 20px;
    padding: 20px;
    height: 100%;
    overflow-y: auto;
  }

  .panel::-webkit-scrollbar {
    width: 4px;
  }

  .panel::-webkit-scrollbar-track {
    background: transparent;
  }

  .panel::-webkit-scrollbar-thumb {
    background: #3a3a50;
    border-radius: 2px;
  }

  .panel-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
  }

  .post-meta {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .platform-label {
    font-size: 13px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .author {
    font-size: 13px;
    color: #6b6b80;
  }

  .date {
    font-size: 12px;
    color: #4a4a60;
  }

  .top-actions {
    border-top: none;
    padding-top: 0;
  }

  .action-btn.open {
    background: #2a2a3a;
    color: #9090a8;
    border-color: #3a3a50;
  }

  .action-btn.open:hover {
    background: #3a3a50;
    color: #e2e2e8;
  }

  .subreddit {
    font-size: 12px;
    color: #7c6af5;
    font-weight: 500;
  }

  .post-title {
    font-size: 18px;
    font-weight: 600;
    color: #d0d0e8;
    line-height: 1.4;
  }

  .post-body {
    background: #13131a;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    padding: 12px;
  }

  .post-body p {
    font-size: 14px;
    color: #8080a0;
    line-height: 1.6;
    white-space: pre-wrap;
  }

  mark.hl-word {
    background: #7c6af560;
    color: #ffffff;
    padding: 1px 4px;
    border-radius: 3px;
    font-weight: 500;
    box-shadow: 0 0 0 1px #7c6af580;
  }

  mark.hl-full {
    background: #ffd54a;
    color: #1a1a24;
    padding: 2px 6px;
    border-radius: 4px;
    font-weight: 700;
    box-shadow: 0 0 0 2px #ffd54a, 0 0 12px #ffd54a80;
  }

  .section-label {
    font-size: 11px;
    font-weight: 700;
    color: #4a4a60;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    margin-bottom: 8px;
  }

  .scoring-section,
  .draft-section,
  .actions-section {
    border-top: 1px solid #1e1e2a;
    padding-top: 16px;
  }

  .score-grid {
    display: flex;
    gap: 16px;
    flex-wrap: wrap;
    margin-bottom: 8px;
  }

  .score-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .score-label {
    font-size: 11px;
    color: #4a4a60;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .score-val {
    font-size: 20px;
    font-weight: 700;
  }

  .score-val.final {
    font-size: 24px;
  }

  .score-val.angle {
    color: #7c6af5;
  }

  .score-why {
    font-size: 13px;
    color: #7c7c96;
    line-height: 1.5;
    font-style: italic;
    margin-bottom: 8px;
  }

  .signal-type {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .signal-label {
    font-size: 11px;
    color: #4a4a60;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .signal-value {
    font-size: 12px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 3px;
    color: #6b6b80;
    background: #2a2a3a;
  }

  .signal-value.frustration {
    color: #e06c75;
    background: #e06c7518;
  }

  .signal-value.workaround {
    color: #e5c07b;
    background: #e5c07b18;
  }

  .signal-value.seeking {
    color: #98c379;
    background: #98c37918;
  }

  .engagement-metrics {
    display: flex;
    gap: 12px;
  }

  .eng-metric {
    font-size: 13px;
    color: #4a4a60;
  }

  .draft-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
  }

  .draft-header .section-label {
    margin-bottom: 0;
  }

  .generate-btn {
    padding: 7px 16px;
    background: #7c6af520;
    color: #a090ff;
    border: 1px solid #7c6af540;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
    margin-bottom: 10px;
  }

  .generate-btn:hover {
    background: #7c6af535;
  }

  .generating-state {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: #7c6af5;
    font-style: italic;
    margin-bottom: 10px;
  }

  .spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid #7c6af530;
    border-top-color: #7c6af5;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .draft-error {
    font-size: 12px;
    color: #e06c75;
    background: #e06c7510;
    border-radius: 4px;
    padding: 6px 10px;
    margin-bottom: 8px;
    line-height: 1.4;
  }

  .draft-area {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .draft-provider {
    font-size: 10px;
    color: #4a4a60;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .draft-input {
    width: 100%;
    background: #13131a;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #c0c0d8;
    font-size: 14px;
    padding: 10px;
    resize: vertical;
    font-family: inherit;
    line-height: 1.6;
    min-height: 120px;
  }

  .draft-input:focus {
    outline: none;
    border-color: #56b6c2;
  }

  .copy-btn {
    align-self: flex-start;
    padding: 5px 12px;
    font-size: 13px;
    color: #56b6c2;
    background: #56b6c210;
    border: 1px solid #56b6c230;
    border-radius: 5px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .copy-btn:hover {
    background: #56b6c225;
  }

  .char-count {
    font-size: 11px;
    color: #6b6b80;
    text-align: right;
    display: block;
    margin-top: -4px;
    margin-bottom: 4px;
  }

  .char-count.over {
    color: #e06c75;
    font-weight: 600;
  }

  .draft-actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .post-reply-btn {
    padding: 5px 12px;
    font-size: 13px;
    color: #0085FF;
    background: #0085FF15;
    border: 1px solid #0085FF30;
    border-radius: 5px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .post-reply-btn:hover:not(:disabled) {
    background: #0085FF30;
  }

  .post-reply-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .post-result {
    font-size: 12px;
    margin-top: 6px;
    padding: 6px 10px;
    border-radius: 5px;
  }

  .post-result.success {
    color: #98c379;
    background: #98c37915;
    border: 1px solid #98c37930;
  }

  .post-result.error {
    color: #e06c75;
    background: #e06c7515;
    border: 1px solid #e06c7530;
  }

  .dm-section {
    border-top: 1px solid #1e1e2a;
    padding-top: 16px;
  }

  .dm-count {
    font-size: 10px;
    font-weight: 700;
    color: #e5c07b;
    background: #e5c07b18;
    padding: 1px 5px;
    border-radius: 3px;
    margin-left: 6px;
    vertical-align: middle;
  }

  .dm-card {
    background: #13131a;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    padding: 12px;
    margin-bottom: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .dm-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .dm-username {
    font-size: 14px;
    font-weight: 600;
    color: #7c6af5;
    text-decoration: none;
  }

  .dm-username:hover {
    text-decoration: underline;
  }

  .dm-intent {
    font-size: 11px;
    font-weight: 700;
    padding: 2px 7px;
    border-radius: 3px;
    color: #6b6b80;
    background: #2a2a3a;
  }

  .dm-intent.high {
    color: #98c379;
    background: #98c37918;
  }

  .dm-intent.medium {
    color: #e5c07b;
    background: #e5c07b18;
  }

  .dm-signal {
    font-size: 13px;
    color: #9090a8;
    font-style: italic;
    line-height: 1.5;
  }

  .dm-context {
    font-size: 12px;
    color: #6b6b80;
    line-height: 1.4;
  }

  .dm-approach {
    font-size: 12px;
    color: #8080a0;
    line-height: 1.4;
  }

  .dm-approach :global(strong) {
    color: #56b6c2;
    font-weight: 600;
  }

  .dm-draft-area {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 6px;
    padding-top: 8px;
    border-top: 1px solid #2a2a3a;
  }

  .generate-dm-btn {
    align-self: flex-start;
    padding: 5px 12px;
    background: #e5c07b18;
    color: #e5c07b;
    border: 1px solid #e5c07b35;
    border-radius: 5px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .generate-dm-btn:hover {
    background: #e5c07b30;
  }

  .dm-draft-input {
    width: 100%;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 5px;
    color: #c0c0d8;
    font-size: 13px;
    padding: 8px;
    resize: vertical;
    font-family: inherit;
    line-height: 1.5;
    min-height: 80px;
  }

  .dm-draft-input:focus {
    outline: none;
    border-color: #e5c07b60;
  }

  .dm-actions {
    display: flex;
    gap: 6px;
  }

  .send-dm-btn {
    padding: 4px 10px;
    font-size: 12px;
    color: #e5c07b;
    background: #e5c07b18;
    border: 1px solid #e5c07b35;
    border-radius: 5px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .send-dm-btn:hover {
    background: #e5c07b30;
  }

  .action-row {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .action-btn {
    padding: 7px 16px;
    font-size: 13px;
    font-weight: 500;
    border-radius: 6px;
    background: #23233a;
    color: #888;
    border: 1px solid #2a2a3a;
    cursor: pointer;
    transition: all 0.15s;
  }

  .action-btn.comment {
    background: #98c37918;
    color: #98c379;
    border-color: #98c37935;
  }

  .action-btn.comment:hover,
  .action-btn.comment.active {
    background: #98c37930;
  }

  .action-btn.skip {
    background: #2a2a3a;
    color: #6b6b80;
    border-color: #3a3a50;
  }

  .action-btn.skip:hover,
  .action-btn.skip.active {
    background: #3a3a50;
    color: #9090a8;
  }

  .action-btn.reviewed.active {
    background: #61afef25;
    border-color: #61afef;
    color: #61afef;
  }

  .action-btn.star {
    border-color: #e5c07b50;
    color: #e5c07b;
  }

  .action-btn.star.active {
    background: #e5c07b25;
    border-color: #e5c07b;
    color: #e5c07b;
  }

  .action-btn.exclude {
    border-color: #e06c7550;
    color: #e06c75;
  }

  .action-btn.exclude.active {
    background: #e06c7525;
    border-color: #e06c75;
    color: #e06c75;
    opacity: 0.6;
  }

  .tooltip-wrap {
    position: relative;
    display: inline-flex;
  }

  .tooltip {
    position: absolute;
    bottom: 100%;
    left: 50%;
    transform: translateX(-50%);
    padding: 4px 10px;
    background: #2a2a3a;
    border: 1px solid #3a3a55;
    border-radius: 4px;
    color: #9090b0;
    font-size: 11px;
    white-space: nowrap;
    pointer-events: none;
    opacity: 0;
    transition: opacity 0.1s;
    margin-bottom: 6px;
  }

  .tooltip-wrap:hover .tooltip {
    opacity: 1;
  }

  .profile-signals {
    margin-top: 4px;
    border-top: 1px solid #2a2a3a;
    padding-top: 6px;
  }

  .profile-signals-summary {
    font-size: 11px;
    font-weight: 600;
    color: #6b6b80;
    cursor: pointer;
    user-select: none;
    list-style: none;
  }

  .profile-signals-summary::-webkit-details-marker {
    display: none;
  }

  .profile-signals-summary::before {
    content: '\25B6';
    display: inline-block;
    margin-right: 4px;
    font-size: 9px;
    transition: transform 0.15s;
  }

  .profile-signals[open] .profile-signals-summary::before {
    transform: rotate(90deg);
  }

  .profile-score {
    font-size: 10px;
    font-weight: 700;
    padding: 1px 5px;
    border-radius: 3px;
    margin-left: 6px;
    color: #6b6b80;
    background: #2a2a3a;
  }

  .profile-score.negative {
    color: #e06c75;
    background: #e06c7518;
  }

  .profile-score.positive {
    color: #98c379;
    background: #98c37918;
  }

  .profile-signals-grid {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 2px 10px;
    margin-top: 6px;
    padding: 6px 8px;
    background: #0f0f13;
    border-radius: 4px;
  }

  .ps-label {
    font-size: 11px;
    color: #4a4a60;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .ps-value {
    font-size: 12px;
    color: #9090a8;
    text-align: right;
  }

</style>
