CREATE TABLE project_env_vars (
    project_id   TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    value_sealed TEXT NOT NULL,
    is_secret    INTEGER NOT NULL CHECK (is_secret IN (0, 1)),
    updated_at   INTEGER NOT NULL,
    PRIMARY KEY (project_id, name)
);
