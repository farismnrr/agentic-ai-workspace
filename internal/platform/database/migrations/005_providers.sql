CREATE TABLE providers (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('anthropic', 'openai', 'compatible')),
    name       TEXT NOT NULL,
    base_url   TEXT NOT NULL,
    key_sealed TEXT NOT NULL,
    key_hint   TEXT NOT NULL,
    verified   INTEGER NOT NULL DEFAULT 0 CHECK (verified IN (0, 1)),
    enabled    INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    models     TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL
);

CREATE INDEX providers_user_idx ON providers (user_id, created_at);
