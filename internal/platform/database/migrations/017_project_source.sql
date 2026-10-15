ALTER TABLE projects ADD COLUMN source TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX projects_source_idx ON projects (user_id, source) WHERE source <> '' AND archived_at IS NULL;
