package egressproxy

import (
	"net/http"
	"net/netip"
	"testing"
	"time"
)

const goodHost = "sb.example.com"

func TestAGrantedTaskTunnelsItsHelloAndTraffic(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)

	c := r.open(scopeA, goodHost+":443")
	hello := helloFor(t, goodHost)

	if c.status != http.StatusOK || !c.echoes(hello) {
		t.Fatalf("status %d", c.status)
	}

	if got := <-r.dialed; got != publicIP {
		t.Fatalf("dialed %v", got)
	}

	if !c.echoes([]byte("later traffic")) || r.received.len() != len(hello)+len("later traffic") {
		t.Fatal("the hello must reach the destination once, unchanged, and traffic must follow")
	}
}

func TestAGrantServesOnlyItsOwnTaskSocket(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)

	for name, scope := range map[string]Scope{"another task": scopeB, "another project": scopeC} {
		if c := r.open(scope, goodHost+":443"); c.status != http.StatusForbidden {
			t.Errorf("%s: %d", name, c.status)
		}
	}

	if r.lookups.Load() != 0 || len(r.dialed) != 0 {
		t.Fatal("a refused request must not even resolve the name")
	}
}

func TestEveryScopeHasItsOwnSocketFolder(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	seen := map[string]Scope{}

	for _, scope := range []Scope{scopeA, scopeB, scopeC} {
		dir, err := r.hub.TaskDir(t.Context(), "ws", scope)
		if err != nil || seen[dir] != (Scope{}) {
			t.Fatalf("%v: %v (shared with %v)", scope, err, seen[dir])
		}

		seen[dir] = scope
	}
}

func TestWithoutAGrantNothingOpens(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)

	for _, target := range []string{
		"other.example.com:443", "example.com:443", "127.0.0.1:443", "localhost:443", "[::1]:443",
		"[::ffff:127.0.0.1]:443", "2130706433:443",
	} {
		if c := r.open(scopeA, target); c.status != http.StatusForbidden {
			t.Errorf("%s: %d", target, c.status)
		}
	}
}

func TestOnlyPort443AndOnlyNamesAreServed(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)

	for _, target := range []string{goodHost + ":80", goodHost + ":8443", goodHost + ":22", goodHost} {
		if c := r.open(scopeA, target); c.status != http.StatusForbidden {
			t.Errorf("%s: %d", target, c.status)
		}
	}

	if c := r.open(scopeA, "SB.Example.COM.:443"); c.status != http.StatusOK {
		t.Errorf("a differently spelled granted name: %d", c.status)
	}
}

func TestResolvedAddressesMustAllBePublic(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)

	for _, bad := range []string{
		"127.0.0.1", "10.1.2.3", "192.168.0.9", "172.16.0.1", "169.254.169.254", "100.64.0.1", "0.0.0.0", "::1",
		"fc00::1", "fe80::1", "::ffff:127.0.0.1", "::ffff:10.0.0.1", "64:ff9b::7f00:1", "2002:7f00:1::1",
	} {
		ip := netip.MustParseAddr(bad)
		r.answers = func(int) []netip.Addr { return []netip.Addr{ip} }

		if c := r.open(scopeA, goodHost+":443"); c.status != http.StatusForbidden {
			t.Errorf("%s: %d", bad, c.status)
		}

		r.answers = func(int) []netip.Addr { return []netip.Addr{publicIP, ip} }

		if c := r.open(scopeA, goodHost+":443"); c.status != http.StatusForbidden {
			t.Errorf("public plus %s: %d", bad, c.status)
		}
	}

	if len(r.dialed) != 0 {
		t.Fatal("nothing may be dialed when any answer is not public")
	}
}

func TestAnUnresolvableNameIsABadGateway(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
	r.allow("g1", scopeA, time.Minute)
	r.answers = func(int) []netip.Addr { return nil }

	if c := r.open(scopeA, goodHost+":443"); c.status != http.StatusBadGateway {
		t.Fatalf("%d", c.status)
	}
}

func TestADNSAnswerThatChangesAfterTheCheckChangesNothing(t *testing.T) {
	r := newRig(t, DefaultLimits(), nil)
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
