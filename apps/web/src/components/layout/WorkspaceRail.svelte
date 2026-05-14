<script>
  import ProjectSelector from '../ProjectSelector.svelte';
  let { projects = [], selectedProjectId = null, view = 'posts', onSelectProject, onCreateProject, onNavigate } = $props();
</script>

<div class="rail-inner">
  <div class="rail-header">
    <img src="/logo.svg" alt="" class="rail-logo" />
    <span class="rail-brand">ThreadLens</span>
  </div>
  
  <div class="rail-project">
    <ProjectSelector
      {projects}
      selectedId={selectedProjectId}
      onSelect={onSelectProject}
      onCreate={onCreateProject}
    />
  </div>

  <nav class="rail-nav">
    <button class="nav-item" class:active={view === 'posts'} onclick={() => onNavigate('posts')}>Inbox</button>
    <button class="nav-item" class:active={view === 'reports'} onclick={() => onNavigate('reports')}>Reports</button>
    <button class="nav-item" class:active={view === 'sources'} onclick={() => onNavigate('settings')}>Sources</button>
    <button class="nav-item" class:active={view === 'settings'} onclick={() => onNavigate('settings')}>Settings</button>
    <button class="nav-item" class:active={view === 'models'} onclick={() => onNavigate('models')}>Models</button>
  </nav>
</div>

<style>
  .rail-inner { display: flex; flex-direction: column; gap: var(--space-16); padding: var(--space-16) 0; height: 100%; }
  .rail-header { display: flex; align-items: center; gap: var(--space-8); padding: 0 var(--space-16); }
  .rail-logo { width: 24px; height: 24px; }
  .rail-brand { font-weight: 600; font-size: 16px; color: var(--color-text-primary); }
  .rail-project { padding: 0 var(--space-16); }
  .rail-nav { display: flex; flex-direction: column; gap: 2px; padding: 0 var(--space-8); }
  .nav-item {
    background: transparent; border: none; color: var(--color-text-secondary); text-align: left;
    padding: var(--space-8) var(--space-8); border-radius: 6px; cursor: pointer; font-size: 13px;
  }
  .nav-item:hover { background: var(--color-surface-elevated); color: var(--color-text-primary); }
  .nav-item.active { background: var(--color-surface-elevated); color: var(--color-brand); font-weight: 500; }
</style>
