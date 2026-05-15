<script>
    let { selectedCount = 0, onCancel, onAction } = $props();
</script>

{#if selectedCount > 0}
    <div class="flex items-center justify-between p-3 bg-brand/10 border-b border-brand/20 sticky top-0 z-10 animate-fade-in">
        <div class="flex items-center space-x-3">
            <span class="text-brand font-medium text-body">{selectedCount} selected</span>
            <button class="text-secondary hover:text-primary text-sm transition-colors" onclick={onCancel}>Cancel</button>
        </div>
        <div class="flex space-x-2">
            <button class="px-3 py-1.5 text-sm font-medium rounded bg-surface border border-base hover:bg-canvas transition-colors" onclick={() => onAction('status', 'drafted')}>Mark Draft</button>
            <button class="px-3 py-1.5 text-sm font-medium rounded bg-surface border border-base hover:bg-canvas transition-colors" onclick={() => onAction('status', 'excluded')}>Exclude</button>
        </div>
    </div>
{/if}

<style>
    .animate-fade-in { animation: fadeIn 0.2s ease-out; }
    @keyframes fadeIn { from { opacity: 0; transform: translateY(-4px); } to { opacity: 1; transform: translateY(0); } }
    @media (prefers-reduced-motion: reduce) { .animate-fade-in { animation: none; } }
</style>
