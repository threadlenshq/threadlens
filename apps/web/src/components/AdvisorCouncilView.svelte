<script>
  let { council } = $props();

  let expanded = $state(new Set());

  function toggleAdvisor(i) {
    if (expanded.has(i)) {
      expanded.delete(i);
    } else {
      expanded.add(i);
    }
    expanded = new Set(expanded);
  }
</script>

<div class="advisor-council">
  <h3 class="council-header">Advisor Council</h3>

  {#if council.synthesis}
    <div class="synthesis-section">
      <div class="chair-verdict">
        <span class="verdict-label">Chair's Verdict</span>
        <p class="verdict-text">{council.synthesis.chair_verdict}</p>
      </div>

      {#if council.synthesis.agreements?.length}
        <div class="synthesis-group">
          <span class="group-label">Agreements</span>
          <ul>
            {#each council.synthesis.agreements as item}
              <li>{item}</li>
            {/each}
          </ul>
        </div>
      {/if}

      {#if council.synthesis.conflicts?.length}
        <div class="synthesis-group">
          <span class="group-label">Conflicts</span>
          <ul>
            {#each council.synthesis.conflicts as item}
              <li>{item}</li>
            {/each}
          </ul>
        </div>
      {/if}

      {#if council.synthesis.blind_spots?.length}
        <div class="synthesis-group">
          <span class="group-label">Blind Spots</span>
          <ul>
            {#each council.synthesis.blind_spots as item}
              <li>{item}</li>
            {/each}
          </ul>
        </div>
      {/if}

      {#if council.synthesis.next_experiment}
        <div class="next-experiment">
          <span class="group-label">Next Experiment</span>
          <div class="experiment-body">
            <p><strong>Hypothesis:</strong> {council.synthesis.next_experiment.hypothesis}</p>
            <p><strong>Test:</strong> {council.synthesis.next_experiment.test}</p>
            <p><strong>Success signal:</strong> {council.synthesis.next_experiment.success_signal}</p>
          </div>
        </div>
      {/if}
    </div>
  {/if}

  {#if council.advisors?.length}
    <div class="advisors-section">
      <span class="group-label">Advisors</span>
      {#each council.advisors as advisor, i}
        <div class="advisor-card">
          <button class="advisor-header" onclick={() => toggleAdvisor(i)}>
            <span class="lens-name">{advisor.lens}</span>
            <span class="key-claim">{advisor.key_claim}</span>
            <span class="expand-toggle">{expanded.has(i) ? '▴' : '▾'}</span>
          </button>
          {#if expanded.has(i)}
            <div class="advisor-details">
              {#if advisor.critique?.length}
                <div class="detail-group">
                  <span class="detail-label">Critique</span>
                  <ul>
                    {#each advisor.critique as item}
                      <li>{item}</li>
                    {/each}
                  </ul>
                </div>
              {/if}
              {#if advisor.risks?.length}
                <div class="detail-group">
                  <span class="detail-label">Risks</span>
                  <ul>
                    {#each advisor.risks as item}
                      <li>{item}</li>
                    {/each}
                  </ul>
                </div>
              {/if}
              {#if advisor.questions_to_test?.length}
                <div class="detail-group">
                  <span class="detail-label">Questions to Test</span>
                  <ul>
                    {#each advisor.questions_to_test as item}
                      <li>{item}</li>
                    {/each}
                  </ul>
                </div>
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .advisor-council {
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 20px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
  }

  .council-header {
    font-size: 11px;
    font-weight: 600;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin: 0;
  }

  .synthesis-section {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .chair-verdict {
    padding: 14px;
    background: #23233a;
    border: 1px solid #3a3a55;
    border-radius: 6px;
  }

  .verdict-label {
    display: block;
    font-size: 10px;
    font-weight: 600;
    color: #7c6af5;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 6px;
  }

  .verdict-text {
    font-size: 14px;
    color: #e2e2e8;
    line-height: 1.6;
    margin: 0;
  }

  .synthesis-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .group-label {
    font-size: 10px;
    font-weight: 600;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .synthesis-group ul,
  .detail-group ul {
    margin: 0;
    padding-left: 18px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .synthesis-group li,
  .detail-group li {
    font-size: 13px;
    color: #c0c0d0;
    line-height: 1.5;
  }

  .next-experiment {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .experiment-body {
    padding: 12px;
    background: #23233a;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .experiment-body p {
    font-size: 13px;
    color: #c0c0d0;
    margin: 0;
    line-height: 1.5;
  }

  .experiment-body strong {
    color: #e2e2e8;
  }

  .advisors-section {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .advisor-card {
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    overflow: hidden;
  }

  .advisor-header {
    width: 100%;
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 14px;
    background: #23233a;
    border: none;
    cursor: pointer;
    text-align: left;
  }

  .advisor-header:hover {
    background: #2a2a44;
  }

  .lens-name {
    font-size: 12px;
    font-weight: 600;
    color: #7c6af5;
    white-space: nowrap;
    min-width: 90px;
  }

  .key-claim {
    font-size: 13px;
    color: #c0c0d0;
    flex: 1;
    line-height: 1.4;
  }

  .expand-toggle {
    font-size: 10px;
    color: #666;
    flex-shrink: 0;
  }

  .advisor-details {
    padding: 12px 14px;
    background: #1a1a24;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .detail-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .detail-label {
    font-size: 10px;
    font-weight: 600;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
</style>
