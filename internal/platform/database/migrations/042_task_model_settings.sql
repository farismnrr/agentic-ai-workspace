-- Existing app tasks follow Settings on their next run. Explicit API selections made after this migration stay pinned.
ALTER TABLE tasks ADD COLUMN follow_settings INTEGER NOT NULL DEFAULT 1;
