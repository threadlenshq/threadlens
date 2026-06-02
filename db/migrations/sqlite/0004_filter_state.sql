-- Add filter state columns to posts
ALTER TABLE posts ADD COLUMN filter_state TEXT NOT NULL DEFAULT 'visible' CHECK (filter_state IN ('visible', 'filtered'));
ALTER TABLE posts ADD COLUMN filter_reason TEXT;
ALTER TABLE posts ADD COLUMN filter_reasons_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE posts ADD COLUMN filter_explanation TEXT NOT NULL DEFAULT '';
ALTER TABLE posts ADD COLUMN filter_confidence REAL;
ALTER TABLE posts ADD COLUMN filter_source TEXT NOT NULL DEFAULT 'none' CHECK (filter_source IN ('none', 'rules', 'ai', 'trusted_override'));
ALTER TABLE posts ADD COLUMN filter_signature TEXT NOT NULL DEFAULT '';
ALTER TABLE posts ADD COLUMN filter_job_id INTEGER;
ALTER TABLE posts ADD COLUMN filtered_at DATETIME;
ALTER TABLE posts ADD COLUMN recovered_at DATETIME;
ALTER TABLE posts ADD COLUMN recovery_note TEXT;
ALTER TABLE posts ADD COLUMN source_identity_json TEXT NOT NULL DEFAULT '{}';

-- Add filter state columns to google_results
ALTER TABLE google_results ADD COLUMN filter_state TEXT NOT NULL DEFAULT 'visible' CHECK (filter_state IN ('visible', 'filtered'));
ALTER TABLE google_results ADD COLUMN filter_reason TEXT;
ALTER TABLE google_results ADD COLUMN filter_reasons_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE google_results ADD COLUMN filter_explanation TEXT NOT NULL DEFAULT '';
ALTER TABLE google_results ADD COLUMN filter_confidence REAL;
ALTER TABLE google_results ADD COLUMN filter_source TEXT NOT NULL DEFAULT 'none' CHECK (filter_source IN ('none', 'rules', 'ai', 'trusted_override'));
ALTER TABLE google_results ADD COLUMN filter_signature TEXT NOT NULL DEFAULT '';
ALTER TABLE google_results ADD COLUMN filter_job_id INTEGER;
ALTER TABLE google_results ADD COLUMN filtered_at DATETIME;
ALTER TABLE google_results ADD COLUMN recovered_at DATETIME;
ALTER TABLE google_results ADD COLUMN recovery_note TEXT;
ALTER TABLE google_results ADD COLUMN source_identity_json TEXT NOT NULL DEFAULT '{}';

-- Add new tables for filtering
CREATE TABLE IF NOT EXISTS filter_trust_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  platform TEXT NOT NULL CHECK (platform IN ('reddit', 'bluesky', 'google', 'all')),
  trust_type TEXT NOT NULL CHECK (trust_type IN ('source', 'filter_signature')),
  source_kind TEXT NOT NULL,
  source_key TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT (datetime('now')),
  created_by TEXT NOT NULL DEFAULT 'self_host_owner',
  UNIQUE(project_id, platform, trust_type, source_kind, source_key)
);

CREATE TABLE IF NOT EXISTS filter_jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
  step TEXT,
  requested_scope TEXT NOT NULL CHECK (requested_scope IN ('selected_visible_posts', 'selected_filtered_findings', 'selected_google_results')),
  target_ids_json TEXT NOT NULL DEFAULT '[]',
  result_json TEXT,
  error TEXT,
  started_at DATETIME NOT NULL DEFAULT (datetime('now')),
  completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_posts_filter_state ON posts(project_id, filter_state, platform);
CREATE INDEX IF NOT EXISTS idx_google_results_filter_state ON google_results(project_id, filter_state, domain);
CREATE INDEX IF NOT EXISTS idx_filter_trust_project ON filter_trust_records(project_id, platform, trust_type, source_kind, source_key);
CREATE INDEX IF NOT EXISTS idx_filter_jobs_project ON filter_jobs(project_id, status, started_at);
