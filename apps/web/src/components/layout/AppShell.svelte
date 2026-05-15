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
  {#if insight}
    <aside class="app-insight hidden xl:flex xl:flex-col" aria-label="Insight pane">
      {@render insight()}
    </aside>
  {/if}
</div>

<style>
  .app-rail {
    width: 100%;
    flex-shrink: 0;
    border-bottom: 1px solid var(--color-border);
    background: var(--color-surface);
    display: flex;
    flex-direction: column;
  }

  @media (min-width: 768px) {
    .app-rail {
      width: var(--rail-width);
      border-bottom: none;
      border-right: 1px solid var(--color-border);
    }
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
