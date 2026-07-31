-- Rebuild dm_targets to add four-state CHECK and dm_status_updated_at.
-- SQLite cannot ALTER a CHECK constraint, so we rebuild the table in-place.
CREATE TABLE dm_targets_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_id TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  username TEXT NOT NULL,
  intent_score REAL NOT NULL DEFAULT 0,
  signal TEXT NOT NULL DEFAULT '',
  context TEXT NOT NULL DEFAULT '',
  approach TEXT NOT NULL DEFAULT '',
  draft_dm TEXT,
  draft_provider TEXT,
  dm_status TEXT NOT NULL DEFAULT 'new' CHECK (dm_status IN ('new', 'sent', 'replied', 'ignored')),
  profile_score REAL,
  profile_signals TEXT,
  dm_status_updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO dm_targets_new (id, post_id, username, intent_score, signal, context,
  approach, draft_dm, draft_provider, dm_status,
  profile_score, profile_signals, dm_status_updated_at)
SELECT id, post_id, username, intent_score, signal, context,
  approach, draft_dm, draft_provider, dm_status,
  profile_score, profile_signals, datetime('now')
FROM dm_targets;

DROP TABLE dm_targets;

ALTER TABLE dm_targets_new RENAME TO dm_targets;

CREATE INDEX IF NOT EXISTS idx_dm_targets_post ON dm_targets(post_id);
