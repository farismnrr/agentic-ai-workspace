package probe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

const mongoSelectionTimeout = 8 * time.Second

func (p *Probe) mongo(ctx context.Context, conn domain.Connection) error {
	policy := egress.Policy{Rule: egress.PrivateAllowed, OwnPort: p.ownPort}
	probeCtx, cancel := context.WithCancel(ctx)
	dialer := &mongoDialer{policy: policy, dialer: egress.Dialer(policy), cancel: cancel}
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(conn.URL()).SetDialer(dialer).
		SetServerSelectionTimeout(mongoSelectionTimeout))
	if err != nil {
		return mongoAuth(err)
	}
	defer func(ctx context.Context) {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_ = client.Disconnect(cleanup)
	}(ctx)
	if err := client.Ping(probeCtx, readpref.Primary()); err != nil {
		if blocked := dialer.blockedError(); blocked != nil {
			return blocked
		}
		return mongoAuth(err)
	}
	return nil
}

type mongoDialer struct {
	blocked error
	dialer  *net.Dialer
	cancel  context.CancelFunc
	policy  egress.Policy
	mu      sync.Mutex
}

func (d *mongoDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, d.block(egress.ErrBlocked)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve MongoDB host: %w", err)
	}
	if len(ips) == 0 {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	for _, candidate := range ips {
		resolved := net.JoinHostPort(candidate.IP.String(), port)
		if checkErr := egress.Check(d.policy, resolved); checkErr != nil {
			return nil, d.block(checkErr)
		}
	}
	conn, err := d.dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	if err != nil {
		return nil, fmt.Errorf("dial MongoDB host: %w", err)
	}
	return conn, nil
}

func (d *mongoDialer) block(err error) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.blocked == nil {
		d.blocked = err
		d.cancel()
	}
	return d.blocked
}

func (d *mongoDialer) blockedError() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.blocked
}

func mongoAuth(err error) error {
	var command mongo.CommandError
	if errors.As(err, &command) && (command.Code == 18 || command.Name == "AuthenticationFailed") {
		return domain.ProbeError{Category: domain.AuthFailed}
	}
	return err
}
