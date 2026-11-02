-- External worker cancellation is durable and must survive process restarts.
CREATE TABLE task_delegations_next (
    task_id     TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    id          TEXT NOT NULL,
    role_key    TEXT NOT NULL,
    role_name   TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    model       TEXT NOT NULL,
    effort      TEXT NOT NULL DEFAULT '',
    brief       TEXT NOT NULL,
    status      TEXT NOT NULL CHECK (status IN ('working', 'waiting', 'done', 'failed', 'cancelled')),
    start_seq   INTEGER NOT NULL DEFAULT 0,
    report      TEXT NOT NULL DEFAULT '',
    started_at  INTEGER NOT NULL,
    ended_at    INTEGER,
    PRIMARY KEY (task_id, id)
);

INSERT INTO task_delegations_next (
    task_id, id, role_key, role_name, provider_id, model, effort, brief,
    status, start_seq, report, started_at, ended_at
)
SELECT
    task_id, id, role_key, role_name, provider_id, model, effort, brief,
    status, start_seq, report, started_at, ended_at
FROM task_delegations;

DROP TABLE task_delegations;
ALTER TABLE task_delegations_next RENAME TO task_delegations;
