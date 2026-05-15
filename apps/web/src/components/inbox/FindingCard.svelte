<script>
  import PostCard from '../PostCard.svelte';
  let { post, selected = false, projectMode = 'marketing', bulkMode = false, checked = false, onSelect, onBulkToggle } = $props();
</script>
<div class="finding-card-wrapper">
  <PostCard {post} {selected} {projectMode} {bulkMode} {checked} {onSelect} {onBulkToggle} />
</div>
<style>
  .finding-card-wrapper {
    margin-bottom: var(--space-8);
  }
  .finding-card-wrapper :global(.card) {
    background: var(--color-surface);
    border-color: var(--color-border);
    border-radius: 6px;
  }
  .finding-card-wrapper :global(.card.selected) {
    border-color: var(--color-brand);
    background: var(--color-surface-elevated);
  }
  .finding-card-wrapper :global(.card:hover) {
    border-color: var(--color-border-hover);
  }
</style>
