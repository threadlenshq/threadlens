<script>
    let { children, elevation = 'base', padding = 'comfortable', class: className = '', ...restProps } = $props();
    
    let elevationClass = $derived(
        elevation === 'elevated' ? 'bg-surface-elevated shadow-elevated border-elevated' :
        elevation === 'sunken'   ? 'bg-canvas-sunken border-subtle' :
        'bg-surface border-base shadow-sm'
    );
    
    let paddingClass = $derived(
        padding === 'none'  ? 'p-0' :
        padding === 'dense' ? 'p-3' : 'p-5'
    );
</script>

<div class="rounded-lg border {elevationClass} {paddingClass} {className}" {...restProps}>
    {@render children?.()}
</div>
