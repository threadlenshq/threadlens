import { describe, expect, it } from 'vitest';
import { getDefaultForTask, TASKS } from './catalog.js';

describe('getDefaultForTask', () => {
  it('returns the opencode-go default for post_scoring with opencode provider', () => {
    expect(getDefaultForTask('post_scoring', 'opencode')).toBe('opencode-go:deepseek-v4-flash');
  });

  it('returns the copilot default for post_scoring with copilot provider', () => {
    expect(getDefaultForTask('post_scoring', 'copilot')).toBe('copilot:gpt-5-mini');
  });

  it('falls back to task default when provider is unknown', () => {
    expect(getDefaultForTask('post_scoring', 'unknown')).toBe('copilot:gpt-5-mini');
  });

  it('returns null when task does not exist', () => {
    expect(getDefaultForTask('missing-task', 'opencode')).toBeNull();
  });

  it('returns the claude-cli default for advisor_council', () => {
    expect(getDefaultForTask('advisor_council', 'claude-cli')).toBe('claude-cli:opus');
  });
});

describe('TASKS defaultByProvider', () => {
  it('every task has a non-empty defaultByProvider object', () => {
    for (const task of TASKS) {
      expect(task.defaultByProvider).toBeDefined();
      expect(Object.keys(task.defaultByProvider).length).toBeGreaterThan(0);
    }
  });

  it('every task has all five providers in defaultByProvider', () => {
    const requiredProviders = ['copilot', 'claude-cli', 'opencode', 'sdk', 'gemini'];
    for (const task of TASKS) {
      for (const provider of requiredProviders) {
        expect(task.defaultByProvider[provider], `task ${task.id} missing provider ${provider}`).toBeDefined();
      }
    }
  });
});
