-- One web app per project that the AI started in the sandbox. token is the random UUID in
-- the address /run/<slug>/<token>/ and is replaced on every start.
CREATE TABLE sites (
    project_id   TEXT PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    slug         TEXT NOT NULL,
    token        TEXT NOT NULL UNIQUE,
    visibility   TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
    status       TEXT NOT NULL CHECK (status IN ('running', 'stopped')),
    port         INTEGER NOT NULL,
    container_id TEXT NOT NULL DEFAULT '',
    started_at   INTEGER NOT NULL
);

CREATE INDEX sites_user_idx ON sites (user_id, started_at);
