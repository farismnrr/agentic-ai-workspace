CREATE TABLE tasks (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects (id),
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    summary     TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL CHECK (status IN
                ('queued', 'planning', 'working', 'needs-input', 'review', 'done', 'failed')),
    mode        TEXT NOT NULL CHECK (mode IN ('risky', 'always')),
    provider_id TEXT NOT NULL,
    model       TEXT NOT NULL,
    plan        TEXT NOT NULL DEFAULT '[]',
    proof       TEXT,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE INDEX tasks_user_idx ON tasks (user_id, updated_at DESC);
CREATE INDEX tasks_project_idx ON tasks (project_id, updated_at DESC);

CREATE TABLE task_events (
    task_id     TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    seq         INTEGER NOT NULL,
    id          TEXT NOT NULL UNIQUE,
    at          INTEGER NOT NULL,
    kind        TEXT NOT NULL,
    tool_use_id TEXT NOT NULL DEFAULT '',
    body        TEXT NOT NULL,
    undo        BLOB,
    PRIMARY KEY (task_id, seq)
);

CREATE INDEX task_events_tool_idx ON task_events (task_id, tool_use_id);

CREATE TABLE task_transcript (
    task_id TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    seq     INTEGER NOT NULL,
    role    TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    block   TEXT NOT NULL,
    PRIMARY KEY (task_id, seq)
);

CREATE TABLE task_usage (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id       TEXT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    user_id       TEXT NOT NULL,
    provider_id   TEXT NOT NULL,
    model         TEXT NOT NULL,
    input_tokens  INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    at            INTEGER NOT NULL
);

CREATE INDEX task_usage_user_idx ON task_usage (user_id, at);
CREATE INDEX task_usage_task_idx ON task_usage (task_id);
