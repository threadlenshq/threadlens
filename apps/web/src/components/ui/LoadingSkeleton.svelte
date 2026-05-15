<script>
    let { type = 'row', count = 1, class: className = '' } = $props();

    // Clamp to a safe integer [1, 20]. The typeof guard prevents Symbol/object
    // coercion throws before passing to Number(); falls back to 1 for any non-numeric.
    let safeCount = $derived(Math.min(Math.max(Math.trunc(
        (typeof count === 'number' || typeof count === 'string') ? (Number(count) || 1) : 1
    ), 1), 20));
</script>

<div class="animate-pulse space-y-4 {className}">
    {#each Array(safeCount) as _}
        {#if type === 'row'}
            <div class="flex items-center space-x-4 p-4 border border-base rounded-md bg-surface">
                <div class="h-10 w-10 bg-muted/20 rounded-full"></div>
                <div class="flex-1 space-y-2">
                    <div class="h-4 bg-muted/20 rounded w-3/4"></div>
                    <div class="h-3 bg-muted/20 rounded w-1/2"></div>
                </div>
            </div>
        {:else if type === 'card'}
            <div class="p-5 border border-base rounded-lg bg-surface space-y-3">
                <div class="h-5 bg-muted/20 rounded w-1/3 mb-4"></div>
                <div class="h-4 bg-muted/20 rounded w-full"></div>
                <div class="h-4 bg-muted/20 rounded w-5/6"></div>
                <div class="h-4 bg-muted/20 rounded w-4/6"></div>
            </div>
        {:else if type === 'text'}
            <div class="h-4 bg-muted/20 rounded w-full"></div>
        {/if}
    {/each}
</div>

<style>
    @media (prefers-reduced-motion: reduce) {
        .animate-pulse { animation: none; opacity: 0.7; }
    }
</style>
