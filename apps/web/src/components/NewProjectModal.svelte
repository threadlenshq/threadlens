<script>
  import Modal from './ui/Modal.svelte';

  let { open = false, onClose, onCreate } = $props();

  let name = $state('');
  let slug = $state('');
  let slugManuallyEdited = $state(false);
  let description = $state('');
  let mode = $state('research');
  let submitting = $state(false);
  let error = $state('');

  // Auto-derive slug from name unless user has manually edited it
  $effect(() => {
    if (!slugManuallyEdited) {
      slug = name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
    }
  });

  // Reset when modal opens
  $effect(() => {
    if (open) {
      name = '';
      slug = '';
      slugManuallyEdited = false;
      description = '';
      mode = 'research';
      submitting = false;
      error = '';
    }
  });

  function handleSlugInput(e) {
    slugManuallyEdited = true;
    slug = e.target.value;
  }

  let canSubmit = $derived(name.trim().length > 0 && slug.trim().length > 0 && !submitting);

  async function handleSubmit() {
    if (!canSubmit) return;
    submitting = true;
    error = '';
    try {
      await onCreate?.({
        id: slug.trim(),
        name: name.trim(),
        description: description.trim() || undefined,
        mode,
      });
      onClose?.();
    } catch (e) {
      error = e.message || 'Failed to create project.';
      submitting = false;
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleSubmit();
  }
</script>

<Modal {open} title="New Project" onClose={onClose}>
  <form class="form" onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
    <div class="field-group">
      <label class="field-label" for="np-name">Project Name <span class="required">*</span></label>
      <input
        id="np-name"
        class="field-input"
        type="text"
        placeholder="e.g. Pain Point Research"
        bind:value={name}
        disabled={submitting}
      />
    </div>

    <div class="field-group">
      <label class="field-label" for="np-slug">Project ID (slug)</label>
      <input
        id="np-slug"
        class="field-input slug-input"
        type="text"
        placeholder="auto-derived from name"
        value={slug}
        oninput={handleSlugInput}
        disabled={submitting}
      />
      <p class="field-hint">Used as a unique identifier. Auto-derived from the name.</p>
    </div>

    <div class="field-group">
      <label class="field-label" for="np-desc">Description</label>
      <textarea
        id="np-desc"
        class="field-textarea"
        placeholder="What are you researching? This helps the scorer evaluate post relevance."
        bind:value={description}
        disabled={submitting}
        rows="3"
      ></textarea>
      <p class="field-hint">Describes the project's focus area for AI scoring context.</p>
    </div>

    <div class="field-group">
      <label class="field-label" for="np-mode">Mode</label>
      <select id="np-mode" class="field-select" bind:value={mode} disabled={submitting}>
        <option value="research">Research — identify pain points &amp; product angles</option>
        <option value="marketing">Marketing — engage leads &amp; grow audience</option>
      </select>
    </div>

    {#if error}
      <div class="error-msg">{error}</div>
    {/if}
  </form>

  {#snippet footer()}
    <button class="btn-cancel" onclick={onClose} disabled={submitting}>Cancel</button>
    <button class="btn-create" onclick={handleSubmit} disabled={!canSubmit}>
      {submitting ? 'Creating...' : 'Create Project'}
    </button>
  {/snippet}
</Modal>

<style>
  .form {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .field-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .field-label {
    font-size: 12px;
    font-weight: 600;
    color: #aaa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .required {
    color: #f87171;
  }

  .field-input,
  .field-select {
    width: 100%;
    padding: 9px 12px;
    background: #0f0f18;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 14px;
    outline: none;
    transition: border-color 0.15s;
    box-sizing: border-box;
  }

  .field-input:focus,
  .field-select:focus {
    border-color: #7c6af5;
  }

  .field-input:disabled,
  .field-select:disabled {
    opacity: 0.6;
  }

  .slug-input {
    font-family: monospace;
    font-size: 13px;
  }

  .field-textarea {
    width: 100%;
    padding: 9px 12px;
    background: #0f0f18;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 14px;
    font-family: inherit;
    line-height: 1.5;
    resize: vertical;
    outline: none;
    transition: border-color 0.15s;
    box-sizing: border-box;
  }

  .field-textarea:focus {
    border-color: #7c6af5;
  }

  .field-textarea:disabled {
    opacity: 0.6;
  }

  .field-select option {
    background: #1a1a24;
  }

  .field-hint {
    font-size: 12px;
    color: #555;
    margin: 0;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .btn-cancel {
    padding: 8px 16px;
    background: none;
    border: 1px solid #3a3a4a;
    border-radius: 6px;
    color: #888;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .btn-cancel:hover:not(:disabled) {
    border-color: #555;
    color: #e2e2e8;
  }

  .btn-cancel:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-create {
    padding: 8px 20px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .btn-create:hover:not(:disabled) {
    opacity: 0.88;
  }

  .btn-create:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
</style>
