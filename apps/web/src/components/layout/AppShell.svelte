<script>
  let { rail, topbar, children } = $props();
</script>

<div class="app-shell">
  <aside class="app-rail">
    {@render rail()}
  </aside>
  <div class="app-main">
    <header class="app-topbar">
      {@render topbar()}
    </header>
    <main class="app-content">
      {@render children()}
    </main>
  </div>
</div>

<style>
  .app-shell {
    display: flex;
    height: 100vh;
    width: 100vw;
    overflow: hidden;
    background: var(--color-canvas);
  }
  .app-rail {
    width: var(--rail-width);
    flex-shrink: 0;
    border-right: 1px solid var(--color-border);
    background: var(--color-surface);
    display: flex;
    flex-direction: column;
  }
  .app-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
  .app-topbar {
    height: var(--topbar-height);
    flex-shrink: 0;
    border-bottom: 1px solid var(--color-border);
    background: var(--color-canvas);
    display: flex;
    align-items: center;
    padding: 0 var(--space-16);
  }
  .app-content {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
</style>
