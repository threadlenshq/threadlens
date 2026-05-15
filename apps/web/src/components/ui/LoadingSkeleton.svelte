<script>
    let { type = 'row', count = 1, class: className = '' } = $props();

    // Clamp count to a safe integer [1, 20].
    // The typeof guard prevents Symbol/object coercion throws before Number().
    function clampCount(n) {
        const num = (typeof n === 'number' || typeof n === 'string') ? Number(n) : NaN;
        return Math.min(Math.max(Math.trunc(num) || 1, 1), 20);
    }
    let safeCount = $derived(clampCount(count));

    // Normalise type: unknown values fall back to 'row' so nothing is silently skipped.
    const VALID_TYPES = new Set(['row', 'card', 'text']);
    let safeType = $derived(VALID_TYPES.has(type) ? type : 'row');
</script>

<div class="animate-pulse space-y-4 {className}">
    {#each Array(safeCount) as _}
        {#if safeType === 'row'}
            <div class="flex items-center space-x-4 p-4 border border-base rounded-md bg-surface">
                <div class="h-10 w-10 bg-muted/20 rounded-full"></div>
                <div class="flex-1 space-y-2">
                    <div class="h-4 bg-muted/20 rounded w-3/4"></div>
                    <div class="h-3 bg-muted/20 rounded w-1/2"></div>
                </div>
            </div>
        {:else if safeType === 'card'}
            <div class="p-5 border border-base rounded-lg bg-surface space-y-3">
                <div class="h-5 bg-muted/20 rounded w-1/3 mb-4"></div>
                <div class="h-4 bg-muted/20 rounded w-full"></div>
                <div class="h-4 bg-muted/20 rounded w-5/6"></div>
                <div class="h-4 bg-muted/20 rounded w-4/6"></div>
            </div>
        {:else if safeType === 'text'}
            <div class="h-4 bg-muted/20 rounded w-full"></div>
        {/if}
    {/each}
</div>

<style>
    @media (prefers-reduced-motion: reduce) {
        .animate-pulse { animation: none; opacity: 0.7; }
    }
</style>
