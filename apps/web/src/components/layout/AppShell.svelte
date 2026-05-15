<script>
  let { rail, topbar, children, insight } = $props();
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
  {#if insight}
    <aside class="app-insight" aria-label="Insight pane">
      {@render insight()}
    </aside>
  {/if}
</div>

<style>
  .app-shell {
    display: flex;
    height: 100vh;
    width: 100vw;
    overflow: hidden;
    background: var(--color-canvas);
    color: var(--color-text-primary);
    flex-direction: row;
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
    min-width: 0;
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
    min-width: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .app-insight {
    width: 300px;
    flex-shrink: 0;
    border-left: 1px solid var(--color-border);
    background: var(--color-canvas);
    overflow: hidden;
    display: none;
  }

  @media (min-width: 1280px) {
    .app-insight {
      display: flex;
      flex-direction: column;
    }
  }

  @media (max-width: 767px) {
    .app-shell {
      flex-direction: column;
    }

    .app-rail {
      width: 100%;
      border-right: none;
      border-bottom: 1px solid var(--color-border);
    }
  }
</style>
