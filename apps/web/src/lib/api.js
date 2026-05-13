const BASE = '';

async function api(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
    body: options.body ? JSON.stringify(options.body) : undefined,
  });
  if (options.method === 'DELETE' && res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

function normalizePagedPostsResponse(data, params = {}) {
  if (!Array.isArray(data)) return data;

  const parsedPage = Number.parseInt(params.page, 10);
  const parsedLimit = Number.parseInt(params.limit, 10);
  const page = Number.isFinite(parsedPage) && parsedPage > 0 ? parsedPage : 1;
  const limit = Number.isFinite(parsedLimit) && parsedLimit > 0 ? parsedLimit : data.length || 20;
  const total = data.length;
  const totalPages = Math.max(1, Math.ceil(total / limit));

  return {
    items: data,
    pagination: {
      page,
      limit,
      total,
      totalPages,
      hasPreviousPage: page > 1,
      hasNextPage: page < totalPages,
    },
  };
}

export const projects = {
  list: () => api('/api/projects'),
  get: (id) => api(`/api/projects/${id}`),
  create: (body) => api('/api/projects', { method: 'POST', body }),
  update: (id, body) => api(`/api/projects/${id}`, { method: 'PATCH', body }),
  delete: (id) => api(`/api/projects/${id}`, { method: 'DELETE' }),
  clone: (id, body) => api(`/api/projects/${id}/clone`, { method: 'POST', body }),
  selectAngle: (id, body) => api(`/api/projects/${id}/select-angle`, { method: 'POST', body }),
  graduate: (id) => api(`/api/projects/${id}/graduate`, { method: 'POST' }),
};

export const queries = {
  list: (pid) => api(`/api/projects/${pid}/queries`),
  create: (pid, body) => api(`/api/projects/${pid}/queries`, { method: 'POST', body }),
  update: (pid, qid, body) => api(`/api/projects/${pid}/queries/${qid}`, { method: 'PATCH', body }),
  delete: (pid, qid) => api(`/api/projects/${pid}/queries/${qid}`, { method: 'DELETE' }),
  suggest: (pid, body = {}) => api(`/api/projects/${pid}/queries/suggest`, { method: 'POST', body }),
  refine: (pid, body = {}) => api(`/api/projects/${pid}/queries/refine`, { method: 'POST', body }),
};

export const prompts = {
  list: (pid) => api(`/api/projects/${pid}/prompts`),
  create: (pid, body) => api(`/api/projects/${pid}/prompts`, { method: 'POST', body }),
  update: (pid, pid2, body) => api(`/api/projects/${pid}/prompts/${pid2}`, { method: 'PATCH', body }),
  delete: (pid, pid2) => api(`/api/projects/${pid}/prompts/${pid2}`, { method: 'DELETE' }),
};

export const posts = {
  list: (pid, params = {}) => {
    const qs = new URLSearchParams(params).toString();
    return api(`/api/projects/${pid}/posts${qs ? '?' + qs : ''}`);
  },
  listPage: (pid, params = {}) => {
    const qs = new URLSearchParams(params).toString();
    return api(`/api/projects/${pid}/posts${qs ? '?' + qs : ''}`).then(data => normalizePagedPostsResponse(data, params));
  },
  get: (pid, postId) => api(`/api/projects/${pid}/posts/${encodeURIComponent(postId)}`),
  update: (pid, postId, body) => api(`/api/projects/${pid}/posts/${encodeURIComponent(postId)}`, { method: 'PATCH', body }),
  bulkUpdate: (pid, body) => api(`/api/projects/${pid}/posts/bulk`, { method: 'PATCH', body }),
  generateDraft: (pid, postId) =>
    api(`/api/projects/${pid}/posts/${encodeURIComponent(postId)}/generate-draft`, { method: 'POST' }),
  postReply: (pid, postId, text) =>
    api(`/api/projects/${pid}/posts/${encodeURIComponent(postId)}/post-reply`, { method: 'POST', body: { text } }),
};

export const scout = {
  run: (pid, platform, wait = false) =>
    api(`/api/projects/${pid}/scout${wait ? '?wait=true' : ''}`, { method: 'POST', body: { platform } }),
  getRun: (pid, runId) => api(`/api/projects/${pid}/scout/runs/${runId}`),
  runs: (pid) => api(`/api/projects/${pid}/scout/runs`),
  cancelRun: (pid, runId) => api(`/api/projects/${pid}/scout/runs/${runId}/cancel`, { method: 'POST' }),
};

export const schedules = {
  list: (pid) => api(`/api/projects/${pid}/schedules`),
  create: (pid, body) => api(`/api/projects/${pid}/schedules`, { method: 'POST', body }),
  update: (pid, sid, body) => api(`/api/projects/${pid}/schedules/${sid}`, { method: 'PATCH', body }),
  delete: (pid, sid) => api(`/api/projects/${pid}/schedules/${sid}`, { method: 'DELETE' }),
};

export const reports = {
  list: (pid) => api(`/api/projects/${pid}/reports`),
  get: (pid, rid) => api(`/api/projects/${pid}/reports/${rid}`),
  create: (pid, body = {}) => api(`/api/projects/${pid}/reports`, { method: 'POST', body }),
  council: (pid, rid) => api(`/api/projects/${pid}/reports/${rid}/council`),
};

export const google = {
  reports: (pid) => api(`/api/projects/${pid}/google/reports`),
  report: (pid, rid) => api(`/api/projects/${pid}/google/reports/${rid}`),
  latest: (pid) => api(`/api/projects/${pid}/google/reports/latest`),
  keywordSummaries: (pid, rid) => api(`/api/projects/${pid}/google/reports/${rid}/keywords`),
  rankedResults: (pid, rid, { mode = 'seo', limit } = {}) => {
    const params = new URLSearchParams();
    if (mode) params.set('mode', mode);
    if (limit !== undefined && limit !== null && limit !== '') params.set('limit', String(limit));
    const qs = params.toString();
    return api(`/api/projects/${pid}/google/reports/${rid}/results${qs ? `?${qs}` : ''}`);
  },
};

export const models = {
  catalog: () => api('/api/models/catalog'),
  config: () => api('/api/models/config'),
  setTask: (taskId, modelId) => api(`/api/models/config/${taskId}`, { method: 'PUT', body: { modelId } }),
  resetTask: (taskId) => api(`/api/models/config/${taskId}`, { method: 'DELETE' }),
};

export const runtime = {
  capabilities: () => api('/api/runtime/capabilities'),
};

export const templates = {
  promptPacks: () => api('/api/templates/prompt-packs'),
};

export const onboarding = {
  status: () => api('/api/onboarding/status'),
  requiredStep: (body) => api('/api/onboarding/required-step', { method: 'POST', body }),
  save: (body) => api('/api/onboarding/save', { method: 'POST', body }),
  exploration: (body) => api('/api/onboarding/exploration', { method: 'POST', body }),
  starterProject: (body) => api('/api/onboarding/starter-project', { method: 'POST', body }),
  reset: (mode = 'progress') => api('/api/onboarding/reset', { method: 'POST', body: { mode } }),
};
