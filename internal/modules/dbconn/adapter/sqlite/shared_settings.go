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

func (s *SharedSettingsStore) Get(ctx context.Context) (application.StoredSharedSettings, error) {
	var record application.StoredSharedSettings
	err := s.db.QueryRowContext(ctx, `SELECT host, port, database_name, user_name,
		password_sealed, ssl_mode FROM shared_database_settings WHERE id = 1`).Scan(
		&record.Settings.Host, &record.Settings.Port, &record.Settings.Database, &record.Settings.User,
		&record.PasswordSealed, &record.Settings.SSLMode)
	if errors.Is(err, sql.ErrNoRows) {
		return record, application.ErrSharedSettingsMissing
	}
	if err != nil {
		return record, fmt.Errorf("load shared database settings: %w", err)
	}
	return record, nil
}

func (s *SharedSettingsStore) Save(ctx context.Context, record application.StoredSharedSettings) error {
	settings := record.Settings
	_, err := s.db.ExecContext(ctx, `INSERT INTO shared_database_settings
		(id, host, port, database_name, user_name, password_sealed, ssl_mode, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET host=excluded.host,
		port=excluded.port, database_name=excluded.database_name, user_name=excluded.user_name,
		password_sealed=excluded.password_sealed, ssl_mode=excluded.ssl_mode, updated_at=excluded.updated_at`,
		settings.Host, settings.Port, settings.Database, settings.User, record.PasswordSealed,
		settings.SSLMode, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("save shared database settings: %w", err)
	}
	return nil
}

var _ application.SharedSettingsStore = (*SharedSettingsStore)(nil)
