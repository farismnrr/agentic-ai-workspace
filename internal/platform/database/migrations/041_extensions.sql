-- The browser extension the AI built in a project and packaged for the person to install. One per project; the
-- package itself is a zip in the data folder.
CREATE TABLE extensions (
    project_id  TEXT PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    slug        TEXT NOT NULL,
    name        TEXT NOT NULL,
    version     TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    permissions TEXT NOT NULL DEFAULT '[]',
    hosts       TEXT NOT NULL DEFAULT '[]',
    size        INTEGER NOT NULL,
    files       INTEGER NOT NULL,
    deployed_at INTEGER NOT NULL
);

CREATE INDEX extensions_user_idx ON extensions (user_id, deployed_at DESC);
