<script>
  let {
    post,
    selected = false,
    projectMode = 'marketing',
    bulkMode = false,
    checked = false,
    onSelect,
    onBulkToggle,
  } = $props();

  const PLATFORM_COLORS = {
    reddit: '#FF4500',
    bluesky: '#0085FF',
  };

  const STATUS_COLORS = {
    new: '#6b6b80',
    drafted: '#61afef',
    commented: '#98c379',
    skipped: '#3a3a50',
    reviewed: '#61afef',
    starred: '#e5c07b',
    excluded: '#e06c75',
  };

  const SIGNAL_COLORS = {
    frustration: { color: '#e06c75', bg: '#e06c7518', border: '#e06c7535' },
    workaround: { color: '#e5c07b', bg: '#e5c07b18', border: '#e5c07b35' },
    seeking_solution: { color: '#98c379', bg: '#98c37918', border: '#98c37935' },
  };

  const SIGNAL_LABELS = {
    frustration: 'Frustration',
    workaround: 'Workaround',
    seeking_solution: 'Seeking Solution',
  };

  function getExcerpt(post) {
    const text = post.body || post.title || '';
    if (!post.title && text.length > 80) {
      return text.slice(0, 80) + '...';
    }
    return post.title || text.slice(0, 80) + (text.length > 80 ? '...' : '');
  }

  function scoreColor(score) {
    if (score >= 9) return '#e5c07b';
    if (score >= 7) return '#98c379';
    if (score >= 5) return '#61afef';
    if (score >= 3) return '#d19a66';
    return '#6b6b80';
  }

  function toggleCheck(e) {
    e.stopPropagation();
    onBulkToggle?.({ id: post.id });
  }

  function formatRelative(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    const seconds = Math.floor((Date.now() - d.getTime()) / 1000);
    if (seconds < 60) return 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}d ago`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months}mo ago`;
    return `${Math.floor(months / 12)}y ago`;
  }

  // Flat field access - Scout API returns flat structure
  let platformColor = $derived(PLATFORM_COLORS[post.platform] || '#7c6af5');
  let statusColor = $derived(STATUS_COLORS[post.status] || '#6b6b80');
  let engagementType = $derived(post.engagement_type || 'product');
  let finalScore = $derived(post.final_score ?? post.post_score ?? 0);
  let upvotes = $derived(post.reddit_score ?? post.like_count ?? 0);
  let comments = $derived(post.num_comments ?? post.reply_count ?? 0);
  let dmCount = $derived(post.dm_targets?.length ?? 0);
  let signalType = $derived(post.signal_type || null);
  let signalStyle = $derived(SIGNAL_COLORS[signalType] || null);
  let excerpt = $derived(getExcerpt(post));
</script>

<div
  class="card"
  class:selected
  class:skipped={post.status === 'skipped'}
  class:excluded={post.status === 'excluded'}
  onclick={() => onSelect?.(post)}
  role="button"
  tabindex="0"
  onkeydown={(e) => e.key === 'Enter' && onSelect?.(post)}
