<script>
  let { view = '', projectName = '', children } = $props();
</script>

<div class="topbar-inner">
  <div class="breadcrumbs">
    <span class="project-name">{projectName || 'No Project'}</span>
    <span class="divider">/</span>
    <span class="view-name">{view.charAt(0).toUpperCase() + view.slice(1)}</span>
  </div>
  <div class="actions">
    {@render children?.()}
  </div>
</div>

<style>
  .topbar-inner { display: flex; align-items: center; justify-content: space-between; width: 100%; }
  .breadcrumbs { display: flex; align-items: center; gap: var(--space-8); font-size: 14px; }
  .project-name { color: var(--color-text-primary); font-weight: 500; }
  .divider { color: var(--color-border); }
  .view-name { color: var(--color-text-secondary); }
  .actions { display: flex; align-items: center; gap: var(--space-12); }
</style>
