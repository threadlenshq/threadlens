export function shouldShowRequiredWizard(status) {
  if (!status || status.enabled === false) return false;
  return status.phase === 'required_setup' || status.requiredSetupComplete === false;
}

export function shouldShowExploration(status) {
  if (!status || status.enabled === false) return false;
  return status.requiredSetupComplete === true && status.explorationComplete === false && status.phase === 'exploration';
}

export function firstIncompleteStep(steps = [], fallback = 'welcome') {
  const found = steps.find((step) => !step.completed);
  return found?.id || fallback;
}

export function explorationFinished(items = []) {
  if (!items.length) return false;
  return items.every((item) => item.state === 'completed' || item.state === 'skipped');
}

export function normalizeOnboardingStatus(status) {
  return {
    enabled: status?.enabled !== false,
    phase: status?.phase || 'required_setup',
    requiredSetupComplete: !!status?.requiredSetupComplete,
    explorationComplete: !!status?.explorationComplete,
    currentRequiredStep: status?.currentRequiredStep || firstIncompleteStep(status?.steps, 'welcome'),
    currentExplorationItem: status?.currentExplorationItem || 'starter_project',
    steps: Array.isArray(status?.steps) ? status.steps : [],
    items: Array.isArray(status?.items) ? status.items : [],
    capabilities: status?.capabilities || { providers: [], sources: {} },
    appDatabase: status?.appDatabase || {},
    context: status?.context || {},
  };
}

export function activationProgressLabel(status) {
  if (!status || status.enabled === false) return 'Onboarding disabled';
  if (status.phase === 'complete' || status.explorationComplete === true) return 'Activated';
  if (status.phase === 'required_setup' || status.requiredSetupComplete === false) return 'Finish setup';
  const item = status.currentExplorationItem;
  if (item === 'starter_project') return 'Create project';
  if (item === 'starter_query') return 'Add one query';
  if (item === 'first_scout') return 'Run first scout';
  if (item === 'review_results') return 'Review findings';
  if (item === 'reports_intro') return 'Open reports';
  if (item === 'settings_intro') return 'Visit models';
  return 'Keep exploring';
}
