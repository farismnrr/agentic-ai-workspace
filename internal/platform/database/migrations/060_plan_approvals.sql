CREATE TABLE external_plan_decisions (
    id          TEXT PRIMARY KEY,
    revision_id TEXT NOT NULL UNIQUE REFERENCES external_plan_revisions(id) ON DELETE CASCADE,
    work_id     TEXT NOT NULL REFERENCES external_works(id) ON DELETE CASCADE,
    task_id     TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    decision    TEXT NOT NULL CHECK (decision IN ('approved', 'changes_requested')),
    source      TEXT NOT NULL CHECK (source IN ('app', 'mcp')),
    client      TEXT NOT NULL DEFAULT '',
    idem_key    TEXT NOT NULL DEFAULT '',
    req_hash    TEXT NOT NULL,
    note        TEXT NOT NULL DEFAULT '',
    decided_at  INTEGER NOT NULL
);

CREATE UNIQUE INDEX external_plan_decision_idempotency_idx
    ON external_plan_decisions(work_id, idem_key) WHERE idem_key <> '';

CREATE TABLE external_plan_handoffs (
    id               TEXT PRIMARY KEY,
    decision_id      TEXT NOT NULL UNIQUE REFERENCES external_plan_decisions(id) ON DELETE CASCADE,
    work_id          TEXT NOT NULL UNIQUE REFERENCES external_works(id) ON DELETE CASCADE,
    state            TEXT NOT NULL CHECK (state IN ('pending', 'created', 'done', 'failed')),
    attempts         INTEGER NOT NULL DEFAULT 0,
    next_attempt_at  INTEGER NOT NULL,
    last_error       TEXT NOT NULL DEFAULT '',
    internal_task_id TEXT REFERENCES tasks(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX external_plan_handoff_task_idx
    ON external_plan_handoffs(internal_task_id) WHERE internal_task_id IS NOT NULL;
CREATE INDEX external_plan_handoff_due_idx
    ON external_plan_handoffs(state, next_attempt_at);
