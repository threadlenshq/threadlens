<script>
    let { title, description, icon = 'inbox', actionLabel = null, onaction = null } = $props();
</script>

<div class="empty-state">
    <div class="icon-container">
        <!-- Minimal icon placeholder -->
        <span class="icon">
            {#if icon === 'inbox'}📭{:else if icon === 'search'}🔍{:else if icon === 'settings'}⚙️{:else}✨{/if}
        </span>
    </div>
    <h3 class="title">{title}</h3>
    <p class="description">{description}</p>
    
    {#if actionLabel && onaction}
        <button class="action-btn" onclick={onaction}>
            {actionLabel}
        </button>
    {/if}
</div>

<style>
    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 48px 32px;
        text-align: center;
        border-radius: 12px;
        border: 1px dashed var(--color-border);
        background-color: rgba(26, 26, 36, 0.3); /* subtle surface blend */
    }

    .icon-container {
        width: 48px;
        height: 48px;
        margin-bottom: 16px;
        border-radius: 50%;
        background-color: var(--color-surface-elevated);
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--color-text-muted);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }

    .icon {
        font-size: 20px;
    }

    .title {
        font-size: 16px;
        font-weight: 600;
        color: var(--color-text-primary);
        margin: 0 0 8px 0;
    }

    .description {
        font-size: 14px;
        color: var(--color-text-secondary);
        max-width: 400px;
        margin: 0;
        line-height: 1.5;
    }

    .action-btn {
        margin-top: 20px;
        padding: 8px 16px;
        border-radius: 6px;
        background-color: var(--color-brand);
        color: white;
        font-weight: 500;
        font-size: 14px;
        border: none;
        cursor: pointer;
        transition: background-color 0.2s;
    }

    .action-btn:hover {
        background-color: var(--color-brand-hover);
    }
</style>
