<script>
  import { fly } from 'svelte/transition';

  let { projects = [], selectedId = null, onSelect, onRequestCreate, collapsed = false } = $props();

  let open = $state(false);
  let container = $state(null);

  let selectedProject = $derived(projects.find(p => p.id === selectedId) || null);

  function toggle() {
    open = !open;
  }

  function close() {
    open = false;
  }

  function selectProject(id) {
    onSelect?.(id);
    close();
  }

  function requestCreate() {
    close();
    onRequestCreate?.();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      close();
    }
  }

  function handleOutsideClick(e) {
    if (open && container && !container.contains(e.target)) {
      close();
    }
  }

  function modeBadgeClass(mode) {
    return mode === 'research' ? 'badge badge-research' : 'badge badge-marketing';
  }
</script>

<svelte:window onkeydown={handleKeydown} onclick={handleOutsideClick} />

<div class="project-selector" data-testid="project-selector" class:collapsed bind:this={container}>
  <button class="selector-btn" onclick={toggle} title={collapsed && selectedProject ? selectedProject.name : "Select Project"}>
    {#if collapsed}
      <span class="project-initial">
        {#if selectedProject}
          {selectedProject.name.charAt(0).toUpperCase()}
        {:else}
          +
        {/if}
      </span>
    {:else}
      {#if selectedProject}
        <span class="project-name">{selectedProject.name}</span>
        <span class={modeBadgeClass(selectedProject.mode)}>{selectedProject.mode}</span>
      {:else}
        <span class="placeholder">Select Project</span>
      {/if}
      <span class="chevron" class:rotated={open}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
      </span>
    {/if}
  </button>

  {#if open}
    <div 
      class="dropdown dropdown-content"
      transition:fly={{ x: collapsed ? -10 : 0, y: collapsed ? 0 : -10, duration: 200 }}
    >
      {#each projects as project (project.id)}
        <button
          class="project-item"
          class:active={project.id === selectedId}
          onclick={() => selectProject(project.id)}
        >
          <span class="item-name">{project.name}</span>
          <span class={modeBadgeClass(project.mode)}>{project.mode}</span>
        </button>
      {/each}

      {#if projects.length === 0}
        <div class="empty-state">No projects yet</div>
      {/if}

      <div class="divider"></div>

      <button class="new-project-btn" onclick={requestCreate}>+ New Project</button>
    </div>
  {/if}
</div>

<style>
  .project-selector {
    position: relative;
    width: 100%;
  }

  .project-selector.collapsed {
    width: auto;
  }

  .selector-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    padding: 6px 12px;
    color: #e2e2e8;
    cursor: pointer;
    font-size: 14px;
    min-width: 180px;
    width: 100%;
    transition: border-color 0.15s;
  }

  .project-selector.collapsed .selector-btn {
    min-width: 0;
    width: 32px;
    height: 32px;
    padding: 0;
    border-radius: 6px;
  }

  .project-initial {
    font-weight: 600;
    font-size: 14px;
    color: #e2e2e8;
  }

  .selector-btn:hover {
    border-color: #7c6af5;
  }

  .project-name {
    flex: 1;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .placeholder {
    flex: 1;
    text-align: left;
    color: #666;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .chevron {
    color: #888;
    transition: transform 0.15s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .chevron.rotated {
    transform: rotate(180deg);
  }

  .dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    width: 100%;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 4px;
    z-index: 200;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  }

  /* Collapsed: fly out to the right of the 64px rail */
  .project-selector.collapsed .dropdown {
    top: 0;
    left: calc(100% + 10px);
    width: 240px;
  }

  .project-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 8px 10px;
    background: none;
    border: none;
    border-radius: 5px;
    color: #e2e2e8;
    cursor: pointer;
    font-size: 14px;
    text-align: left;
    transition: background 0.1s;
  }

  .project-item:hover {
    background: #23233a;
  }

  .project-item.active {
    background: #2a2a45;
  }

  .item-name {
    flex: 1;
  }

  .badge {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 10px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .badge-marketing {
    background: rgba(124, 106, 245, 0.2);
    color: #7c6af5;
  }

  .badge-research {
    background: rgba(80, 200, 120, 0.2);
    color: #50c878;
  }

  .empty-state {
    padding: 8px 10px;
    color: #666;
    font-size: 13px;
  }

  .divider {
    height: 1px;
    background: #2a2a3a;
    margin: 4px 0;
  }

  .new-project-btn {
    width: 100%;
    padding: 8px 10px;
    background: none;
    border: none;
    border-radius: 5px;
    color: #7c6af5;
    cursor: pointer;
    font-size: 14px;
    text-align: left;
    transition: background 0.1s;
  }

  .new-project-btn:hover {
    background: #23233a;
  }
</style>
