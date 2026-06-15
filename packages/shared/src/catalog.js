export const MODEL_CATALOG = [
  { id: 'copilot:gpt-4.1',           provider: 'copilot',    model: 'gpt-4.1',                   label: 'GPT-4.1 (Copilot)',           tier: 'low',       cost: 'usage-based' },
  { id: 'copilot:gpt-5-mini',        provider: 'copilot',    model: 'gpt-5-mini',                label: 'GPT-5 mini (Copilot)',        tier: 'low',       cost: 'usage-based' },
  { id: 'copilot:gpt-5.4-mini',      provider: 'copilot',    model: 'gpt-5.4-mini',              label: 'GPT-5.4 mini (Copilot)',      tier: 'medium',    cost: 'usage-based' },
  { id: 'copilot:gpt-5.4',           provider: 'copilot',    model: 'gpt-5.4',                   label: 'GPT-5.4 (Copilot)',           tier: 'high',      cost: 'usage-based' },
  { id: 'copilot:claude-haiku-4.5',  provider: 'copilot',    model: 'claude-haiku-4.5',          label: 'Claude Haiku 4.5 (Copilot)',  tier: 'low',       cost: 'usage-based' },
  { id: 'copilot:claude-sonnet-4.6', provider: 'copilot',    model: 'claude-sonnet-4.6',         label: 'Claude Sonnet 4.6 (Copilot)', tier: 'high',      cost: 'usage-based' },
  { id: 'claude-cli:haiku',          provider: 'claude-cli', model: 'haiku',                     label: 'Claude Haiku (CLI)',          tier: 'low',       cost: 'token-billed' },
  { id: 'claude-cli:sonnet',         provider: 'claude-cli', model: 'sonnet',                    label: 'Claude Sonnet (CLI)',         tier: 'high',      cost: 'token-billed' },
  { id: 'claude-cli:opus',           provider: 'claude-cli', model: 'opus',                      label: 'Claude Opus (CLI)',           tier: 'reasoning', cost: 'token-billed' },
  { id: 'sdk:haiku',                 provider: 'sdk',        model: 'claude-haiku-4-5-20251001', label: 'Claude Haiku (SDK)',          tier: 'low',       cost: 'token-billed' },
  { id: 'gemini:2.5-flash',          provider: 'gemini',     model: 'gemini-2.5-flash',          label: 'Gemini 2.5 Flash',            tier: 'low',       cost: 'token-billed' },
  { id: 'opencode:big-pickle',              provider: 'opencode',     model: 'big-pickle',              label: 'Big Pickle (opencode)',              tier: 'low',       cost: 'free (0x)' },
  { id: 'opencode:deepseek-v4-flash-free',  provider: 'opencode',     model: 'deepseek-v4-flash-free',  label: 'DeepSeek V4 Flash Free (opencode)',  tier: 'low',       cost: 'free (0x)' },
  { id: 'opencode-go:deepseek-v4-flash',    provider: 'opencode-go',  model: 'deepseek-v4-flash',       label: 'DeepSeek V4 Flash (opencode Go)',    tier: 'low',       cost: 'subscription' },
  { id: 'opencode-go:mimo-v2.5',            provider: 'opencode-go',  model: 'mimo-v2.5',               label: 'Mimo v2.5 (opencode Go)',            tier: 'low',       cost: 'subscription' },
  { id: 'opencode-go:minimax-m2.5',         provider: 'opencode-go',  model: 'minimax-m2.5',            label: 'MiniMax M2.5 (opencode Go)',         tier: 'low',       cost: 'subscription' },
  { id: 'opencode-go:glm-5',               provider: 'opencode-go',  model: 'glm-5',                   label: 'GLM-5 (opencode Go)',                tier: 'low',       cost: 'subscription' },
  { id: 'opencode-go:glm-5.1',             provider: 'opencode-go',  model: 'glm-5.1',                 label: 'GLM-5.1 (opencode Go)',              tier: 'low',       cost: 'subscription' },
  { id: 'opencode-go:mimo-v2.5-pro',       provider: 'opencode-go',  model: 'mimo-v2.5-pro',           label: 'Mimo v2.5 Pro (opencode Go)',        tier: 'medium',    cost: 'subscription' },
  { id: 'opencode-go:minimax-m2.7',        provider: 'opencode-go',  model: 'minimax-m2.7',            label: 'MiniMax M2.7 (opencode Go)',         tier: 'medium',    cost: 'subscription' },
  { id: 'opencode-go:qwen3.6-plus',        provider: 'opencode-go',  model: 'qwen3.6-plus',            label: 'Qwen 3.6 Plus (opencode Go)',        tier: 'medium',    cost: 'subscription' },
  { id: 'opencode-go:deepseek-v4-pro',     provider: 'opencode-go',  model: 'deepseek-v4-pro',         label: 'DeepSeek V4 Pro (opencode Go)',      tier: 'high',      cost: 'subscription' },
  { id: 'opencode-go:minimax-m3',          provider: 'opencode-go',  model: 'minimax-m3',              label: 'MiniMax M3 (opencode Go)',           tier: 'high',      cost: 'subscription' },
  { id: 'opencode-go:qwen3.7-plus',        provider: 'opencode-go',  model: 'qwen3.7-plus',            label: 'Qwen 3.7 Plus (opencode Go)',        tier: 'high',      cost: 'subscription' },
  { id: 'opencode-go:qwen3.7-max',         provider: 'opencode-go',  model: 'qwen3.7-max',             label: 'Qwen 3.7 Max (opencode Go)',         tier: 'high',      cost: 'subscription' },
  { id: 'opencode-go:kimi-k2.5',           provider: 'opencode-go',  model: 'kimi-k2.5',               label: 'Kimi K2.5 (opencode Go)',            tier: 'reasoning', cost: 'subscription' },
  { id: 'opencode-go:kimi-k2.6',           provider: 'opencode-go',  model: 'kimi-k2.6',               label: 'Kimi K2.6 (opencode Go)',            tier: 'reasoning', cost: 'subscription' },
];

