ALTER TABLE oauth_clients ADD COLUMN secret_hash TEXT;
ALTER TABLE oauth_clients ADD COLUMN auth_method TEXT NOT NULL DEFAULT 'none';
