// src/lib/url.js

const DEFAULTS = {
  project: null,
  view: 'posts', // posts | settings | sources | reports | models
  reportSource: 'social',
  platform: 'all',
  status: 'new',
  dm: false,
  score: '',
  page: 1,
  limit: 20,
  post: null,
  report: null,
  greport: null,
  tab: 'general',
};

/**
 * Read all URL state from current search params.
 * Returns an object with keys matching DEFAULTS, using defaults for missing params.
 */
export function readUrlState() {
  const params = new URLSearchParams(window.location.search);
  return {
    project: params.get('project') || DEFAULTS.project,
    view: params.get('view') || DEFAULTS.view,
    reportSource: params.get('reportSource') || DEFAULTS.reportSource,
    platform: params.get('platform') || DEFAULTS.platform,
    status: params.get('status') || DEFAULTS.status,
    dm: params.get('dm') === 'true',
    score: params.get('score') || DEFAULTS.score,
    page: params.get('page') ? Number(params.get('page')) : DEFAULTS.page,
    limit: params.get('limit') ? Number(params.get('limit')) : DEFAULTS.limit,
    post: params.get('post') ? Number(params.get('post')) : DEFAULTS.post,
    report: params.get('report') ? Number(params.get('report')) : DEFAULTS.report,
    greport: params.get('greport') ? Number(params.get('greport')) : DEFAULTS.greport,
    tab: params.get('tab') || DEFAULTS.tab,
  };
}

/**
 * Write state to URL search params.
 * Only includes non-default values to keep URLs clean.
 * @param {Partial<typeof DEFAULTS>} state - key/value pairs to set
 * @param {'replace'|'push'} mode - history mode
 */
export function writeUrlState(state, mode = 'replace') {
  const params = new URLSearchParams();

  // Merge current URL state with new state
  const current = readUrlState();
  const merged = { ...current, ...state };

  // Only write non-default values
  if (merged.project) params.set('project', merged.project);
  if (merged.view && merged.view !== DEFAULTS.view) params.set('view', merged.view);
  if (merged.reportSource && merged.reportSource !== DEFAULTS.reportSource) params.set('reportSource', merged.reportSource);
  if (merged.platform && merged.platform !== DEFAULTS.platform) params.set('platform', merged.platform);
  if (merged.status && merged.status !== DEFAULTS.status) params.set('status', merged.status);
  if (merged.dm) params.set('dm', 'true');
  if (merged.score && merged.score !== DEFAULTS.score) params.set('score', merged.score);
  if (merged.page && merged.page !== DEFAULTS.page) params.set('page', String(merged.page));
  if (merged.limit && merged.limit !== DEFAULTS.limit) params.set('limit', String(merged.limit));
  if (merged.post) params.set('post', String(merged.post));
  if (merged.report) params.set('report', String(merged.report));
  if (merged.greport) params.set('greport', String(merged.greport));
  if (merged.tab && merged.tab !== DEFAULTS.tab) params.set('tab', merged.tab);

  const qs = params.toString();
  const newUrl = `${window.location.pathname}${qs ? '?' + qs : ''}`;

  if (mode === 'push') {
    window.history.pushState({}, '', newUrl);
  } else {
    window.history.replaceState({}, '', newUrl);
  }
}

/**
 * Clear specific keys from URL state (reset them to defaults).
 * @param {string[]} keys - keys to remove
 */
export function clearUrlState(keys) {
  const reset = {};
  for (const key of keys) {
    reset[key] = DEFAULTS[key];
  }
  writeUrlState(reset);
}
