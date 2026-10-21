CREATE TABLE shared_database_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    database_name TEXT NOT NULL,
    user_name TEXT NOT NULL,
    password_sealed TEXT NOT NULL,
    ssl_mode TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);
