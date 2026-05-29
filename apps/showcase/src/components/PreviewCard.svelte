<script>
  let { example, Preview = null } = $props();

  function entries(value) {
    return Object.entries(value || {});
  }
</script>

<article class="preview-card">
  <header class="preview-header">
    <div>
      <h3>{example.title}</h3>
      <p>{example.components.join(' + ')}</p>
    </div>
    <code>{example.id}</code>
  </header>

  <div class="preview-stage">
    {#if Preview}
      <Preview {example} />
    {:else}
      <div class="missing-preview">
        Missing preview component for key <code>{example.preview}</code>.
      </div>
    {/if}
  </div>

  <div class="preview-meta">
    <section>
      <h4>Sample state</h4>
      <dl>
        {#each entries(example.sampleState) as [key, value]}
          <div><dt>{key}</dt><dd>{value}</dd></div>
        {/each}
      </dl>
    </section>

    <section>
      <h4>Important props</h4>
      <dl>
        {#each entries(example.props) as [key, value]}
          <div><dt>{key}</dt><dd>{String(value)}</dd></div>
        {/each}
      </dl>
    </section>

    <section class="notes-section">
      <h4>Usage notes</h4>
      <ul>
        {#each example.notes as note}
          <li>{note}</li>
        {/each}
      </ul>
    </section>
  </div>
</article>

<style>
  .preview-card {
    border: 1px solid var(--color-border);
    border-radius: 14px;
    background: color-mix(in srgb, var(--color-surface) 82%, #000 18%);
    overflow: hidden;
    box-shadow: 0 14px 44px rgba(0, 0, 0, 0.24);
  }

  .preview-header {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    padding: 16px 18px;
    border-bottom: 1px solid var(--color-border);
    background: rgba(255, 255, 255, 0.02);
  }

  h3,
  h4,
  p {
    margin: 0;
  }

  h3 {
    font-size: 16px;
  }

  .preview-header p {
    margin-top: 4px;
    color: var(--color-text-secondary);
    font-size: 12px;
  }

  code {
    height: max-content;
    padding: 3px 7px;
    border: 1px solid var(--color-border);
    border-radius: 6px;
    background: var(--color-canvas);
    color: var(--color-text-secondary);
    font-size: 12px;
  }

  .preview-stage {
    padding: 22px;
    background:
      radial-gradient(circle at top left, rgba(124, 106, 245, 0.08), transparent 35%),
      var(--color-canvas);
  }

  .preview-meta {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 18px;
    padding: 18px;
    border-top: 1px solid var(--color-border);
  }

  .notes-section {
    grid-column: 1 / -1;
  }

  h4 {
    margin-bottom: 8px;
    color: var(--color-text-primary);
    font-size: 12px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  dl,
  ul {
    margin: 0;
    padding: 0;
  }

  dl {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  dl div {
    display: grid;
    grid-template-columns: 110px 1fr;
    gap: 8px;
  }

  dt {
    color: var(--color-text-muted);
    font-size: 12px;
    text-transform: capitalize;
  }

  dd {
    margin: 0;
    color: var(--color-text-secondary);
    font-size: 12px;
  }

  ul {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding-left: 18px;
    color: var(--color-text-secondary);
    font-size: 13px;
    line-height: 1.5;
  }

  .missing-preview {
    padding: 18px;
    border: 1px dashed var(--color-warning);
    border-radius: 10px;
    color: var(--color-warning);
    background: rgba(229, 192, 123, 0.08);
  }

  @media (max-width: 720px) {
    .preview-meta {
      grid-template-columns: 1fr;
    }

    dl div {
      grid-template-columns: 1fr;
      gap: 2px;
    }
  }
</style>
