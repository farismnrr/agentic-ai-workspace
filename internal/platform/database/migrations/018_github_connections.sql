CREATE TABLE github_connections (
    user_id         TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    github_id       INTEGER NOT NULL,
    login           TEXT NOT NULL,
    name            TEXT NOT NULL,
    access_sealed   TEXT NOT NULL,
    refresh_sealed  TEXT NOT NULL,
    access_expires  INTEGER NOT NULL,
    refresh_expires INTEGER NOT NULL,
    created_at      INTEGER NOT NULL
);
