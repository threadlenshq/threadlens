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
  .topbar-inner { display: flex; align-items: center; justify-content: space-between; width: 100%; gap: var(--space-12); min-width: 0; }
  .breadcrumbs { display: flex; align-items: center; gap: var(--space-8); font-size: 14px; min-width: 0; flex: 1; }
  .project-name { color: var(--color-text-primary); font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; min-width: 0; }
  .divider { color: var(--color-border); flex-shrink: 0; }
  .view-name { color: var(--color-text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; min-width: 0; }
  .actions { display: flex; align-items: center; gap: var(--space-12); min-width: 0; flex-shrink: 0; }

  @media (max-width: 767px) {
    .topbar-inner { flex-wrap: wrap; gap: var(--space-8); }
    .breadcrumbs { flex: 1 1 100%; min-width: 0; }
    .actions { flex: 1 1 100%; justify-content: stretch; min-width: 0; width: 100%; }
  }
</style>
