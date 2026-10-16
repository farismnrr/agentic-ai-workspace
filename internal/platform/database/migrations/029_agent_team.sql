-- The agent team: a person's roles (each with its own model and personality), whether new tasks use
-- the team, and how a team task keeps the work its teammates did.
CREATE TABLE agent_teams (
    user_id        TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    use_by_default INTEGER NOT NULL DEFAULT 0,
    updated_at     INTEGER NOT NULL
);

CREATE TABLE agent_roles (
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    name        TEXT NOT NULL,
    purpose     TEXT NOT NULL DEFAULT '',
    prompt      TEXT NOT NULL DEFAULT '',
    provider_id TEXT NOT NULL DEFAULT '',
    model       TEXT NOT NULL DEFAULT '',
    effort      TEXT NOT NULL DEFAULT '',
    access      TEXT NOT NULL DEFAULT '["read"]',
    lead        INTEGER NOT NULL DEFAULT 0,
    enabled     INTEGER NOT NULL DEFAULT 1,
    position    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, key)
);

ALTER TABLE tasks ADD COLUMN team INTEGER NOT NULL DEFAULT 0;

-- A teammate's conversation lives in the same table as the orchestrator's, under its own scope:
-- the id of the delegate call that started it. The empty scope is the orchestrator.
ALTER TABLE task_transcript ADD COLUMN scope TEXT NOT NULL DEFAULT '';

CREATE TABLE task_delegations (
    task_id     TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    id          TEXT NOT NULL,
    role_key    TEXT NOT NULL,
    role_name   TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    model       TEXT NOT NULL,
    effort      TEXT NOT NULL DEFAULT '',
    brief       TEXT NOT NULL,
    status      TEXT NOT NULL CHECK (status IN ('working', 'waiting', 'done', 'failed')),
    start_seq   INTEGER NOT NULL DEFAULT 0,
    report      TEXT NOT NULL DEFAULT '',
    started_at  INTEGER NOT NULL,
    ended_at    INTEGER,
    PRIMARY KEY (task_id, id)
);

CREATE INDEX task_transcript_scope_idx ON task_transcript (task_id, scope, seq);
