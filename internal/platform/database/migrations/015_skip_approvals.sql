CREATE TABLE approval_settings (
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    space      TEXT NOT NULL,
    skip       INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (user_id, space)
);
