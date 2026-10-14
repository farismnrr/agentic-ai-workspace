-- SQLite cannot change a CHECK constraint in place, so the table is rebuilt to
-- allow the 'gemini' kind. No other table references providers.
CREATE TABLE providers_next (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('anthropic', 'openai', 'gemini', 'compatible')),
    name       TEXT NOT NULL,
    base_url   TEXT NOT NULL,
    key_sealed TEXT NOT NULL,
    key_hint   TEXT NOT NULL,
    verified   INTEGER NOT NULL DEFAULT 0 CHECK (verified IN (0, 1)),
    enabled    INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    models     TEXT NOT NULL DEFAULT '[]',
    created_at INTEGER NOT NULL
);

INSERT INTO providers_next SELECT id, user_id, kind, name, base_url, key_sealed, key_hint, verified, enabled, models, created_at FROM providers;

DROP TABLE providers;

ALTER TABLE providers_next RENAME TO providers;

CREATE INDEX providers_user_idx ON providers (user_id, created_at);
