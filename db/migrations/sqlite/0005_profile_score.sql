-- Add profile score and profile signals columns to dm_targets
ALTER TABLE dm_targets ADD COLUMN profile_score REAL;
ALTER TABLE dm_targets ADD COLUMN profile_signals TEXT;
