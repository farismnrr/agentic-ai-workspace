CREATE TABLE project_env_reveal_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    variable_name TEXT NOT NULL,
    outcome TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX project_env_reveal_audit_created_at
    ON project_env_reveal_audit(created_at);
