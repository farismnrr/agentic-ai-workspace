package probe

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

func (p *Probe) postgres(ctx context.Context, conn domain.Connection) error {
	config, err := pgxpool.ParseConfig(conn.URL())
	if err != nil {
		return fmt.Errorf("parse PostgreSQL connection: %w", err)
	}
	config.ConnConfig.DialFunc = egress.Dialer(egress.Policy{Rule: egress.PrivateAllowed, OwnPort: p.ownPort}).DialContext
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return postgresAuth(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return postgresAuth(err)
	}
	return nil
}

func postgresAuth(err error) error {
	var server *pgconn.PgError
	if errors.As(err, &server) && len(server.Code) >= 2 && server.Code[:2] == "28" {
		return domain.ProbeError{Category: domain.AuthFailed}
	}
	return err
}
