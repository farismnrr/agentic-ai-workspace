CREATE TABLE project_gateway (
    project_id  TEXT PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    provider_id TEXT NOT NULL,
    updated_at  INTEGER NOT NULL
);
