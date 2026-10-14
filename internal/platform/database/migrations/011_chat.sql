CREATE TABLE project_chats (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX project_chats_user_project_idx ON project_chats (user_id, project_id, updated_at DESC);

CREATE TABLE chat_messages (
    id         TEXT PRIMARY KEY,
    chat_id    TEXT NOT NULL REFERENCES project_chats (id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX chat_messages_chat_idx ON chat_messages (chat_id, created_at ASC);
