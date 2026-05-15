<script>
    let { title, description, icon = 'inbox', actionLabel = null, onAction = null, ...restProps } = $props();
</script>

<div class="flex flex-col items-center justify-center p-8 text-center rounded-lg border border-dashed border-subtle bg-canvas-sunken" {...restProps}>
    <div class="w-12 h-12 mb-4 rounded-full bg-surface flex items-center justify-center text-muted">
        <!-- Minimal icon placeholder -->
        <span class="text-xl">
            {#if icon === 'inbox'}📭{:else if icon === 'search'}🔍{:else if icon === 'settings'}⚙️{:else}✨{/if}
        </span>
    </div>
    <h3 class="text-section-title font-semibold text-primary mb-1">{title}</h3>
    <p class="text-body text-secondary max-w-sm mb-5">{description}</p>
    
    {#if actionLabel && onAction}
        <button class="px-4 py-2 rounded-md bg-brand text-white font-medium hover:bg-brand-hover transition-colors" onclick={onAction}>
            {actionLabel}
        </button>
    {/if}
</div>
