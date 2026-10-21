package probe

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

var errGenericProbe = errors.New("driver details must not be returned")

func TestFailureCategories(t *testing.T) {
	blocked := egress.ErrBlocked
	var refused error = &net.OpError{Err: syscall.ECONNREFUSED}
	var missing error = &net.DNSError{Err: "no such host", Name: "private.invalid", IsNotFound: true}
	cases := []struct {
		name string
		err  error
		want domain.FailureCategory
	}{
		{name: "blocked address", err: blocked, want: domain.Blocked},
		{name: "refused", err: refused, want: domain.Refused},
		{name: "host missing", err: missing, want: domain.HostNotFound},
		{name: "timeout", err: context.DeadlineExceeded, want: domain.TimedOut},
		{name: "generic", err: errGenericProbe, want: domain.Other},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := classify(test.err); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestEveryDriverBlocksMetadataAddressAtDialTime(t *testing.T) {
	prober := New("8081")
	for _, kind := range []domain.Kind{domain.Postgres, domain.MySQL, domain.MongoDB} {
		t.Run(string(kind), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			conn := domain.Connection{
				Kind: kind, Host: "169.254.169.254", Port: 27017,
				Database: "test", User: "user", Password: "secret", EnvName: "DATABASE_URL",
			}
			err := prober.Probe(ctx, conn)
			var failure domain.ProbeError
			if !errors.As(err, &failure) || failure.Category != domain.Blocked {
				t.Fatalf("%s probe must stop at the dial policy, got %v", kind, err)
			}
			if ctx.Err() != nil {
				t.Fatalf("%s probe waited for the timeout", kind)
			}
		})
	}
}
