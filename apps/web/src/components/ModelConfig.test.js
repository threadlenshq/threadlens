// @vitest-environment jsdom
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import ModelConfig from './ModelConfig.svelte';
import { models as modelsApi } from '../lib/api.js';

vi.mock('../lib/api.js', () => ({
  models: {
    catalog: vi.fn(),
    config: vi.fn(),
    setTask: vi.fn(),
    resetTask: vi.fn(),
  },
}));

const baseModels = [
  { id: 'sdk:haiku', provider: 'sdk', model: 'haiku', label: 'Claude Haiku (SDK)', tier: 'low', cost: 'token-billed' },
  { id: 'opencode-go:kimi-k2.5', provider: 'opencode-go', model: 'kimi-k2.5', label: 'Kimi K2.5 (opencode Go)', tier: 'reasoning', cost: 'subscription' },
];

const baseTasks = [
  { id: 'post_scoring', label: 'Post Scoring', complexity: 'low', volume: 'high', timeoutMs: 60000, description: 'Test task', default: 'sdk:haiku' },
];

describe('ModelConfig recommended badge', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    modelsApi.config.mockResolvedValue({});
  });

  it('renders ✦ next to options matching currentProvider', async () => {
    modelsApi.catalog.mockResolvedValue({
      models: baseModels,
      tasks: baseTasks,
      managedProvider: { available: false },
      currentProvider: 'sdk',
    });

    render(ModelConfig);

    // Wait for the async load to complete.
    const options = await screen.findAllByRole('option');
    const sdkOption = options.find(o => o.value === 'sdk:haiku');
    expect(sdkOption.textContent).toContain('✦ recommended');
  });

  it('does not render ✦ next to options from a different provider', async () => {
    modelsApi.catalog.mockResolvedValue({
      models: baseModels,
      tasks: baseTasks,
      managedProvider: { available: false },
      currentProvider: 'sdk',
    });

    render(ModelConfig);

    const options = await screen.findAllByRole('option');
    const opencodeOption = options.find(o => o.value === 'opencode-go:kimi-k2.5');
    expect(opencodeOption.textContent).not.toContain('✦ recommended');
  });

  it('falls back to sdk when currentProvider is missing from response', async () => {
    modelsApi.catalog.mockResolvedValue({
      models: baseModels,
      tasks: baseTasks,
      managedProvider: { available: false },
      // currentProvider intentionally omitted
    });

    render(ModelConfig);

    const options = await screen.findAllByRole('option');
    const sdkOption = options.find(o => o.value === 'sdk:haiku');
    expect(sdkOption.textContent).toContain('✦ recommended');
  });
});
