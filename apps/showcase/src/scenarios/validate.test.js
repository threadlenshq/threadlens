import { describe, expect, it } from 'vitest';
import { validateScenarios } from './validate.js';

const previews = { ExamplePreview: {} };

const validScenario = {
  id: 'empty-and-setup-states',
  title: 'Empty and setup states',
  description: 'Use when a workflow has no data yet.',
  group: 'Workflow states',
  examples: [
    {
      id: 'first-run-empty-panel',
      title: 'First-run empty panel',
      preview: 'ExamplePreview',
      notes: ['Use calm copy that names the next product step.'],
    },
  ],
};

describe('validateScenarios', () => {
  it('accepts a valid manifest and preview registry', () => {
    expect(validateScenarios([validScenario], previews)).toBe(true);
  });

  it('rejects duplicate scenario ids', () => {
    expect(() => validateScenarios([validScenario, validScenario], previews)).toThrow(
      'Duplicate scenario id "empty-and-setup-states"'
    );
  });

  it('rejects duplicate example ids across scenarios', () => {
    const secondScenario = {
      title: validScenario.title,
      description: validScenario.description,
      group: validScenario.group,
      id: 'finding-review-cards',
      examples: [
        {
          id: validScenario.examples[0].id,
          title: validScenario.examples[0].title,
          preview: validScenario.examples[0].preview,
          notes: validScenario.examples[0].notes,
        },
      ],
    };

    expect(() => validateScenarios([validScenario, secondScenario], previews)).toThrow(
      'Duplicate example id "first-run-empty-panel"'
    );
  });

  it('rejects missing preview keys', () => {
    const scenario = {
      id: validScenario.id,
      title: validScenario.title,
      description: validScenario.description,
      group: validScenario.group,
      examples: [
        {
          id: validScenario.examples[0].id,
          title: validScenario.examples[0].title,
          preview: 'MissingPreview',
          notes: validScenario.examples[0].notes,
        },
      ],
    };

    expect(() => validateScenarios([scenario], previews)).toThrow(
      'references missing preview "MissingPreview"'
    );
  });

  it('rejects scenarios with no examples', () => {
    const scenario = {
      id: validScenario.id,
      title: validScenario.title,
      description: validScenario.description,
      group: validScenario.group,
      examples: [],
    };

    expect(() => validateScenarios([scenario], previews)).toThrow(
      'must include at least one example'
    );
  });
});
