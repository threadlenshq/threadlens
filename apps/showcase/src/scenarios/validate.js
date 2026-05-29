const KEBAB_CASE = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

function assertKebabCase(kind, id) {
  if (!KEBAB_CASE.test(id)) {
    throw new Error(`${kind} id "${id}" must be kebab-case.`);
  }
}

function assertUnique(kind, id, seen) {
  if (seen.has(id)) {
    throw new Error(`Duplicate ${kind} id "${id}" in showcase manifest.`);
  }
  seen.add(id);
}

export function validateScenarios(scenarios, previewRegistry) {
  if (!Array.isArray(scenarios)) {
    throw new Error('Showcase scenarios must be an array.');
  }

  if (scenarios.length === 0) {
    throw new Error('Showcase manifest has no scenarios. Add at least one usage scenario.');
  }

  const scenarioIds = new Set();
  const exampleIds = new Set();

  for (const scenario of scenarios) {
    assertKebabCase('Scenario', scenario.id);
    assertUnique('scenario', scenario.id, scenarioIds);

    if (!scenario.title || !scenario.description || !scenario.group) {
      throw new Error(`Scenario "${scenario.id}" needs title, description, and group.`);
    }

    if (!Array.isArray(scenario.examples) || scenario.examples.length === 0) {
      throw new Error(`Scenario "${scenario.id}" must include at least one example.`);
    }

    for (const example of scenario.examples) {
      assertKebabCase('Example', example.id);
      assertUnique('example', example.id, exampleIds);

      if (!example.title || !example.preview) {
        throw new Error(`Example "${example.id}" needs title and preview.`);
      }

      if (!previewRegistry[example.preview]) {
        throw new Error(`Example "${example.id}" references missing preview "${example.preview}".`);
      }

      if (!Array.isArray(example.notes) || example.notes.length === 0) {
        throw new Error(`Example "${example.id}" must include usage notes.`);
      }
    }
  }

  return true;
}
