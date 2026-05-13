<script>
  let { projects = [], selectedId = null, onSelect, onCreate } = $props();

  let open = $state(false);
  let showForm = $state(false);
  let newSlug = $state('');
  let newName = $state('');
  let newMode = $state('research');

  let selectedProject = $derived(projects.find(p => p.id === selectedId) || null);

  function toggle() {
    open = !open;
    if (!open) showForm = false;
  }

  function selectProject(id) {
    onSelect?.(id);
    open = false;
    showForm = false;
  }

  function openForm() {
    showForm = true;
  }

  function cancelForm() {
    showForm = false;
    newSlug = '';
    newName = '';
    newMode = 'research';
  }

  function submitForm() {
    if (!newSlug.trim() || !newName.trim()) return;
    onCreate?.({ id: newSlug.trim(), name: newName.trim(), mode: newMode });
    cancelForm();
    open = false;
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      open = false;
      showForm = false;
    }
  }

  function modeBadgeClass(mode) {
    return mode === 'research' ? 'badge badge-research' : 'badge badge-marketing';
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="project-selector" data-testid="project-selector">
  <button class="selector-btn" onclick={toggle}>
    {#if selectedProject}
      <span class="project-name">{selectedProject.name}</span>
      <span class={modeBadgeClass(selectedProject.mode)}>{selectedProject.mode}</span>
    {:else}
      <span class="placeholder">Select Project</span>
    {/if}
    <span class="chevron" class:rotated={open}>&#8964;</span>
  </button>

  {#if open}
    <div class="dropdown">
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

      {#if !showForm}
        <button class="new-project-btn" onclick={openForm}>+ New Project</button>
      {:else}
        <div class="new-project-form">
          <input
            class="form-input"
            type="text"
            placeholder="slug (e.g. my-project)"
            bind:value={newSlug}
          />
          <input
            class="form-input"
            type="text"
            placeholder="Display name"
            bind:value={newName}
          />
          <select class="form-select" bind:value={newMode}>
            <option value="research">Research</option>
            <option value="marketing">Marketing</option>
          </select>
          <div class="form-actions">
            <button class="btn-create" onclick={submitForm}>Create</button>
            <button class="btn-cancel" onclick={cancelForm}>Cancel</button>
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .project-selector {
    position: relative;
  }

  .selector-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    padding: 6px 12px;
    color: #e2e2e8;
    cursor: pointer;
    font-size: 14px;
    min-width: 180px;
    max-width: 250px;
    transition: border-color 0.15s;
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
    font-size: 12px;
    color: #888;
    transition: transform 0.15s;
    display: inline-block;
  }

  .chevron.rotated {
    transform: rotate(180deg);
  }

  .dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    min-width: 220px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 4px;
    z-index: 100;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
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

  .new-project-form {
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .form-input,
  .form-select {
    width: 100%;
    padding: 6px 8px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 4px;
    color: #e2e2e8;
    font-size: 13px;
    outline: none;
    transition: border-color 0.15s;
  }

  .form-input:focus,
  .form-select:focus {
    border-color: #7c6af5;
  }

  .form-select option {
    background: #1a1a24;
  }

  .form-actions {
    display: flex;
    gap: 6px;
  }

  .btn-create {
    flex: 1;
    padding: 6px;
    background: #7c6af5;
    border: none;
    border-radius: 4px;
    color: #fff;
    font-size: 13px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-create:hover {
    background: #6a58e3;
  }

  .btn-cancel {
    flex: 1;
    padding: 6px;
    background: #2a2a3a;
    border: none;
    border-radius: 4px;
    color: #e2e2e8;
    font-size: 13px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-cancel:hover {
    background: #33334a;
  }
</style>
