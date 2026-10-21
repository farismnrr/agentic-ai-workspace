package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Get(ctx context.Context, projectID string) (application.StoredConnection, error) {
	var record application.StoredConnection
	var kind string
	var tlsFlag int
	var managedFlag int
	err := s.db.QueryRowContext(ctx, `SELECT type, host, port, database_name, user_name,
		password_sealed, tls, env_name, managed FROM project_db_connections WHERE project_id = ?`,
		projectID).Scan(&kind, &record.Connection.Host, &record.Connection.Port,
		&record.Connection.Database, &record.Connection.User, &record.PasswordSealed, &tlsFlag,
		&record.Connection.EnvName, &managedFlag)
	if errors.Is(err, sql.ErrNoRows) {
		return record, domain.ErrMissing
	}
	if err != nil {
		return record, fmt.Errorf("load project database connection: %w", err)
	}
	record.Connection.Kind = domain.Kind(kind)
	record.Connection.TLS = tlsFlag != 0
	record.Connection.Managed = managedFlag != 0
	return record, nil
}

func (s *Store) Save(ctx context.Context, projectID string, record application.StoredConnection) error {
	conn := record.Connection
	_, err := s.db.ExecContext(ctx, `INSERT INTO project_db_connections
		(project_id, type, host, port, database_name, user_name, password_sealed, tls, env_name, managed, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id) DO UPDATE SET type=excluded.type, host=excluded.host, port=excluded.port,
		database_name=excluded.database_name, user_name=excluded.user_name,
		password_sealed=excluded.password_sealed, tls=excluded.tls, env_name=excluded.env_name,
		managed=excluded.managed,
		updated_at=excluded.updated_at`, projectID, conn.Kind, conn.Host, conn.Port, conn.Database,
		conn.User, record.PasswordSealed, conn.TLS, conn.EnvName, conn.Managed, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("save project database connection: %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, projectID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM project_db_connections WHERE project_id = ?`, projectID)
	if err != nil {
		return fmt.Errorf("delete project database connection: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check database connection deletion: %w", err)
	}
	if count == 0 {
		return domain.ErrMissing
	}
	return nil
}
