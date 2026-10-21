package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
)

type SharedSettingsStore struct{ db *sql.DB }

func NewSharedSettingsStore(db *sql.DB) *SharedSettingsStore {
	return &SharedSettingsStore{db: db}
}

func (s *SharedSettingsStore) Get(ctx context.Context, kind string) (application.StoredSharedSettings, error) {
	var record application.StoredSharedSettings
	err := s.db.QueryRowContext(ctx, `SELECT host, port, database_name, user_name,
		password_sealed, ssl_mode, tls, ca_cert_sealed, client_cert_sealed, client_key_sealed
		FROM shared_database_servers WHERE kind = ?`, kind).Scan(
		&record.Settings.Host, &record.Settings.Port, &record.Settings.Database, &record.Settings.User,
		&record.PasswordSealed, &record.Settings.SSLMode, &record.Settings.TLS, &record.CACertSealed,
		&record.ClientCertSealed, &record.ClientKeySealed)
	if errors.Is(err, sql.ErrNoRows) {
		return record, application.ErrSharedSettingsMissing
	}
	if err != nil {
		return record, fmt.Errorf("load shared database settings: %w", err)
	}
	record.Settings.Kind = kind
	return record, nil
}

func (s *SharedSettingsStore) Save(ctx context.Context, kind string, record application.StoredSharedSettings) error {
	settings := record.Settings
	_, err := s.db.ExecContext(ctx, `INSERT INTO shared_database_servers
		(kind, host, port, database_name, user_name, password_sealed, ssl_mode, tls,
		 ca_cert_sealed, client_cert_sealed, client_key_sealed, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(kind) DO UPDATE SET host=excluded.host,
		port=excluded.port, database_name=excluded.database_name, user_name=excluded.user_name,
		password_sealed=excluded.password_sealed, ssl_mode=excluded.ssl_mode, tls=excluded.tls,
		ca_cert_sealed=excluded.ca_cert_sealed, client_cert_sealed=excluded.client_cert_sealed,
		client_key_sealed=excluded.client_key_sealed, updated_at=excluded.updated_at`,
		kind, settings.Host, settings.Port, settings.Database, settings.User, record.PasswordSealed,
		settings.SSLMode, settings.TLS, record.CACertSealed, record.ClientCertSealed, record.ClientKeySealed,
		time.Now().Unix())
	if err != nil {
		return fmt.Errorf("save shared database settings: %w", err)
	}
	return nil
}

var _ application.SharedSettingsStore = (*SharedSettingsStore)(nil)
