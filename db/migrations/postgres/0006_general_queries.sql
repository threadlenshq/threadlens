CREATE TABLE IF NOT EXISTS general_queries (
  id BIGSERIAL PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  query_text TEXT NOT NULL,
  angle TEXT NOT NULL,
  platforms TEXT NOT NULL DEFAULT '[]',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE project_queries ADD COLUMN IF NOT EXISTS general_query_id BIGINT
  REFERENCES general_queries(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_general_queries_project ON general_queries(project_id);
CREATE INDEX IF NOT EXISTS idx_project_queries_general ON project_queries(general_query_id);