>
  <div class="card-top">
    {#if projectMode === 'research'}
      <div
        class="bulk-checkbox"
        class:visible={bulkMode || checked}
        onclick={toggleCheck}
        role="checkbox"
        aria-checked={checked}
        tabindex="-1"
      >
        {#if checked}
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <rect width="16" height="16" rx="4" fill="#7c6af5"/>
            <path d="M4 8l3 3 5-5" stroke="#fff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        {:else}
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <rect x="0.5" y="0.5" width="15" height="15" rx="3.5" stroke="#4a4a60"/>
          </svg>
        {/if}
      </div>
    {/if}
    <div class="badges">
      <span class="platform-badge" style="background: {platformColor}20; color: {platformColor}; border-color: {platformColor}40">
        {post.platform === 'reddit' ? 'Reddit' : 'Bluesky'}
      </span>
      {#if projectMode === 'research' && signalStyle}
        <span class="type-badge" style="color: {signalStyle.color}; background: {signalStyle.bg}; border: 1px solid {signalStyle.border}">
          {SIGNAL_LABELS[signalType] || signalType}
        </span>
      {:else if projectMode === 'marketing'}
        {#if engagementType === 'product'}
          <span class="type-badge product">Product</span>
        {:else}
          <span class="type-badge karma">Karma</span>
        {/if}
      {/if}
      <span class="status-badge" style="color: {statusColor}; background: {statusColor}15; border-color: {statusColor}30">
        {post.status}
      </span>
      {#if projectMode === 'marketing' && dmCount > 0}
        <span class="dm-badge">{dmCount} DM</span>
      {/if}
    </div>
    <div class="scores">
      <span class="final-score" style="color: {scoreColor(finalScore)}">
        {finalScore.toFixed ? finalScore.toFixed(1) : finalScore}
      </span>
    </div>
  </div>

  <p class="excerpt">{excerpt}</p>

  <div class="card-footer">
    <div class="metrics">
      {#if upvotes > 0}
        <span class="metric">
          {#if post.platform === 'reddit'}
            &#9650; {upvotes}
          {:else}
            &#9825; {upvotes}
          {/if}
        </span>
      {/if}
      {#if comments > 0}
        <span class="metric">&#128172; {comments}</span>
      {/if}
    </div>
    <span class="date">{formatRelative(post.created_at)}</span>
  </div>
</div>

<style>
  .card {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s;
    user-select: none;
  }

  .card:hover {
    border-color: #3a3a55;
    background: #1e1e2a;
  }

  .card.selected {
    border-color: #7c6af5;
    background: #1e1e2e;
  }

  .card.skipped {
    opacity: 0.45;
  }

  .card.excluded {
    opacity: 0.45;
  }

  .card-top {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 8px;
    position: relative;
  }

  .badges {
    display: flex;
    align-items: center;
    gap: 5px;
    flex-wrap: wrap;
    transition: margin-left 0.15s;
  }

  .platform-badge {
    font-size: 10px;
    font-weight: 700;
    padding: 2px 7px;
    border-radius: 3px;
    border: 1px solid transparent;
    letter-spacing: 0.03em;
    text-transform: uppercase;
  }

  .type-badge {
    font-size: 10px;
    font-weight: 700;
    padding: 2px 6px;
    border-radius: 3px;
    letter-spacing: 0.03em;
    text-transform: uppercase;
  }

  .type-badge.product {
    color: #98c379;
    background: #98c37918;
    border: 1px solid #98c37935;
  }

  .type-badge.karma {
    color: #d19a66;
    background: #d19a6618;
    border: 1px solid #d19a6635;
  }

  .status-badge {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 3px;
    border: 1px solid transparent;
    letter-spacing: 0.03em;
    text-transform: uppercase;
  }

  .dm-badge {
    font-size: 10px;
    font-weight: 700;
    padding: 2px 6px;
    border-radius: 3px;
    letter-spacing: 0.03em;
    text-transform: uppercase;
    color: #e5c07b;
    background: #e5c07b18;
    border: 1px solid #e5c07b35;
  }

  .scores {
    flex-shrink: 0;
  }

  .final-score {
    font-size: 16px;
    font-weight: 700;
  }

  .excerpt {
    font-size: 13px;
    color: #9090a8;
    line-height: 1.4;
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .card-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .metrics {
    display: flex;
    gap: 8px;
  }

  .metric {
    font-size: 12px;
    color: #4a4a60;
  }

  .date {
    font-size: 11px;
    color: #3a3a50;
  }

  .bulk-checkbox {
    position: absolute;
    top: 0;
    left: 0;
    width: 16px;
    height: 16px;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .bulk-checkbox.visible {
    opacity: 1;
  }

  .card:hover .bulk-checkbox {
    opacity: 1;
  }

  .card:hover .bulk-checkbox + .badges,
  .bulk-checkbox.visible + .badges {
    margin-left: 22px;
  }
</style>
