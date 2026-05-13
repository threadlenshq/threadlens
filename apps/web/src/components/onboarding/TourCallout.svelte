<script>
  let { title, body, actionLabel = 'Open checklist', onDismiss = () => {} } = $props();
</script>

<aside class="tour-callout" data-testid="tour-callout" aria-live="polite">
  <div>
    <h3>{title}</h3>
    <p>{body}</p>
  </div>
  <button class="tour-dismiss" onclick={onDismiss}>{actionLabel}</button>
</aside>

<style>
  .tour-callout {
    border: 1px solid #3a3a5a;
    background: #171726;
    color: #e2e2e8;
    border-radius: 10px;
    padding: 14px;
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: flex-start;
  }

  h3 {
    margin: 0 0 4px;
    font-size: 14px;
  }

  p {
    margin: 0;
    color: #aaaac0;
    font-size: 13px;
    line-height: 1.4;
  }

  .tour-dismiss {
    border: 1px solid #4a4a68;
    background: #23233a;
    color: #e2e2e8;
    border-radius: 6px;
    padding: 6px 10px;
    cursor: pointer;
    white-space: nowrap;
  }
</style>
