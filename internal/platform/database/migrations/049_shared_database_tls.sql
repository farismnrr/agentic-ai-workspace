ALTER TABLE shared_database_settings ADD COLUMN ca_cert_sealed TEXT NOT NULL DEFAULT '';
ALTER TABLE shared_database_settings ADD COLUMN client_cert_sealed TEXT NOT NULL DEFAULT '';
ALTER TABLE shared_database_settings ADD COLUMN client_key_sealed TEXT NOT NULL DEFAULT '';
