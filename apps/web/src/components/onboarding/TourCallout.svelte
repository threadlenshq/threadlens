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
    border: 1px solid #4a4a68;
    background: linear-gradient(180deg, #1f1f33 0%, #171726 100%);
    box-shadow: 0 4px 12px rgba(0,0,0,0.2);
    color: #e2e2e8;
    border-radius: 10px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    align-items: flex-start;
  }

  h3 {
    margin: 0 0 6px;
    font-size: 15px;
    font-weight: 600;
    color: #fff;
  }

  p {
    margin: 0;
    color: #aaaac0;
    font-size: 13px;
    line-height: 1.4;
  }

  .tour-dismiss {
    border: 1px solid #4a4a68;
    background: #2a2a40;
    color: #e2e2e8;
    border-radius: 6px;
    padding: 8px 12px;
    cursor: pointer;
    white-space: nowrap;
    width: 100%;
    font-weight: 500;
    transition: background 0.2s;
  }
  
  .tour-dismiss:hover {
    background: #363652;
  }
</style>
