<script>
  import InsightPane from './components/layout/InsightPane.svelte';

  const project = { mode: 'Research' };
  const enabledQueryCount = 5;
  const filterSummary = 'Showing: new · reddit';
  const sampleReport = { title: 'Pain Points in Developer Tooling (June 2025)' };

  let selectedReport = $state(null);
</script>

<div class="page">

  <div class="header">
    <h1>Component Preview</h1>
    <p>InsightPane — layout/InsightPane.svelte</p>
  </div>

  <!-- InsightPane -->
  <section>
    <div class="section-title">
      <span class="section-name">InsightPane</span>
      <span class="section-path">layout/InsightPane.svelte</span>
    </div>
    <p class="section-desc">300 px right sidebar. Normally hidden below the <code>xl</code> breakpoint — forced visible here. Toggle the report state to see both variants.</p>

    <div class="demo-box controls-row">
      <span class="label">Active report:</span>
      <button onclick={() => selectedReport = selectedReport ? null : sampleReport}>
        {selectedReport ? 'Clear report' : 'Set sample report'}
      </button>
    </div>

    <div class="insight-demo">
      <div class="fake-content">← Main content area (simulated)</div>
      <!-- force the aside visible by overriding its hidden/xl:flex classes -->
      <div class="insight-wrapper">
        <InsightPane
          {project}
          activeRuns={[]}
          {enabledQueryCount}
          {filterSummary}
          {selectedReport}
        />
      </div>
    </div>
  </section>

</div>

<style>
  :global(*) { box-sizing: border-box; margin: 0; padding: 0; }

  :global(body) {
    background: #0f0f13;
    color: #e2e2e8;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  }

  /* ── CSS variable aliases so child components work ── */
  :global(:root) {
    --color-canvas: #0f0f13;
    --color-surface: #1a1a24;
    --color-surface-elevated: #23233a;
    --color-border: #2a2a3a;
    --color-border-hover: #3a3a50;
    --color-text-primary: #e2e2e8;
    --color-text-secondary: #8f8fa5;
    --color-text-muted: #5e5e70;
    --color-brand: #7c6af5;
    --color-brand-hover: #9082f7;
    --color-success: #98c379;
    --color-warning: #e5c07b;
    --color-danger: #e06c75;
  }

  /* ── Utility class shims used by child components ── */
  :global(.bg-canvas)        { background: var(--color-canvas) !important; }
  :global(.bg-surface)       { background: var(--color-surface) !important; }
  :global(.bg-canvas-sunken) { background: #0b0b0f !important; }
  :global(.border-base)      { border-color: var(--color-border) !important; }
  :global(.border-subtle)    { border-color: #1e1e2a !important; }
  :global(.border-elevated)  { border-color: #323248 !important; }
  :global(.border-brand\/30) { border-color: rgba(124,106,245,0.3) !important; }
  :global(.text-primary)     { color: var(--color-text-primary) !important; }
  :global(.text-secondary)   { color: var(--color-text-secondary) !important; }
  :global(.text-brand)       { color: var(--color-brand) !important; }
  :global(.bg-brand\/5)      { background: rgba(124,106,245,0.05) !important; }
  :global(.bg-brand\/10)     { background: rgba(124,106,245,0.1) !important; }
  :global(.border-brand\/20) { border-color: rgba(124,106,245,0.2) !important; }
  :global(.font-semibold)    { font-weight: 600 !important; }
  :global(.font-medium)      { font-weight: 500 !important; }
  :global(.shadow-sm)        { box-shadow: 0 1px 3px rgba(0,0,0,0.3) !important; }
  :global(.shadow-elevated)  { box-shadow: 0 4px 12px rgba(0,0,0,0.4) !important; }
  :global(.rounded-lg)       { border-radius: 8px !important; }
  :global(.border)           { border-width: 1px !important; border-style: solid !important; }
  :global(.p-0)              { padding: 0 !important; }
  :global(.p-3)              { padding: 12px !important; }
  :global(.p-4)              { padding: 16px !important; }
  :global(.p-5)              { padding: 20px !important; }
  :global(.p-6)              { padding: 24px !important; }
  :global(.space-y-2 > * + *) { margin-top: 8px !important; }
  :global(.space-y-6 > * + *) { margin-top: 24px !important; }
  :global(.flex)             { display: flex !important; }
  :global(.flex-col)         { flex-direction: column !important; }
  :global(.items-center)     { align-items: center !important; }
  :global(.justify-between)  { justify-content: space-between !important; }
  :global(.overflow-y-auto)  { overflow-y: auto !important; }
  :global(.h-full)           { height: 100% !important; }
  :global(.w-\[300px\])      { width: 300px !important; }
  :global(.truncate)         { overflow: hidden; text-overflow: ellipsis; white-space: nowrap !important; }
  :global(.text-sm)          { font-size: 13px !important; }
  :global(.text-label)       { font-size: 11px !important; }
  :global(.text-body)        { font-size: 14px !important; }
  :global(.uppercase)        { text-transform: uppercase !important; }
  :global(.tracking-wider)   { letter-spacing: 0.06em !important; }
  :global(.mb-2)             { margin-bottom: 8px !important; }

  /* ── Layout ── */
  .page {
    max-width: 900px;
    margin: 0 auto;
    padding: 48px 32px;
    display: flex;
    flex-direction: column;
    gap: 56px;
  }

  .header h1 {
    font-size: 22px;
    font-weight: 700;
    color: #e2e2e8;
    margin-bottom: 6px;
  }

  .header p {
    font-size: 13px;
    color: #8f8fa5;
  }

  section {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .section-title {
    display: flex;
    align-items: baseline;
    gap: 10px;
    padding-bottom: 10px;
    border-bottom: 1px solid #2a2a3a;
  }

  .section-name {
    font-size: 17px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .section-path {
    font-size: 12px;
    color: #5e5e70;
    font-family: monospace;
  }

  .section-desc {
    font-size: 13px;
    color: #8f8fa5;
    line-height: 1.6;
  }

  .section-desc code {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 4px;
    padding: 1px 5px;
    font-size: 12px;
    color: #e2e2e8;
  }

  .demo-box {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    padding: 20px;
  }

  .controls-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .label {
    font-size: 13px;
    color: #8f8fa5;
  }

  .controls-row button {
    padding: 5px 12px;
    font-size: 13px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 5px;
    color: #e2e2e8;
    cursor: pointer;
    transition: background 0.15s;
  }

  .controls-row button:hover {
    background: #23233a;
  }

  .insight-demo {
    display: flex;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
    overflow: hidden;
    min-height: 280px;
  }

  .fake-content {
    flex: 1;
    padding: 24px;
    background: #0f0f13;
    font-size: 13px;
    color: #5e5e70;
  }

  /* Force the InsightPane aside visible regardless of xl breakpoint */
  .insight-wrapper :global(aside) {
    display: flex !important;
    position: static !important;
    height: auto !important;
  }
</style>
