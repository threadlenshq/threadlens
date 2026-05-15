<script>
  import ProjectSelector from '../ProjectSelector.svelte';
  let { projects = [], selectedProjectId = null, view = 'posts', collapsed = false, onToggleCollapse, onSelectProject, onCreateProject, onNavigate } = $props();

  const navItems = [
    {
      id: 'posts',
      label: 'Inbox',
      icon: `<path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"></path><polyline points="22,6 12,13 2,6"></polyline>`,
    },
    {
      id: 'reports',
      label: 'Reports',
      icon: `<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line>`,
    },
    {
      id: 'sources',
      label: 'Sources',
      icon: `<ellipse cx="12" cy="5" rx="9" ry="3"></ellipse><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"></path><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"></path>`,
    },
    {
      id: 'settings',
      label: 'Settings',
      icon: `<circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>`,
    },
    {
      id: 'models',
      label: 'Models',
      icon: `<rect x="4" y="4" width="16" height="16" rx="2" ry="2"></rect><rect x="9" y="9" width="6" height="6"></rect><line x1="9" y1="1" x2="9" y2="4"></line><line x1="15" y1="1" x2="15" y2="4"></line><line x1="9" y1="20" x2="9" y2="23"></line><line x1="15" y1="20" x2="15" y2="23"></line><line x1="20" y1="9" x2="23" y2="9"></line><line x1="20" y1="14" x2="23" y2="14"></line><line x1="1" y1="9" x2="4" y2="9"></line><line x1="1" y1="14" x2="4" y2="14"></line>`,
    },
  ];
</script>

<div class="rail-inner" class:collapsed>
  <!-- Header: logo + brand + collapse toggle -->
  <div class="rail-header">
    <img src="/logo.svg" alt="" class="rail-logo" />
    {#if !collapsed}
      <span class="rail-brand">ThreadLens</span>
      <button class="collapse-btn" onclick={onToggleCollapse} title="Collapse sidebar">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="18" height="18" rx="2"></rect>
          <line x1="9" y1="3" x2="9" y2="21"></line>
        </svg>
      </button>
    {:else}
      <button class="collapse-btn expand-btn" onclick={onToggleCollapse} title="Expand sidebar">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="18" height="18" rx="2"></rect>
          <line x1="15" y1="3" x2="15" y2="21"></line>
        </svg>
      </button>
    {/if}
  </div>

  <!-- Project selector -->
  <div class="rail-project">
    <ProjectSelector
      {projects}
      selectedId={selectedProjectId}
      onSelect={onSelectProject}
      onCreate={onCreateProject}
      {collapsed}
    />
  </div>

  <!-- Nav items -->
  <nav class="rail-nav">
    {#each navItems as item}
      <div class="nav-item-wrap">
        <button
          type="button"
          class="nav-item"
          class:active={view === item.id}
          onclick={() => onNavigate(item.id)}
        >
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            {@html item.icon}
          </svg>
          {#if !collapsed}
            <span class="nav-label">{item.label}</span>
          {/if}
        </button>
        {#if collapsed}
          <div class="nav-tooltip">{item.label}</div>
        {/if}
      </div>
    {/each}
  </nav>
</div>

<style>
  /* ---------- Shell ---------- */
  .rail-inner {
    display: flex;
    flex-direction: column;
    gap: var(--space-16);
    padding: var(--space-16) 0;
    height: 100%;
    overflow: visible;
    width: 100%;
  }

  /* ---------- Header ---------- */
  .rail-header {
    display: flex;
    align-items: center;
    gap: var(--space-8);
    padding: 0 var(--space-12, 12px);
    min-height: 28px;
  }

  .rail-logo {
    width: 24px;
    height: 24px;
    flex-shrink: 0;
  }

  .rail-brand {
    flex: 1;
    font-weight: 600;
    font-size: 15px;
    color: var(--color-text-primary);
    white-space: nowrap;
    overflow: hidden;
  }

  .collapse-btn {
    margin-left: auto;
    background: transparent;
    border: none;
    color: var(--color-text-secondary);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background 0.15s, color 0.15s;
  }

  .collapse-btn:hover {
    background: var(--color-surface-hover, rgba(255,255,255,0.06));
    color: var(--color-text-primary);
  }

  /* When collapsed, centre everything in header */
  .collapsed .rail-header {
    flex-direction: column;
    align-items: center;
    padding: 0;
    gap: var(--space-8);
  }

  .collapsed .expand-btn {
    margin-left: 0;
  }

  /* ---------- Project selector ---------- */
  .rail-project {
    padding: 0 var(--space-8);
  }

  .collapsed .rail-project {
    display: flex;
    justify-content: center;
    padding: 0;
    width: 100%;
  }

  /* ---------- Nav ---------- */
  .rail-nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 0 var(--space-8);
  }

  .collapsed .rail-nav {
    padding: 0;
    align-items: center;
    width: 100%;
  }

  .nav-item-wrap {
    position: relative;
    width: 100%;
  }

  .collapsed .nav-item-wrap {
    display: flex;
    justify-content: center;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: var(--space-8);
    width: 100%;
    background: transparent;
    border: none;
    color: var(--color-text-secondary);
    text-align: left;
    padding: 7px var(--space-8);
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    min-height: 34px;
    white-space: nowrap;
    transition: background 0.15s, color 0.15s;
  }

  .collapsed .nav-item {
    width: 36px;
    height: 36px;
    min-height: 0;
    padding: 0;
    justify-content: center;
  }

  .nav-item:hover { background: var(--color-surface-elevated); color: var(--color-text-primary); }
  .nav-item.active { background: var(--color-surface-elevated); color: var(--color-brand); font-weight: 500; }

  .nav-icon {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
  }

  .nav-label {
    flex: 1;
  }

  /* ---------- Tooltip ---------- */
  .nav-tooltip {
    pointer-events: none;
    position: absolute;
    left: calc(100% + 10px);
    top: 50%;
    transform: translateY(-50%);
    background: var(--color-surface-elevated, #2a2a3a);
    color: var(--color-text-primary, #e2e2e8);
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
    padding: 5px 10px;
    border-radius: 5px;
    border: 1px solid var(--color-border, #3a3a4a);
    box-shadow: 0 4px 12px rgba(0,0,0,0.35);
    opacity: 0;
    transition: opacity 0.12s ease;
    z-index: 200;
  }

  /* tiny left arrow */
  .nav-tooltip::before {
    content: '';
    position: absolute;
    right: 100%;
    top: 50%;
    transform: translateY(-50%);
    border: 5px solid transparent;
    border-right-color: var(--color-border, #3a3a4a);
  }

  .nav-tooltip::after {
    content: '';
    position: absolute;
    right: calc(100% - 1px);
    top: 50%;
    transform: translateY(-50%);
    border: 5px solid transparent;
    border-right-color: var(--color-surface-elevated, #2a2a3a);
  }

  .nav-item-wrap:hover .nav-tooltip {
    opacity: 1;
  }
</style>
