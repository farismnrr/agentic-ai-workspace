package probe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"

	"github.com/go-sql-driver/mysql"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

func (p *Probe) mysql(ctx context.Context, conn domain.Connection) error {
	cfg := mysql.NewConfig()
	cfg.User, cfg.Passwd, cfg.Net, cfg.Addr, cfg.DBName = conn.User, conn.Password, "tcp", conn.Address(), conn.Database
	if conn.TLS {
		cfg.TLSConfig = "true"
	}
	dsn := cfg.FormatDSN()
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("parse MySQL connection: %w", err)
	}
	dialer := egress.Dialer(egress.Policy{Rule: egress.PrivateAllowed, OwnPort: p.ownPort})
	parsed.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, address)
	}
	connector, err := mysql.NewConnector(parsed)
	if err != nil {
		return fmt.Errorf("create MySQL connector: %w", err)
	}
	db := sql.OpenDB(connector)
	defer func() { _ = db.Close() }()
	if err := db.PingContext(ctx); err != nil {
		return mysqlAuth(err)
	}
	return nil
}

func mysqlAuth(err error) error {
	var server *mysql.MySQLError
	if errors.As(err, &server) && (server.Number == 1044 || server.Number == 1045 || server.Number == 1698) {
		return domain.ProbeError{Category: domain.AuthFailed}
	}
	return err
}
