<script>
  import { projects as projectsApi } from '../lib/api.js';
  import PromptEditor from './PromptEditor.svelte';
  import ScheduleManager from './ScheduleManager.svelte';

  let {
    projectId,
    project,
    initialTab = 'general',
    onProjectUpdated,
    onProjectDeleted,
    onProjectCloned,
    onTabChange,
  } = $props();

  let activeTab = $state(initialTab);
  let editName = $state(project?.name || '');
  let savingGeneral = $state(false);
  let deleting = $state(false);
  let error = $state('');
  let editDescription = $state(project?.description || '');
  let cloning = $state(false);
  let cloneError = $state('');
  let graduating = $state(false);
  let graduateError = $state('');
  let showDeleteConfirm = $state(false);
  let deleteConfirmInput = $state('');

  const tabs = [
    { id: 'general', label: 'General' },
    { id: 'prompts', label: 'Prompts' },
    { id: 'schedules', label: 'Schedules' },
    { id: 'advanced', label: 'Advanced' },
  ];

  const validTabs = tabs.map(t => t.id);
  let lastProjectId = projectId;
  $effect(() => {
    if (!project) return;
    if (projectId !== lastProjectId) {
      lastProjectId = projectId;
      editName = project.name;
      editDescription = project.description || '';
    }
    if (!validTabs.includes(activeTab)) {
      activeTab = 'general';
    }
  });

  let generalDirty = $derived(editName !== (project?.name || '') || editDescription !== (project?.description || ''));
  let projectSlug = $derived((project?.name || '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, ''));
  let deleteConfirmed = $derived(deleteConfirmInput === projectSlug);

  async function saveGeneral() {
    if (!editName.trim()) return;
    savingGeneral = true;
    error = '';
    try {
      const updates = {};
      if (editName.trim() !== project?.name) updates.name = editName.trim();
      if (editDescription !== (project?.description || '')) updates.description = editDescription;
      if (Object.keys(updates).length === 0) return;
      const updated = await projectsApi.update(projectId, updates);
      onProjectUpdated?.(updated);
    } catch (e) {
      error = e.message;
    } finally {
      savingGeneral = false;
    }
  }

  function cancelDelete() {
    showDeleteConfirm = false;
    deleteConfirmInput = '';
  }

  async function cloneAsResearch() {
    cloning = true;
    cloneError = '';
    try {
      const cloneId = `${projectId}-research-${Date.now().toString(36)}`;
      const cloneName = `${project?.name || 'Project'} - Research`;
      const cloned = await projectsApi.clone(projectId, { id: cloneId, name: cloneName });
      onProjectCloned?.(cloned);
    } catch (e) {
      cloneError = e.message;
    } finally {
      cloning = false;
    }
  }

  async function graduateToMarketing() {
    graduating = true;
    graduateError = '';
    try {
      const updated = await projectsApi.graduate(projectId);
      onProjectUpdated?.(updated);
    } catch (e) {
      graduateError = e.message;
    } finally {
      graduating = false;
    }
  }

  async function deleteProject() {
    if (!deleteConfirmed) return;
    deleting = true;
    error = '';
    try {
      await projectsApi.delete(projectId);
      onProjectDeleted?.({ id: projectId });
    } catch (e) {
      error = e.message;
      deleting = false;
    }
  }
</script>

<div class="project-settings">
  <div class="settings-header">
    <h2 class="settings-title">
      Project Settings
      <a class="doc-link" href="https://docs.threadlens.dev/user-guide/projects-and-modes/" target="_blank" rel="noopener" title="How projects and modes work">?</a>
    </h2>
    {#if project}
      <span class="project-name-label">{project.name}</span>
    {/if}
  </div>

  <div class="tab-bar">
    {#each tabs as tab}
      {#if tab.id !== 'prompts' || project?.mode === 'marketing'}
        <button
          class="tab-btn"
          class:active={activeTab === tab.id}
          onclick={() => { activeTab = tab.id; onTabChange?.(tab.id); }}
        >
          {tab.label}
        </button>
      {:else if tab.id === 'prompts'}
        <button
          class="tab-btn disabled"
          title="Prompts are only available in marketing mode"
          disabled
        >
          {tab.label}
          <span class="tab-badge">marketing only</span>
        </button>
      {/if}
    {/each}
  </div>

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  <div class="tab-content">
    {#if activeTab === 'general'}
      <div class="general-tab">
        <div class="field-group">
          <label class="field-label" for="project-name">Project Name</label>
          <div class="field-row">
            <input
              id="project-name"
              class="field-input"
              type="text"
              bind:value={editName}
              disabled={savingGeneral}
            />
          </div>
        </div>

        <div class="field-group">
          <label class="field-label" for="project-description">Description</label>
          <textarea
            id="project-description"
            class="field-textarea"
            bind:value={editDescription}
            disabled={savingGeneral}
            placeholder="What are you researching? This helps the scorer evaluate post relevance."
            rows="3"
          ></textarea>
          <p class="field-hint">Describes the project's focus area for AI scoring context.</p>
        </div>

        <div class="save-row">
           <button class="save-btn"
            onclick={saveGeneral}
            disabled={!generalDirty || savingGeneral}
          >
            {savingGeneral ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>

    {:else if activeTab === 'advanced'}
      <div class="general-tab">
        <div class="field-group">
          <div class="field-label">
            Mode
            <a class="doc-link" href="https://docs.threadlens.dev/user-guide/projects-and-modes/" target="_blank" rel="noopener" title="Research vs marketing mode">?</a>
          </div>
          <div class="mode-display">
            <span class="mode-badge" class:research={project?.mode === 'research'} class:marketing={project?.mode === 'marketing'}>
              {project?.mode === 'research' ? 'Research' : 'Marketing'}
            </span>
            <span class="mode-hint">
              {project?.mode === 'research'
                ? 'Analyze pain points and discover product angles. Select an angle from a report to graduate to marketing mode.'
                : 'Engage with leads, manage replies, and grow your audience.'}
            </span>
          </div>
        </div>

        {#if project?.mode === 'research' && project?.selected_report_id}
          <div class="field-group">
            <div class="field-label">
              Graduate to Marketing
              <a class="doc-link" href="https://docs.threadlens.dev/user-guide/projects-and-modes/#marketing-mode" target="_blank" rel="noopener" title="Switch from research discovery to engagement workflows">?</a>
            </div>
            <p class="field-hint">You have selected a product angle. Review the generated marketing config, then graduate this project to marketing mode.</p>
            {#if graduateError}
              <div class="error-msg">{graduateError}</div>
            {/if}
            <button class="graduate-btn" onclick={graduateToMarketing} disabled={graduating}>
              {graduating ? 'Graduating...' : 'Switch to Marketing Mode'}
            </button>
          </div>
        {/if}

        {#if project?.mode === 'marketing'}
          <div class="field-group">
            <div class="field-label">Re-Research</div>
            <p class="field-hint">Clone this project as a new research project with the same queries but fresh data.</p>
            {#if cloneError}
              <div class="error-msg">{cloneError}</div>
            {/if}
            <button class="clone-btn" onclick={cloneAsResearch} disabled={cloning}>
              {cloning ? 'Cloning...' : 'Clone as Research Project'}
            </button>
          </div>
        {/if}

        <div class="danger-zone">
          <div class="danger-header">
            <h3 class="danger-title">Danger Zone</h3>
          </div>
          <div class="danger-body">
            <div class="danger-row">
              <div>
                <div class="danger-action-title">Delete this project</div>
                <div class="danger-action-desc">Permanently removes the project and all associated data.</div>
              </div>
              {#if !showDeleteConfirm}
                <button
                  class="delete-project-btn"
                  onclick={() => showDeleteConfirm = true}
                >
                  Delete Project
                </button>
              {/if}
            </div>
            {#if showDeleteConfirm}
              <div class="delete-confirm">
                <p class="delete-confirm-text">
                  Type <strong>{projectSlug}</strong> to confirm deletion:
                </p>
                <div class="delete-confirm-row">
                  <input
                    class="delete-confirm-input"
                    type="text"
                    bind:value={deleteConfirmInput}
                    placeholder={projectSlug}
                  />
                   <button
                     class="delete-project-btn delete-confirm-btn"
                     onclick={deleteProject}
                    disabled={!deleteConfirmed || deleting}
                  >
                    {deleting ? 'Deleting...' : 'Confirm Delete'}
                  </button>
                   <button
                     class="cancel-delete-btn"
                     onclick={cancelDelete}
                    disabled={deleting}
                  >
                    Cancel
                  </button>
                </div>
              </div>
            {/if}
          </div>
        </div>
      </div>

    {:else if activeTab === 'prompts'}
      <PromptEditor {projectId} />

    {:else if activeTab === 'schedules'}
      <ScheduleManager {projectId} />
    {/if}
  </div>
</div>

<style>
  .project-settings {
    max-width: 800px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  .settings-header {
    display: flex;
    align-items: baseline;
    gap: 12px;
    margin-bottom: 20px;
  }

  .settings-title {
    font-size: 20px;
    font-weight: 700;
    color: #e2e2e8;
  }

  .project-name-label {
    font-size: 14px;
    color: #666;
  }

  .tab-bar {
    display: flex;
    gap: 2px;
    border-bottom: 1px solid #2a2a3a;
    margin-bottom: 24px;
  }

  .tab-btn {
    padding: 8px 16px;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    color: #888;
    font-size: 14px;
    cursor: pointer;
    transition: color 0.15s, border-color 0.15s;
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: -1px;
  }

  .tab-btn:hover:not(:disabled) {
    color: #e2e2e8;
  }

  .tab-btn.active {
    color: #e2e2e8;
    border-bottom-color: #7c6af5;
  }

  .tab-btn.disabled,
  .tab-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .tab-badge {
    font-size: 10px;
    padding: 1px 5px;
    background: #2a2a3a;
    border-radius: 3px;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
    margin-bottom: 16px;
  }

  .tab-content {
    flex: 1;
  }

  /* General tab */
  .general-tab {
    display: flex;
    flex-direction: column;
    gap: 28px;
  }

  .field-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .field-label {
    font-size: 13px;
    font-weight: 600;
    color: #aaa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .field-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .field-input {
    flex: 1;
    max-width: 360px;
    padding: 9px 12px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 14px;
    transition: border-color 0.15s;
  }

  .field-input:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .field-input:disabled {
    opacity: 0.6;
  }

  .field-textarea {
    width: 100%;
    max-width: 500px;
    padding: 9px 12px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 14px;
    font-family: inherit;
    line-height: 1.5;
    resize: vertical;
    transition: border-color 0.15s;
  }

  .field-textarea:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .field-textarea:disabled {
    opacity: 0.6;
  }

  .field-hint {
    font-size: 12px;
    color: #555;
  }

  .save-row {
    display: flex;
    justify-content: flex-start;
  }

  .save-btn {
    padding: 9px 20px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .save-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .save-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .mode-display {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .mode-badge {
    display: inline-block;
    padding: 6px 14px;
    border-radius: 6px;
    font-size: 14px;
    font-weight: 600;
    width: fit-content;
  }

  .mode-badge.research {
    background: #61afef20;
    color: #61afef;
    border: 1px solid #61afef40;
  }

  .mode-badge.marketing {
    background: #98c37920;
    color: #98c379;
    border: 1px solid #98c37940;
  }

  .mode-hint {
    font-size: 12px;
    color: #666;
    line-height: 1.4;
  }

  .graduate-btn {
    padding: 9px 20px;
    background: #98c379;
    border: none;
    border-radius: 6px;
    color: #1a1a24;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
    width: fit-content;
  }

  .graduate-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .graduate-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .clone-btn {
    padding: 9px 20px;
    background: #23233a;
    border: 1px solid #7c6af5;
    border-radius: 6px;
    color: #7c6af5;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
    width: fit-content;
  }

  .clone-btn:hover:not(:disabled) {
    background: #2a2a45;
  }

  .clone-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Danger zone */
  .danger-zone {
    border: 1px solid #5a2020;
    border-radius: 8px;
    overflow: hidden;
  }

  .danger-header {
    padding: 12px 16px;
    background: #2a1515;
    border-bottom: 1px solid #5a2020;
  }

  .danger-title {
    font-size: 13px;
    font-weight: 600;
    color: #f87171;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .danger-body {
    padding: 16px;
    background: #1a1010;
  }

  .danger-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .danger-action-title {
    font-size: 14px;
    font-weight: 500;
    color: #e2e2e8;
    margin-bottom: 2px;
  }

  .danger-action-desc {
    font-size: 13px;
    color: #888;
  }

  .delete-project-btn {
    flex-shrink: 0;
    padding: 8px 16px;
    background: none;
    border: 1px solid #f87171;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s;
    white-space: nowrap;
  }

  .delete-project-btn:hover:not(:disabled) {
    background: #3a1515;
  }

  .delete-project-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .delete-confirm {
    margin-top: 14px;
    padding-top: 14px;
    border-top: 1px solid #5a2020;
  }

  .delete-confirm-text {
    font-size: 13px;
    color: #ccc;
    margin-bottom: 10px;
  }

  .delete-confirm-text strong {
    color: #f87171;
    font-family: monospace;
    font-size: 14px;
  }

  .delete-confirm-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .delete-confirm-input {
    flex: 1;
    max-width: 260px;
    padding: 8px 12px;
    background: #1a1010;
    border: 1px solid #5a2020;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 14px;
    font-family: monospace;
  }

  .delete-confirm-input:focus {
    outline: none;
    border-color: #f87171;
  }

  .delete-confirm-btn {
    background: #f87171;
    border-color: #f87171;
    color: #fff;
  }

  .delete-confirm-btn:hover:not(:disabled) {
    background: #ef4444;
  }

  .cancel-delete-btn {
    padding: 8px 16px;
    background: none;
    border: 1px solid #444;
    border-radius: 6px;
    color: #888;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .cancel-delete-btn:hover:not(:disabled) {
    color: #e2e2e8;
    border-color: #666;
  }

  .doc-link {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    font-size: 9px;
    font-weight: 700;
    color: #4a4a60;
    background: #2a2a3a;
    border-radius: 50%;
    text-decoration: none;
    margin-left: 5px;
    vertical-align: middle;
    transition: color 0.15s, background 0.15s;
    cursor: help;
  }
  .doc-link:hover {
    color: #61afef;
    background: #61afef20;
  }
</style>
