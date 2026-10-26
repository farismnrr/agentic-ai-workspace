-- Extra outbound HTTPS hosts one task's sandbox may reach once a person approves them. Off until the feature is
-- enabled; see agents/egress-grants.md. A row is scoped to one task of one project.
CREATE TABLE egress_grants (
    id                TEXT PRIMARY KEY,
    project_id        TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    task_id           TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    host              TEXT NOT NULL,
    reason            TEXT NOT NULL,
    status            TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'denied', 'revoked')),
    requested_by_kind TEXT NOT NULL CHECK (requested_by_kind IN ('agent', 'person')),
    requested_by      TEXT NOT NULL,
    decided_by        TEXT NOT NULL DEFAULT '',
    created_at        INTEGER NOT NULL,
    decided_at        INTEGER NOT NULL DEFAULT 0,
    expires_at        INTEGER NOT NULL
);

CREATE INDEX egress_grants_task_idx ON egress_grants (project_id, task_id, status);
