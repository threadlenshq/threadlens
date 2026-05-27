CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  mode TEXT NOT NULL CHECK (mode IN ('research', 'marketing')),
  scoring_prompt TEXT,
  description TEXT,
  selected_report_id INTEGER,
  selected_cluster_index INTEGER,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_queries (
  id BIGSERIAL PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL CHECK (platform IN ('reddit', 'bluesky', 'google')),
  query_url TEXT NOT NULL,
  angle TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS project_prompts (
  id BIGSERIAL PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  type TEXT NOT NULL CHECK (type IN ('product', 'karma', 'dm')),
  platform TEXT NOT NULL CHECK (platform IN ('reddit', 'bluesky')),
  prompt_text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS posts (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL CHECK (platform IN ('reddit', 'bluesky')),
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  author TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  subreddit TEXT,
  reddit_score INTEGER,
  num_comments INTEGER,
  like_count INTEGER,
  reply_count INTEGER,
  repost_count INTEGER,
  bluesky_uri TEXT,
  bluesky_cid TEXT,
  post_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  comment_score DOUBLE PRECISION,
  final_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  angle TEXT,
  why TEXT NOT NULL DEFAULT '',
  engagement_type TEXT NOT NULL DEFAULT 'karma' CHECK (engagement_type IN ('product', 'karma')),
  karma_topic TEXT,
  top_comment_signals TEXT,
  status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'drafted', 'commented', 'skipped', 'reviewed', 'starred', 'excluded')),
  draft_comment TEXT,
  draft_provider TEXT,
  signal_type TEXT,
  created_at TIMESTAMPTZ,
  found_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  scouted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dm_targets (
  id BIGSERIAL PRIMARY KEY,
  post_id TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  username TEXT NOT NULL,
  intent_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  signal TEXT NOT NULL DEFAULT '',
  context TEXT NOT NULL DEFAULT '',
  approach TEXT NOT NULL DEFAULT '',
  draft_dm TEXT,
  draft_provider TEXT,
  dm_status TEXT NOT NULL DEFAULT 'new' CHECK (dm_status IN ('new', 'sent'))
);

CREATE TABLE IF NOT EXISTS seen_posts (
  id TEXT NOT NULL,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL,
  seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id, project_id)
);

CREATE TABLE IF NOT EXISTS scout_runs (
  id BIGSERIAL PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL CHECK (platform IN ('reddit', 'bluesky', 'google')),
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  posts_checked INTEGER NOT NULL DEFAULT 0,
  posts_found INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
  error TEXT,
  step TEXT,
  warnings TEXT
);

