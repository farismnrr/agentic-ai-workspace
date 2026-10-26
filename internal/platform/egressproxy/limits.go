package egressproxy

import (
	"context"
	"time"
)

// Limits bound what a grant can hold open.
type Limits struct {
	// PerScope and Total are the most open tunnels for one task and for the whole registry.
	PerScope, Total int
	// Handshake is how long a client may take to send its ClientHello after the tunnel opens.
	Handshake time.Duration
	// Idle closes a tunnel that moved no bytes for this long.
	Idle time.Duration
	// Dial bounds name resolution plus the connection to the destination.
	Dial time.Duration
}

// DefaultLimits are the limits used unless a test says otherwise.
func DefaultLimits() Limits {
	const (
		perScope  = 16
		total     = 128
		handshake = 10 * time.Second
		idle      = 5 * time.Minute
		dial      = 10 * time.Second
	)

	return Limits{PerScope: perScope, Total: total, Handshake: handshake, Idle: idle, Dial: dial}
}

// Verifier is the last word on a connection, asked on every CONNECT that matched a live grant. An error refuses the
// connection: when the source of truth cannot answer, nothing opens.
type Verifier interface {
	Verify(ctx context.Context, grantID string, scope Scope, host string) error
}
