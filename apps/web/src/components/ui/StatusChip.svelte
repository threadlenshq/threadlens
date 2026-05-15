<script>
  let { status = '', type = 'default' } = $props();
  let colorVar = $derived(type === 'success' ? 'var(--color-success)' : type === 'danger' ? 'var(--color-danger)' : type === 'warning' ? 'var(--color-warning)' : 'var(--color-brand)');
</script>
<span class="status-chip" style="color: {colorVar}; background: {colorVar}20; border: 1px solid {colorVar}40">
  {status}
</span>
<style>
  .status-chip {
    font-size: 10px; font-weight: 600; padding: 2px 6px; border-radius: 4px;
    letter-spacing: 0.03em; text-transform: uppercase; display: inline-flex;
  }
</style>
