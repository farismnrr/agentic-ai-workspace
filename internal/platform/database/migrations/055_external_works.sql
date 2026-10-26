ALTER TABLE tasks ADD COLUMN work_id TEXT NOT NULL DEFAULT '';

CREATE TABLE external_works (
    id         TEXT PRIMARY KEY,
    task_id    TEXT NOT NULL UNIQUE REFERENCES tasks(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    client     TEXT NOT NULL,
    state      TEXT NOT NULL CHECK (state IN
               ('open', 'blocked', 'completed', 'failed', 'stopped', 'abandoned')),
    role_key   TEXT NOT NULL DEFAULT '',
    role_name  TEXT NOT NULL DEFAULT '',
    rev        INTEGER NOT NULL DEFAULT 0,
    plan_rev   INTEGER NOT NULL DEFAULT 0,
    next_step  INTEGER NOT NULL DEFAULT 1,
    idem_key   TEXT NOT NULL DEFAULT '',
    idem_hash  TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    ended_at   INTEGER
);

CREATE UNIQUE INDEX external_works_idempotency_idx
    ON external_works(user_id, client, project_id, idem_key) WHERE idem_key <> '';
CREATE INDEX external_works_user_state_updated_idx
    ON external_works(user_id, state, updated_at);

CREATE TABLE external_work_ops (
    work_id  TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE,
    op_key   TEXT NOT NULL,
    tool     TEXT NOT NULL,
    req_hash TEXT NOT NULL,
    response BLOB NOT NULL,
    at       INTEGER NOT NULL,
    PRIMARY KEY(work_id, op_key)
);
