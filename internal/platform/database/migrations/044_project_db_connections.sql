CREATE TABLE project_db_connections (
    project_id      TEXT PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    type            TEXT NOT NULL CHECK (type IN ('postgres', 'mysql', 'mongodb')),
    host            TEXT NOT NULL,
    port            INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    database_name   TEXT NOT NULL,
    user_name       TEXT NOT NULL,
    password_sealed TEXT NOT NULL,
    tls             INTEGER NOT NULL CHECK (tls IN (0, 1)),
    env_name        TEXT NOT NULL,
    updated_at      INTEGER NOT NULL
);
