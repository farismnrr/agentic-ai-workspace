ALTER TABLE project_db_connections
ADD COLUMN managed INTEGER NOT NULL DEFAULT 0 CHECK (managed IN (0, 1));
