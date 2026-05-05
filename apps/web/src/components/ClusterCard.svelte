<script>
  import { scoreColor } from '../lib/format.js';

  let { cluster, index, posts = [], onSelectAngle } = $props();

  let expanded = $state(false);

  let clusterPosts = $derived(posts.filter(p => (cluster.post_ids || []).includes(p.id)));
  let totalSignals = $derived((cluster.signals?.frustration || 0) + (cluster.signals?.seeking_solution || 0) + (cluster.signals?.workaround || 0));
</script>

<div class="cluster-card">
  <button class="cluster-header" onclick={() => expanded = !expanded}>
    <div class="cluster-header-left">
      <span class="cluster-rank">#{index + 1}</span>
      <div class="cluster-info">
        <span class="cluster-name">{cluster.name}</span>
        <div class="cluster-stats">
          <span>{cluster.post_count} posts</span>
          <span style="color: {scoreColor(cluster.avg_pain_score)}">
            avg pain: {cluster.avg_pain_score?.toFixed(1)}
          </span>
          {#if totalSignals > 0}
            <span class="signal-split">
              {#if cluster.signals?.frustration}
                <span class="sig-frust">{cluster.signals.frustration} frustration</span>
              {/if}
              {#if cluster.signals?.seeking_solution}
                <span class="sig-seek">{cluster.signals.seeking_solution} seeking</span>
              {/if}
            </span>
          {/if}
        </div>
      </div>
    </div>
    <span class="expand-icon">{expanded ? '▴' : '▾'}</span>
  </button>

  {#if expanded}
    <div class="cluster-body">
      {#if cluster.key_quotes?.length > 0}
        <div class="section">
          <h4 class="section-label">Key Quotes</h4>
          {#each cluster.key_quotes as quote}
            <blockquote class="quote">{quote}</blockquote>
          {/each}
        </div>
      {/if}

      {#if cluster.product_angle}
        <div class="section angle-section">
          <h4 class="section-label">Product Angle</h4>
          <div class="angle-card">
            <p class="angle-idea">{cluster.product_angle.idea}</p>
            <div class="angle-meta">
              <div class="angle-field">
                <span class="angle-label">Target Niche</span>
                <span>{cluster.product_angle.target_niche}</span>
              </div>
              <div class="angle-field">
                <span class="angle-label">Why</span>
                <span>{cluster.product_angle.why}</span>
              </div>
            </div>
            <button
              class="select-angle-btn"
              onclick={(e) => { e.stopPropagation(); onSelectAngle?.({ clusterIndex: index }); }}
            >
              Select This Angle
            </button>
          </div>
        </div>
      {/if}

      {#if clusterPosts.length > 0}
        <div class="section">
          <h4 class="section-label">Supporting Posts ({clusterPosts.length})</h4>
          <div class="supporting-posts">
            {#each clusterPosts as post}
              <div class="supporting-post">
                <div class="sp-header">
                  <span class="sp-platform" class:reddit={post.platform === 'reddit'} class:bluesky={post.platform === 'bluesky'}>
                    {post.platform}
                  </span>
                  <span class="sp-score" style="color: {scoreColor(post.final_score)}">{post.final_score?.toFixed(1)}</span>
                </div>
                <p class="sp-title">{post.title}</p>
                {#if post.body}
                  <p class="sp-body">{post.body.slice(0, 200)}{post.body.length > 200 ? '...' : ''}</p>
                {/if}
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .cluster-card {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    overflow: hidden;
  }

  .cluster-header {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
    background: none;
    border: none;
    color: #e2e2e8;
    cursor: pointer;
    text-align: left;
    transition: background 0.15s;
  }

  .cluster-header:hover {
    background: #1e1e2a;
  }

  .cluster-header-left {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }

  .cluster-rank {
    font-size: 14px;
    font-weight: 700;
    color: #7c6af5;
    min-width: 28px;
  }

  .cluster-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .cluster-name {
    font-size: 15px;
    font-weight: 600;
  }

  .cluster-stats {
    display: flex;
    gap: 12px;
    font-size: 12px;
    color: #666;
  }

  .sig-frust { color: #e06c75; }
  .sig-seek { color: #98c379; }

  .signal-split {
    display: flex;
    gap: 8px;
  }

  .expand-icon {
    color: #666;
    font-size: 14px;
    flex-shrink: 0;
  }

  .cluster-body {
    padding: 0 16px 16px;
    display: flex;
    flex-direction: column;
    gap: 20px;
    border-top: 1px solid #2a2a3a;
    padding-top: 16px;
  }

  .section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .section-label {
    font-size: 11px;
    font-weight: 600;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .quote {
    padding: 8px 12px;
    border-left: 3px solid #7c6af5;
    background: #0f0f13;
    color: #9090b0;
    font-size: 13px;
    line-height: 1.5;
    font-style: italic;
  }

  .angle-section {
    background: #15152a;
    padding: 16px;
    border-radius: 8px;
    border: 1px solid #2a2a50;
  }

  .angle-card {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .angle-idea {
    font-size: 15px;
    font-weight: 500;
    color: #e2e2e8;
    line-height: 1.4;
  }

  .angle-meta {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .angle-field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-size: 13px;
    color: #9090b0;
  }

  .angle-label {
    font-size: 11px;
    font-weight: 600;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .select-angle-btn {
    align-self: flex-start;
    padding: 8px 18px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .select-angle-btn:hover {
    opacity: 0.9;
  }

  .supporting-posts {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .supporting-post {
    padding: 10px 12px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
  }

  .sp-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;
  }

  .sp-platform {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .sp-platform.reddit { color: #FF4500; }
  .sp-platform.bluesky { color: #0085FF; }

  .sp-score {
    font-size: 14px;
    font-weight: 700;
  }

  .sp-title {
    font-size: 13px;
    font-weight: 500;
    color: #e2e2e8;
    margin-bottom: 4px;
  }

  .sp-body {
    font-size: 12px;
    color: #666;
    line-height: 1.4;
  }
</style>
