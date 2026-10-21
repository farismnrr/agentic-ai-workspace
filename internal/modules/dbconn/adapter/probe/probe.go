package probe

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

type Probe struct{ ownPort string }

func New(ownPort string) *Probe { return &Probe{ownPort: ownPort} }

func (p *Probe) Probe(ctx context.Context, conn domain.Connection) error {
	var err error
	switch conn.Kind {
	case domain.Postgres:
		err = p.postgres(ctx, conn)
	case domain.MySQL:
		err = p.mysql(ctx, conn)
	case domain.MongoDB:
		err = p.mongo(ctx, conn)
	default:
		return domain.ProbeError{Category: domain.Other}
	}
	if err == nil {
		return nil
	}
	return domain.ProbeError{Category: classify(err)}
}

func classify(err error) domain.FailureCategory {
	if isBlocked(err) {
		return domain.Blocked
	}
	if isMissingHost(err) {
		return domain.HostNotFound
	}
	if isRefused(err) {
		return domain.Refused
	}
	if isTimeout(err) {
		return domain.TimedOut
	}
	var tagged domain.ProbeError
	if errors.As(err, &tagged) {
		return tagged.Category
	}
	return classifyMessage(err.Error())
}

func classifyMessage(raw string) domain.FailureCategory {
	message := strings.ToLower(raw)
	if containsOne(message, "tls", "certificate", "x509") {
		return domain.TLSFailed
	}
	if containsOne(message, "authentication failed", "bad auth", "access denied") {
		return domain.AuthFailed
	}
	if strings.Contains(message, "connection refused") {
		return domain.Refused
	}
	return classifyHostMessage(message)
}

func classifyHostMessage(message string) domain.FailureCategory {
	if strings.Contains(message, "server selection timeout") {
		return domain.TimedOut
	}
	if strings.Contains(message, "no such host") {
		return domain.HostNotFound
	}
	return domain.Other
}

func containsOne(message string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(message, phrase) {
			return true
		}
	}
	return false
}

func isBlocked(err error) bool { return errors.Is(err, egress.ErrBlocked) }

func isMissingHost(err error) bool {
	var dns *net.DNSError
	return errors.As(err, &dns) && dns.IsNotFound
}

func isRefused(err error) bool { return errors.Is(err, syscall.ECONNREFUSED) }

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) && netErr.Timeout()
}
