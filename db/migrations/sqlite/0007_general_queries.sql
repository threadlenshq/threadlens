CREATE TABLE IF NOT EXISTS general_queries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  query_text TEXT NOT NULL,
  angle TEXT NOT NULL,
  platforms TEXT NOT NULL DEFAULT '[]',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

ALTER TABLE project_queries ADD COLUMN general_query_id INTEGER
  REFERENCES general_queries(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_general_queries_project ON general_queries(project_id);
CREATE INDEX IF NOT EXISTS idx_project_queries_general ON project_queries(general_query_id);
