<script>
  import FindingCard from '$web/components/inbox/FindingCard.svelte';
  import { mockFindings, noop } from '../scenarios/mock-data.js';

  let { example } = $props();

  const postByExample = {
    'high-score-finding': mockFindings[0],
    'selected-research-finding': mockFindings[1],
    'low-signal-finding': mockFindings[2],
  };

  let post = $derived(postByExample[example.id] || mockFindings[0]);
  let selected = $derived(example.id === 'selected-research-finding');
  let checked = $derived(example.id === 'selected-research-finding');
</script>

<div class="finding-stage">
  <FindingCard
    {post}
    {selected}
    projectMode="research"
    bulkMode={selected}
    {checked}
    onSelect={noop}
    onBulkToggle={noop}
  />
</div>

<style>
  .finding-stage {
    max-width: 560px;
  }
</style>
