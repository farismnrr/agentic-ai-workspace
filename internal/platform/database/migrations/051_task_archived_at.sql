ALTER TABLE tasks ADD COLUMN archived_at INTEGER;

CREATE INDEX tasks_active_user_idx ON tasks (user_id, updated_at DESC) WHERE archived_at IS NULL;
