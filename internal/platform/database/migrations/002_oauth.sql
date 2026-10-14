CREATE TABLE oauth_clients (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    redirect_uris TEXT NOT NULL,
    source        TEXT NOT NULL CHECK (source IN ('dcr', 'cimd')),
    created_at    INTEGER NOT NULL
);

CREATE TABLE oauth_requests (
    id             TEXT PRIMARY KEY,
    client_id      TEXT NOT NULL,
    redirect_uri   TEXT NOT NULL,
    scope          TEXT NOT NULL,
    state          TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    nonce          TEXT NOT NULL,
    resource       TEXT NOT NULL,
    expires_at     INTEGER NOT NULL
);

CREATE TABLE oauth_grants (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    client_id    TEXT NOT NULL,
    client_name  TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    resource     TEXT NOT NULL,
    scope        TEXT NOT NULL,
    created_at   INTEGER NOT NULL,
    last_used_at INTEGER,
    revoked_at   INTEGER
);

CREATE INDEX oauth_grants_user_idx ON oauth_grants (user_id);

CREATE TABLE oauth_codes (
    hash           TEXT PRIMARY KEY,
    grant_id       TEXT NOT NULL REFERENCES oauth_grants (id) ON DELETE CASCADE,
    code_challenge TEXT NOT NULL,
    nonce          TEXT NOT NULL,
    expires_at     INTEGER NOT NULL,
    consumed_at    INTEGER
);

CREATE TABLE oauth_tokens (
    hash        TEXT PRIMARY KEY,
    grant_id    TEXT NOT NULL REFERENCES oauth_grants (id) ON DELETE CASCADE,
    kind        TEXT NOT NULL CHECK (kind IN ('access', 'refresh')),
    expires_at  INTEGER NOT NULL,
    consumed_at INTEGER
);

CREATE INDEX oauth_tokens_grant_idx ON oauth_tokens (grant_id);
