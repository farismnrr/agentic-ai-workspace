CREATE TABLE shared_database_servers (
    kind TEXT PRIMARY KEY CHECK (kind IN ('postgres', 'mysql', 'mongodb')),
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    database_name TEXT NOT NULL,
    user_name TEXT NOT NULL,
    password_sealed TEXT NOT NULL,
    ssl_mode TEXT NOT NULL,
    tls TEXT NOT NULL DEFAULT 'none' CHECK (tls IN ('none', 'ca', 'full')),
    ca_cert_sealed TEXT NOT NULL DEFAULT '',
    client_cert_sealed TEXT NOT NULL DEFAULT '',
    client_key_sealed TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL
);

INSERT INTO shared_database_servers (
    kind, host, port, database_name, user_name, password_sealed, ssl_mode,
    tls, ca_cert_sealed, client_cert_sealed, client_key_sealed, updated_at
)
SELECT
    'postgres', host, port, database_name, user_name, password_sealed, ssl_mode,
    CASE
        WHEN ssl_mode = 'disable' THEN 'none'
        WHEN client_cert_sealed != '' THEN 'full'
        ELSE 'ca'
    END,
    ca_cert_sealed, client_cert_sealed, client_key_sealed, updated_at
FROM shared_database_settings WHERE id = 1;
