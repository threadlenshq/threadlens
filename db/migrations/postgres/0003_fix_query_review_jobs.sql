DROP TABLE IF EXISTS query_review_jobs;

CREATE TABLE IF NOT EXISTS query_review_jobs (
  id           BIGSERIAL PRIMARY KEY,
  project_id   TEXT     NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  kind         TEXT     NOT NULL CHECK(kind IN ('suggest','refine')),
  status       TEXT     NOT NULL DEFAULT 'running'
                        CHECK(status IN ('running','completed','failed')),
  step         TEXT     NOT NULL DEFAULT '',
  refinement   TEXT,
  result_json  TEXT,
  error        TEXT,
  resolution   TEXT     CHECK(resolution IN ('applied','denied')),
  reviewed_at  TIMESTAMPTZ,
  started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_query_review_jobs_project ON query_review_jobs(project_id, status, reviewed_at);
CREATE INDEX IF NOT EXISTS idx_query_review_jobs_started ON query_review_jobs(started_at);
