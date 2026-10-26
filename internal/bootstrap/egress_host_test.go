package bootstrap

import (
	"testing"

	egressdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

// A grant stores the domain's spelling and the proxy compares its own, so for every name the domain accepts the
// proxy must produce the very same spelling. Otherwise an approved name could never match (safe, but broken) or,
// worse, a different name could. The proxy accepting more than the domain is fine: nothing is stored for those.
func TestTheProxyAndTheDomainSpellHostsTheSame(t *testing.T) {
	for _, in := range []string{
		"sb-f6c7943aa05fb36c.sb.molab.run", "Example.COM", "example.com.", "bücher.example", "xn--bcher-kva.example",
		"example。com", "a.b.c.d.example.org", "x-y.example.io", "UPPER.CASE.EXAMPLE", "127.0.0.1", "localhost", "a..b.com",
		"*.example.com", "example.com:443", "1.2.3.4.example", "０.example.com", "ℬ.example.com", "xn--a.example.com",
	} {
		want, err := egressdomain.NormalizeHost(in)
		if err != nil {
			continue
		}

		if got, ok := egress.CanonicalHost(in); !ok || got != want {
			t.Errorf("%q: the domain says %q, the proxy says %q (%v)", in, want, got, ok)
		}
	}
}
