ALTER TABLE task_events ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
UPDATE task_events SET version = seq;
CREATE INDEX task_events_version_idx ON task_events (task_id, version);
