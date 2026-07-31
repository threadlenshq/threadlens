-- Add dm_status_updated_at and expand dm_status CHECK to four states.
-- Postgres supports adding a column and modifying constraints via ALTER TABLE.
ALTER TABLE dm_targets ADD COLUMN IF NOT EXISTS dm_status_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Drop the old two-state CHECK and add the new four-state one.
-- We use a named constraint so we can reference it.
ALTER TABLE dm_targets DROP CONSTRAINT IF EXISTS dm_targets_dm_status_check;
ALTER TABLE dm_targets ADD CONSTRAINT dm_targets_dm_status_check CHECK (dm_status IN ('new', 'sent', 'replied', 'ignored'));
