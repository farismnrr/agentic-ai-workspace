CREATE TABLE preferences (
    user_id             TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    default_provider_id TEXT NOT NULL DEFAULT '',
    default_model       TEXT NOT NULL DEFAULT '',
    persona_name        TEXT NOT NULL DEFAULT '',
    talk_style          TEXT NOT NULL DEFAULT '',
    language            TEXT NOT NULL DEFAULT '',
    about               TEXT NOT NULL DEFAULT '',
    rules               TEXT NOT NULL DEFAULT '[]',
    updated_at          INTEGER NOT NULL
);
