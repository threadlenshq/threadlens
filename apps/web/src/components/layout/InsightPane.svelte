<script>
    import Surface from '../ui/Surface.svelte';
    let { project, activeRuns, enabledQueryCount, filterSummary, selectedReport } = $props();
</script>

<aside class="w-[300px] border-l border-base bg-canvas flex flex-col h-full overflow-y-auto hidden xl:flex">
    <div class="p-4 border-b border-base">
        <h2 class="text-section-title font-semibold text-primary">Insights</h2>
    </div>
    
    <div class="p-4 space-y-6">
        <!-- Project Context -->
        <Surface padding="dense" class="space-y-2">
            <h3 class="text-label text-secondary uppercase tracking-wider">Project Status</h3>
            <div class="flex justify-between items-center text-body">
                <span>Mode:</span>
                <span class="text-primary font-medium">{project?.mode || 'Research'}</span>
            </div>
            <div class="flex justify-between items-center text-body">
                <span>Queries:</span>
                <span class="text-primary font-medium">{enabledQueryCount || 0} active</span>
            </div>
        </Surface>

        <!-- Current Filters -->
        <Surface padding="dense" class="space-y-2">
            <h3 class="text-label text-secondary uppercase tracking-wider">Current View</h3>
            <div class="text-body text-primary">{filterSummary || 'All findings'}</div>
        </Surface>
        
        <!-- Report Context -->
        {#if selectedReport}
            <Surface padding="dense" class="border-brand/30 bg-brand/5">
                <h3 class="text-label text-brand uppercase tracking-wider mb-2">Active Report</h3>
                <p class="text-body text-primary truncate">{selectedReport.title}</p>
            </Surface>
        {:else}
            <Surface padding="dense" elevation="sunken">
                <p class="text-body text-secondary text-sm">Select posts and generate a report to see clustered pain points and angles here.</p>
            </Surface>
        {/if}
    </div>
</aside>