export const TASKS = [
  { id: 'post_scoring',      label: 'Post Scoring',           complexity: 'low',       volume: 'high',     timeoutMs: 60000,
    description: 'Per-post pain-signal scoring during Reddit/Bluesky scout runs. High volume - runs 10 batches of 15 posts in parallel.',
    default: 'copilot:gpt-5-mini',
    defaultByProvider: {
      copilot: 'copilot:gpt-5-mini',
      'claude-cli': 'claude-cli:haiku',
      opencode: 'opencode-go:deepseek-v4-flash',
      sdk: 'sdk:haiku',
      gemini: 'gemini:2.5-flash',
    } },
  { id: 'query_suggestion',  label: 'Query Suggestion',       complexity: 'medium',    volume: 'one-time', timeoutMs: 60000,
    description: 'Generates 10 search URLs and angles when adding queries.',
    default: 'copilot:gpt-5-mini',
    defaultByProvider: {
      copilot: 'copilot:claude-haiku-4.5',
      'claude-cli': 'claude-cli:sonnet',
      opencode: 'opencode-go:mimo-v2.5-pro',
      sdk: 'sdk:haiku',
      gemini: 'gemini:2.5-flash',
    } },
  { id: 'query_refinement',  label: 'Query Refinement',       complexity: 'reasoning', volume: 'one-time', timeoutMs: 300000,
    description: 'Reviews current queries against report findings and recommends which to disable or add next. Uses a 5-minute timeout because report-backed refinement can run longer than basic suggestions.',
    default: 'claude-cli:opus',
    defaultByProvider: {
      copilot: 'copilot:claude-sonnet-4.6',
      'claude-cli': 'claude-cli:opus',
      opencode: 'opencode-go:kimi-k2.5',
      sdk: 'sdk:haiku',
      gemini: 'gemini:2.5-flash',
    } },
  { id: 'report_clustering', label: 'Research Report',        complexity: 'reasoning', volume: 'one-time', timeoutMs: 300000,
    description: 'Clusters posts into pain themes and proposes product angles. One-time per report; 5-minute timeout.',
    default: 'claude-cli:opus',
    defaultByProvider: {
      copilot: 'copilot:claude-sonnet-4.6',
      'claude-cli': 'claude-cli:opus',
      opencode: 'opencode-go:kimi-k2.5',
      sdk: 'sdk:haiku',
      gemini: 'gemini:2.5-flash',
    } },
  { id: 'draft_generation',  label: 'Draft Comment / DM',     complexity: 'medium',    volume: 'per-post', timeoutMs: 60000,
    description: 'Drafts contextual replies and direct messages for outreach.',
    default: 'claude-cli:haiku',
    defaultByProvider: {
      copilot: 'copilot:claude-sonnet-4.6',
      'claude-cli': 'claude-cli:sonnet',
      opencode: 'opencode-go:mimo-v2.5-pro',
      sdk: 'sdk:haiku',
      gemini: 'gemini:2.5-flash',
    } },
  { id: 'google_analysis',   label: 'Google Search Analysis', complexity: 'low',       volume: 'high',     timeoutMs: 60000,
    description: 'Per-result product extraction plus relevance and intent scoring in Google scout.',
    default: 'copilot:claude-haiku-4.5',
    defaultByProvider: {
      copilot: 'copilot:gpt-5-mini',
      'claude-cli': 'claude-cli:haiku',
      opencode: 'opencode-go:deepseek-v4-flash',
      sdk: 'sdk:haiku',
      gemini: 'gemini:2.5-flash',
    } },
  { id: 'advisor_council',   label: 'Advisor Council',         complexity: 'reasoning', volume: 'one-time', timeoutMs: 300000,
    description: 'Five-advisor council critique of a completed research report.',
    default: 'claude-cli:opus',
    defaultByProvider: {
      copilot: 'copilot:claude-sonnet-4.6',
      'claude-cli': 'claude-cli:opus',
      opencode: 'opencode-go:kimi-k2.5',
      sdk: 'sdk:haiku',
      gemini: 'gemini:2.5-flash',
    } },
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

/**
 * Returns the recommended model ID for a task given the user's provider.
 * Falls back to the task's hardcoded default when the provider is not in
 * defaultByProvider, and returns null when the task itself is not found.
 */
export function getDefaultForTask(taskId, provider) {
  const task = getTask(taskId);
  return task?.defaultByProvider?.[provider] || task?.default || null;
}
