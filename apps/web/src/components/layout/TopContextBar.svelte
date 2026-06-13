<script>
  let { view = '', projectName = '', children } = $props();

  let showProjectCrumb = $derived(view !== 'models' && view !== 'privacy');
  let normalizedView = $derived(view ? view.charAt(0).toUpperCase() + view.slice(1) : '');
</script>

<div class="topbar-inner">
  <div class="breadcrumbs">
    {#if showProjectCrumb}
      <span class="project-name">{projectName || 'No Project'}</span>
      <span class="divider">/</span>
    {/if}
    <span class="view-name">{normalizedView}</span>
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
