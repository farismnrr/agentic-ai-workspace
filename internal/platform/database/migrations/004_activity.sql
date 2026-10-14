CREATE TABLE activity (
    id        TEXT PRIMARY KEY,
    user_id   TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    space     TEXT NOT NULL,
    tool      TEXT NOT NULL,
    summary   TEXT NOT NULL,
    outcome   TEXT NOT NULL CHECK (outcome IN ('ok', 'failed', 'off', 'forbidden')),
    detail    TEXT NOT NULL,
    at        INTEGER NOT NULL,
    undo      BLOB,
    undone_at INTEGER
);

CREATE INDEX activity_user_idx ON activity (user_id, at DESC);
