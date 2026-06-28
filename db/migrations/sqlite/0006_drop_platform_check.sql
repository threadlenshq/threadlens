PRAGMA defer_foreign_keys=ON;

-- project_queries -----------------------------------------------------------
CREATE TABLE project_queries_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL,
  query_url TEXT NOT NULL,
  angle TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO project_queries_new (id, project_id, platform, query_url, angle, enabled, created_at)
  SELECT id, project_id, platform, query_url, angle, enabled, created_at FROM project_queries;
DROP TABLE project_queries;
ALTER TABLE project_queries_new RENAME TO project_queries;
CREATE INDEX IF NOT EXISTS idx_queries_project ON project_queries(project_id);

-- scout_runs ----------------------------------------------------------------
CREATE TABLE scout_runs_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL,
  started_at DATETIME NOT NULL DEFAULT (datetime('now')),
  completed_at DATETIME,
  posts_checked INTEGER NOT NULL DEFAULT 0,
  posts_found INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
  error TEXT,
  step TEXT,
  warnings TEXT
);
INSERT INTO scout_runs_new (id, project_id, platform, started_at, completed_at, posts_checked, posts_found, status, error, step, warnings)
  SELECT id, project_id, platform, started_at, completed_at, posts_checked, posts_found, status, error, step, warnings FROM scout_runs;
DROP TABLE scout_runs;
ALTER TABLE scout_runs_new RENAME TO scout_runs;
CREATE INDEX IF NOT EXISTS idx_runs_project ON scout_runs(project_id);

-- schedules -----------------------------------------------------------------
CREATE TABLE schedules_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL,
  cron_expr TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  last_run_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
INSERT INTO schedules_new (id, project_id, platform, cron_expr, enabled, last_run_at, created_at)
  SELECT id, project_id, platform, cron_expr, enabled, last_run_at, created_at FROM schedules;
DROP TABLE schedules;
ALTER TABLE schedules_new RENAME TO schedules;
CREATE INDEX IF NOT EXISTS idx_schedules_project ON schedules(project_id);

-- posts ---------------------------------------------------------------------
CREATE TABLE posts_new (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL,
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
  post_score REAL NOT NULL DEFAULT 0,
  comment_score REAL,
  final_score REAL NOT NULL DEFAULT 0,
  angle TEXT,
  why TEXT NOT NULL DEFAULT '',
  engagement_type TEXT NOT NULL DEFAULT 'karma' CHECK (engagement_type IN ('product', 'karma')),
  karma_topic TEXT,
  top_comment_signals TEXT,
  status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'drafted', 'commented', 'skipped', 'reviewed', 'starred', 'excluded')),
  draft_comment TEXT,
  draft_provider TEXT,
  signal_type TEXT,
  created_at DATETIME,
  found_at DATETIME NOT NULL DEFAULT (datetime('now')),
  scouted_at DATETIME NOT NULL DEFAULT (datetime('now')),
  filter_state TEXT NOT NULL DEFAULT 'visible' CHECK (filter_state IN ('visible', 'filtered')),
  filter_reason TEXT,
  filter_reasons_json TEXT NOT NULL DEFAULT '[]',
  filter_explanation TEXT NOT NULL DEFAULT '',
  filter_confidence REAL,
  filter_source TEXT NOT NULL DEFAULT 'none' CHECK (filter_source IN ('none', 'rules', 'ai', 'trusted_override')),
  filter_signature TEXT NOT NULL DEFAULT '',
  filter_job_id INTEGER,
  filtered_at DATETIME,
  recovered_at DATETIME,
  recovery_note TEXT,
  source_identity_json TEXT NOT NULL DEFAULT '{}'
);
INSERT INTO posts_new (
  id, project_id, platform, title, body, author, url, subreddit, reddit_score, num_comments,
  like_count, reply_count, repost_count, bluesky_uri, bluesky_cid, post_score, comment_score,
  final_score, angle, why, engagement_type, karma_topic, top_comment_signals, status,
  draft_comment, draft_provider, signal_type, created_at, found_at, scouted_at,
  filter_state, filter_reason, filter_reasons_json, filter_explanation, filter_confidence,
  filter_source, filter_signature, filter_job_id, filtered_at, recovered_at, recovery_note,
  source_identity_json
)
  SELECT
  id, project_id, platform, title, body, author, url, subreddit, reddit_score, num_comments,
  like_count, reply_count, repost_count, bluesky_uri, bluesky_cid, post_score, comment_score,
  final_score, angle, why, engagement_type, karma_topic, top_comment_signals, status,
  draft_comment, draft_provider, signal_type, created_at, found_at, scouted_at,
  filter_state, filter_reason, filter_reasons_json, filter_explanation, filter_confidence,
  filter_source, filter_signature, filter_job_id, filtered_at, recovered_at, recovery_note,
  source_identity_json
  FROM posts;
DROP TABLE posts;
ALTER TABLE posts_new RENAME TO posts;
CREATE INDEX IF NOT EXISTS idx_posts_project ON posts(project_id, status);
CREATE INDEX IF NOT EXISTS idx_posts_score ON posts(project_id, final_score DESC);
CREATE INDEX IF NOT EXISTS idx_posts_filter_state ON posts(project_id, filter_state, platform);
