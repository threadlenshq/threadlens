export const noop = () => {};

export const mockProject = {
  id: 'ai-note-taking',
  name: 'AI Note Taking Research',
  mode: 'research',
};

export const mockProjects = [
  mockProject,
  { id: 'solo-founder-crm', name: 'Solo Founder CRM', mode: 'marketing' },
];

export const mockActiveRuns = [
  {
    id: 'run_static_reddit_001',
    platform: 'reddit',
    status: 'running',
    started_at: '2026-05-29T09:35:00.000Z',
  },
];

export const mockReport = {
  id: 'report_static_001',
  title: 'Recurring Meeting Notes Pain Points',
};

export const mockFindings = [
  {
    id: 'reddit_high_signal_static',
    platform: 'reddit',
    status: 'starred',
    title: 'I spend more time cleaning up meeting notes than doing follow-up work',
    body: 'Every tool captures a transcript, but I still have to rewrite action items and send summaries manually.',
    url: 'https://reddit.example/thread/high-signal',
    created_at: '2026-05-28T18:20:00.000Z',
    reddit_score: 142,
    num_comments: 38,
    final_score: 9.2,
    post_score: 9.2,
    signal_type: 'frustration',
    engagement_type: 'product',
    dm_targets: ['ops-lead@example.com'],
  },
  {
    id: 'bluesky_selected_static',
    platform: 'bluesky',
    status: 'new',
    title: 'Need a lightweight way to turn calls into project updates without another dashboard',
    body: 'A tiny workflow that sends action items to the tools I already use would beat a full meeting suite.',
    url: 'https://bsky.example/profile/founder/post/selected',
    created_at: '2026-05-29T08:05:00.000Z',
    like_count: 27,
    reply_count: 6,
    final_score: 7.4,
    post_score: 7.4,
    signal_type: 'seeking_solution',
    engagement_type: 'product',
    dm_targets: [],
  },
  {
    id: 'reddit_low_signal_static',
    platform: 'reddit',
    status: 'excluded',
    title: 'What note app do you use?',
    body: 'Mostly looking for general recommendations without a specific workflow pain.',
    url: 'https://reddit.example/thread/low-signal',
    created_at: '2026-05-26T12:00:00.000Z',
    reddit_score: 8,
    num_comments: 3,
    final_score: 2.1,
    post_score: 2.1,
    signal_type: null,
    engagement_type: 'karma',
    dm_targets: [],
  },
];
