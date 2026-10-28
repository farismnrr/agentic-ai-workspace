package egressproxy_test

import (
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egressproxy"
)

const goodHost = "sb.example.com"

func TestADNSAnswerThatChangesAfterTheCheckChangesNothing(t *testing.T) {
	r := newRig(t, egressproxy.DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)
	r.answers = func(call int) []netip.Addr {
		if call == 1 {
			return []netip.Addr{publicIP}
		}

		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}
	}

	c := r.open(scopeA, goodHost+":443")
	if c.status != http.StatusOK || !c.echoes(helloFor(t, goodHost)) {
		t.Fatalf("status %d", c.status)
	}

	if got := <-r.dialed; got != publicIP || r.lookups.Load() != 1 {
		t.Fatalf("dialed %v after %d lookups: the checked address must be the one used", got, r.lookups.Load())
	}
}