CREATE TABLE IF NOT EXISTS schedules (
  id BIGSERIAL PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL CHECK (platform IN ('reddit', 'bluesky', 'google')),
  cron_expr TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  last_run_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS research_reports (
  id BIGSERIAL PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  title TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
  post_count INTEGER NOT NULL DEFAULT 0,
  clusters TEXT NOT NULL DEFAULT '[]',
  assessment TEXT NOT NULL DEFAULT '',
  model_used TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  error TEXT
);

CREATE TABLE IF NOT EXISTS report_posts (
  report_id INTEGER NOT NULL REFERENCES research_reports(id) ON DELETE CASCADE,
  post_id TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  PRIMARY KEY (report_id, post_id)
);

CREATE TABLE IF NOT EXISTS google_results (
  id BIGSERIAL PRIMARY KEY,
  run_id INTEGER NOT NULL REFERENCES scout_runs(id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  root_keyword TEXT NOT NULL,
  query TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  display_url TEXT NOT NULL DEFAULT '',
  snippet TEXT NOT NULL DEFAULT '',
  rank INTEGER,
  result_type TEXT NOT NULL DEFAULT '',
  domain TEXT NOT NULL DEFAULT '',
  published_at TIMESTAMPTZ,
  author TEXT NOT NULL DEFAULT '',
  page_text TEXT NOT NULL DEFAULT '',
  content_type TEXT NOT NULL DEFAULT '',
  intent_type TEXT NOT NULL DEFAULT '',
  relevance_fit TEXT NOT NULL DEFAULT '',
  relevance_score DOUBLE PRECISION,
  confidence_score DOUBLE PRECISION,
  opportunity_types TEXT NOT NULL DEFAULT '[]',
  keepgoing_fit_reasons TEXT NOT NULL DEFAULT '[]',
  disqualifiers TEXT NOT NULL DEFAULT '[]',
  summary TEXT NOT NULL DEFAULT '',
  action_recommendation TEXT NOT NULL DEFAULT '',
  outreach_candidate INTEGER NOT NULL DEFAULT 0 CHECK (outreach_candidate IN (0, 1)),
  canonical_url TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL DEFAULT '',
  mentioned_products TEXT NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS google_keyword_summaries (
  id BIGSERIAL PRIMARY KEY,
  run_id INTEGER NOT NULL REFERENCES scout_runs(id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  root_keyword TEXT NOT NULL,
  total_results INTEGER NOT NULL DEFAULT 0,
  relevant_results INTEGER NOT NULL DEFAULT 0,
  outreach_candidates INTEGER NOT NULL DEFAULT 0,
  avg_relevance_score DOUBLE PRECISION,
  avg_confidence_score DOUBLE PRECISION,
  result_types_json TEXT NOT NULL DEFAULT '{}',
  content_types_json TEXT NOT NULL DEFAULT '{}',
  intent_types_json TEXT NOT NULL DEFAULT '{}',
  recommendation_json TEXT NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (run_id, root_keyword)
);

CREATE TABLE IF NOT EXISTS google_reports (
  id BIGSERIAL PRIMARY KEY,
  run_id INTEGER NOT NULL UNIQUE REFERENCES scout_runs(id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  executive_summary_json TEXT NOT NULL DEFAULT '{}',
  keyword_summary_json TEXT NOT NULL DEFAULT '[]',
  opportunities_json TEXT NOT NULL DEFAULT '[]',
  risks_json TEXT NOT NULL DEFAULT '[]',
  next_actions_json TEXT NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS report_councils (
  id BIGSERIAL PRIMARY KEY,
  report_id INTEGER NOT NULL UNIQUE REFERENCES research_reports(id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
  council_json TEXT NOT NULL DEFAULT '{}',
  model_used TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  error TEXT
);

CREATE TABLE IF NOT EXISTS google_domain_stats (
  id BIGSERIAL PRIMARY KEY,
  run_id INTEGER NOT NULL REFERENCES scout_runs(id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  domain TEXT NOT NULL,
  result_count INTEGER NOT NULL DEFAULT 0,
  relevant_count INTEGER NOT NULL DEFAULT 0,
  outreach_candidate_count INTEGER NOT NULL DEFAULT 0,
  avg_relevance_score DOUBLE PRECISION,
  avg_confidence_score DOUBLE PRECISION,
  top_intent_types_json TEXT NOT NULL DEFAULT '[]',
  top_content_types_json TEXT NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (run_id, domain)
);

CREATE TABLE IF NOT EXISTS query_review_jobs (
  id BIGSERIAL PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('suggest', 'refine')),
  status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
  step TEXT,
  refinement TEXT,
  result_json TEXT,
  error TEXT,
  resolution TEXT CHECK (resolution IN ('applied', 'denied')),
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  reviewed_at TIMESTAMPTZ,
  CONSTRAINT chk_query_review_jobs_review_consistency CHECK (
    (reviewed_at IS NULL AND resolution IS NULL) OR
    (reviewed_at IS NOT NULL AND resolution IS NOT NULL AND status IN ('completed', 'failed'))
  )
);

CREATE INDEX IF NOT EXISTS idx_report_councils_project ON report_councils(project_id);
CREATE INDEX IF NOT EXISTS idx_report_councils_status ON report_councils(status);
CREATE INDEX IF NOT EXISTS idx_posts_project ON posts(project_id, status);
CREATE INDEX IF NOT EXISTS idx_posts_score ON posts(project_id, final_score DESC);
CREATE INDEX IF NOT EXISTS idx_seen_project ON seen_posts(project_id, platform);
CREATE INDEX IF NOT EXISTS idx_queries_project ON project_queries(project_id);
CREATE INDEX IF NOT EXISTS idx_prompts_project ON project_prompts(project_id);
CREATE INDEX IF NOT EXISTS idx_dm_targets_post ON dm_targets(post_id);
CREATE INDEX IF NOT EXISTS idx_runs_project ON scout_runs(project_id);
CREATE INDEX IF NOT EXISTS idx_schedules_project ON schedules(project_id);
CREATE INDEX IF NOT EXISTS idx_reports_project ON research_reports(project_id);
CREATE INDEX IF NOT EXISTS idx_report_posts_report ON report_posts(report_id);
CREATE INDEX IF NOT EXISTS idx_google_results_run ON google_results(run_id);
CREATE INDEX IF NOT EXISTS idx_google_results_project ON google_results(project_id);
CREATE INDEX IF NOT EXISTS idx_google_results_domain ON google_results(run_id, domain);
CREATE INDEX IF NOT EXISTS idx_google_keyword_summaries_run ON google_keyword_summaries(run_id);
CREATE INDEX IF NOT EXISTS idx_google_reports_project ON google_reports(project_id);
CREATE INDEX IF NOT EXISTS idx_google_domain_stats_run ON google_domain_stats(run_id);
CREATE INDEX IF NOT EXISTS idx_query_review_jobs_project ON query_review_jobs(project_id, status, reviewed_at);
CREATE INDEX IF NOT EXISTS idx_query_review_jobs_started ON query_review_jobs(started_at);
