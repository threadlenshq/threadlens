<script>
  let { rail, topbar, children, insight } = $props();
</script>

<div class="flex h-screen w-full bg-canvas text-primary overflow-hidden flex-col md:flex-row">
  <aside class="app-rail">
    {@render rail()}
  </aside>
  <div class="app-main">
    <header class="app-topbar">
      {@render topbar()}
    </header>
    <main class="flex-1 flex flex-col min-w-0 h-full overflow-hidden">
      {@render children()}
    </main>
  </div>
  <!-- Insight Pane (Desktop Only) -->
  <div class="hidden xl:block">
    {@render insight?.()}
  </div>
</div>

<style>
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
</style>
