package provision

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

func connect(ctx context.Context, adminURL, ownPort string) (session, error) {
	cfg, err := pgx.ParseConfig(adminURL)
	if err != nil {
		return nil, fmt.Errorf("parse admin connection: %w", err)
	}
	cfg.DialFunc = egress.Dialer(egress.Policy{Rule: egress.PrivateAllowed, OwnPort: ownPort}).DialContext
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect to admin database: %w", err)
	}
	return pgxSession{conn: conn}, nil
}

type pgxSession struct{ conn *pgx.Conn }

func (s pgxSession) Begin(ctx context.Context) (transaction, error) {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin admin transaction: %w", err)
	}
	return pgxTransaction{tx: tx}, nil
}

func (s pgxSession) Close(ctx context.Context) error {
	if err := s.conn.Close(ctx); err != nil {
		return fmt.Errorf("close admin connection: %w", err)
	}
	return nil
}

type pgxTransaction struct{ tx pgx.Tx }

func (t pgxTransaction) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tag, err := t.tx.Exec(ctx, sql, args...)
	if err != nil {
		return tag, fmt.Errorf("execute provisioning statement: %w", err)
	}
	return tag, nil
}

func (t pgxTransaction) QueryRow(ctx context.Context, sql string, args ...any) row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t pgxTransaction) Commit(ctx context.Context) error {
	if err := t.tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit database transaction: %w", err)
	}
	return nil
}

func (t pgxTransaction) Rollback(ctx context.Context) error {
	if err := t.tx.Rollback(ctx); err != nil {
		return fmt.Errorf("rollback database transaction: %w", err)
	}
	return nil
}
