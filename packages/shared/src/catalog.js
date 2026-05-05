export const MODEL_CATALOG = [
  { id: 'copilot:gpt-4.1',           provider: 'copilot',    model: 'gpt-4.1',                   label: 'GPT-4.1 (Copilot)',           tier: 'low',       cost: 'free (0x)' },
  { id: 'copilot:gpt-5-mini',        provider: 'copilot',    model: 'gpt-5-mini',                label: 'GPT-5 mini (Copilot)',        tier: 'low',       cost: 'free (0x)' },
  { id: 'copilot:gpt-5.4-mini',      provider: 'copilot',    model: 'gpt-5.4-mini',              label: 'GPT-5.4 mini (Copilot)',      tier: 'medium',    cost: '0.33x' },
  { id: 'copilot:gpt-5.4',           provider: 'copilot',    model: 'gpt-5.4',                   label: 'GPT-5.4 (Copilot)',           tier: 'high',      cost: '1x' },
  { id: 'copilot:claude-haiku-4.5',  provider: 'copilot',    model: 'claude-haiku-4.5',          label: 'Claude Haiku 4.5 (Copilot)',  tier: 'low',       cost: '0.33x' },
  { id: 'copilot:claude-sonnet-4.6', provider: 'copilot',    model: 'claude-sonnet-4.6',         label: 'Claude Sonnet 4.6 (Copilot)', tier: 'high',      cost: '1x' },
  { id: 'claude-cli:haiku',          provider: 'claude-cli', model: 'haiku',                     label: 'Claude Haiku (CLI)',          tier: 'low',       cost: 'token-billed' },
  { id: 'claude-cli:sonnet',         provider: 'claude-cli', model: 'sonnet',                    label: 'Claude Sonnet (CLI)',         tier: 'high',      cost: 'token-billed' },
  { id: 'claude-cli:opus',           provider: 'claude-cli', model: 'opus',                      label: 'Claude Opus (CLI)',           tier: 'reasoning', cost: 'token-billed' },
  { id: 'sdk:haiku',                 provider: 'sdk',        model: 'claude-haiku-4-5-20251001', label: 'Claude Haiku (SDK)',          tier: 'low',       cost: 'token-billed' },
  { id: 'gemini:2.5-flash',          provider: 'gemini',     model: 'gemini-2.5-flash',          label: 'Gemini 2.5 Flash',            tier: 'low',       cost: 'token-billed' },
];

export const TASKS = [
  { id: 'post_scoring',      label: 'Post Scoring',           complexity: 'low',       volume: 'high',     timeoutMs: 60000,
    description: 'Per-post pain-signal scoring during Reddit/Bluesky scout runs. High volume - runs 10 batches of 15 posts in parallel.',
    default: 'copilot:gpt-5-mini' },
  { id: 'query_suggestion',  label: 'Query Suggestion',       complexity: 'medium',    volume: 'one-time', timeoutMs: 60000,
    description: 'Generates 10 search URLs and angles when adding queries.',
    default: 'copilot:gpt-5-mini' },
  { id: 'query_refinement',  label: 'Query Refinement',       complexity: 'reasoning', volume: 'one-time', timeoutMs: 300000,
    description: 'Reviews current queries against report findings and recommends which to disable or add next. Uses a 5-minute timeout because report-backed refinement can run longer than basic suggestions.',
    default: 'claude-cli:opus' },
  { id: 'report_clustering', label: 'Research Report',        complexity: 'reasoning', volume: 'one-time', timeoutMs: 300000,
    description: 'Clusters posts into pain themes and proposes product angles. One-time per report; 5-minute timeout.',
    default: 'claude-cli:opus' },
  { id: 'draft_generation',  label: 'Draft Comment / DM',     complexity: 'medium',    volume: 'per-post', timeoutMs: 60000,
    description: 'Drafts contextual replies and direct messages for outreach.',
    default: 'claude-cli:haiku' },
  { id: 'google_analysis',   label: 'Google Search Analysis', complexity: 'low',       volume: 'high',     timeoutMs: 60000,
    description: 'Per-result product extraction plus relevance and intent scoring in Google scout.',
    default: 'copilot:claude-haiku-4.5' },
  { id: 'advisor_council',   label: 'Advisor Council',         complexity: 'reasoning', volume: 'one-time', timeoutMs: 300000,
    description: 'Five-advisor council critique of a completed research report.',
    default: 'claude-cli:opus' },
];

const TIER_RANK = { low: 0, medium: 1, high: 2, reasoning: 3 };
const VALID_TIERS = new Set(Object.keys(TIER_RANK));

function getTierRank(tier, label) {
  if (!VALID_TIERS.has(tier)) {
    throw new RangeError(`Invalid ${label}: ${tier}`);
  }

  return TIER_RANK[tier];
}

export function isOverkill(modelTier, taskComplexity) {
  const modelRank = getTierRank(modelTier, 'modelTier');
  const taskRank = getTierRank(taskComplexity, 'taskComplexity');

  return modelRank > taskRank + 1;
}

export function isUnderkill(modelTier, taskComplexity) {
  const modelRank = getTierRank(modelTier, 'modelTier');
  const taskRank = getTierRank(taskComplexity, 'taskComplexity');

  return modelRank < taskRank;
}

export function getModel(id) {
  return MODEL_CATALOG.find(m => m.id === id);
}

export function getTask(id) {
  return TASKS.find(t => t.id === id);
}
