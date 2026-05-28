CREATE TABLE IF NOT EXISTS query_review_jobs (
  id           INTEGER  PRIMARY KEY AUTOINCREMENT,
  project_id   TEXT     NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  kind         TEXT     NOT NULL CHECK(kind IN ('suggest','refine')),
  status       TEXT     NOT NULL DEFAULT 'running'
                        CHECK(status IN ('running','completed','failed')),
  step         TEXT     NOT NULL DEFAULT '',
  refinement   TEXT,
  result_json  TEXT,
  error        TEXT,
  resolution   TEXT     CHECK(resolution IN ('applied','denied')),
  reviewed_at  DATETIME,
  started_at   DATETIME NOT NULL DEFAULT (datetime('now')),
  completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_query_review_jobs_project ON query_review_jobs(project_id, status, reviewed_at);
CREATE INDEX IF NOT EXISTS idx_query_review_jobs_started ON query_review_jobs(started_at);
