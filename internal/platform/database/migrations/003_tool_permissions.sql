CREATE TABLE tool_permissions (
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    space      TEXT NOT NULL,
    tool       TEXT NOT NULL,
    enabled    INTEGER NOT NULL CHECK (enabled IN (0, 1)),
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (user_id, space, tool)
);
