package egressproxy

import (
	"context"
	"maps"
	"net"
	"strings"
	"syscall"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

// WithOwnPort makes exact targets refuse this server's own port, so a database setting can never
// point a sandbox at the server itself.
func (p *Proxy) WithOwnPort(port string) *Proxy {
	next := *p
	next.dialTarget = targetDial(port)

	return &next
}

func targetDial(ownPort string) dialFunc {
	dialer := &net.Dialer{
		Timeout: dialTimeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			return egress.Check(egress.Policy{Rule: egress.PrivateAllowed, OwnPort: ownPort}, address)
		},
	}

	return func(ctx context.Context, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", address)
	}
}

// WithHosts returns a proxy that allows the extra entries as well; the original is unchanged. A plain
// name is an HTTPS host. A "host:port" entry is an exact target: that name and that port, nothing else.
func (p *Proxy) WithHosts(extra []string) *Proxy {
	next := *p
	next.hosts = maps.Clone(p.hosts)
	next.targets = maps.Clone(p.targets)

	if next.hosts == nil {
		next.hosts = map[string]struct{}{}
	}

	if next.targets == nil {
		next.targets = map[string]struct{}{}
	}

	for _, entry := range extra {
		entry = strings.ToLower(strings.TrimSpace(entry))

		switch host, port, err := net.SplitHostPort(entry); {
		case entry == "":
		case err == nil && host != "" && port != "":
			next.targets[net.JoinHostPort(host, port)] = struct{}{}
		default:
			next.hosts[entry] = struct{}{}
		}
	}

	return &next
}

func (p *Proxy) isTarget(host, port string) bool {
	_, ok := p.targets[net.JoinHostPort(strings.ToLower(host), port)]

	return ok
}
