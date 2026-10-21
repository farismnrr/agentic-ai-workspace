package egress_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

func TestPublicOnlyRefusesEverythingInternal(t *testing.T) {
	t.Parallel()

	policy := egress.Policy{Rule: egress.PublicOnly}
	blocked := []string{
		"127.0.0.1:80", "[::1]:80", "10.0.0.5:80", "172.16.3.4:80", "192.168.1.9:80", "169.254.169.254:80",
		"100.64.0.1:80", "0.0.0.0:80", "224.0.0.1:80", "[fe80::1]:80", "[fc00::1]:80", "[::ffff:127.0.0.1]:80",
		"[::ffff:10.0.0.1]:80", "[64:ff9b::7f00:1]:80", "192.0.0.8:80", "240.0.0.1:80", "not-an-ip:80", "1.2.3.4",
	}

	for _, address := range blocked {
		if err := egress.Check(policy, address); !errors.Is(err, egress.ErrBlocked) {
			t.Errorf("%s must be blocked, got %v", address, err)
		}
	}

	for _, address := range []string{"8.8.8.8:443", "[2606:4700:4700::1111]:443", "104.18.0.1:443"} {
		if err := egress.Check(policy, address); err != nil {
			t.Errorf("%s is public and must be allowed: %v", address, err)
		}
	}
}

func TestLocalOnlyAllowsLoopbackButNotThisServerOrTheNetwork(t *testing.T) {
	t.Parallel()

	policy := egress.Policy{Rule: egress.LocalOnly, OwnPort: "8081"}

	for _, address := range []string{"127.0.0.1:20128", "[::1]:20128", "[::ffff:127.0.0.1]:20128"} {
		if err := egress.Check(policy, address); err != nil {
			t.Errorf("%s is this computer and must be allowed: %v", address, err)
		}
	}

	blocked := []string{"127.0.0.1:8081", "[::1]:8081", "192.168.1.9:20128", "10.0.0.1:20128", "8.8.8.8:443"}
	for _, address := range blocked {
		if err := egress.Check(policy, address); !errors.Is(err, egress.ErrBlocked) {
			t.Errorf("%s must be blocked, got %v", address, err)
		}
	}
}

func TestPrivateAllowed(t *testing.T) {
	t.Parallel()

	policy := egress.Policy{Rule: egress.PrivateAllowed, OwnPort: "8081"}
	cases := []struct {
		address string
		blocked bool
	}{
		{address: "8.8.8.8:443"},
		{address: "10.0.0.5:5432"},
		{address: "172.16.3.4:3306"},
		{address: "192.168.1.9:27017"},
		{address: "127.0.0.1:5432"},
		{address: "[::1]:5432"},
		{address: "[fc00::1]:5432"},
		{address: "169.254.169.254:80", blocked: true},
		{address: "169.254.1.2:80", blocked: true},
		{address: "[fe80::1]:80", blocked: true},
		{address: "[::ffff:169.254.169.254]:80", blocked: true},
		{address: "0.0.0.0:80", blocked: true},
		{address: "[::]:80", blocked: true},
		{address: "224.0.0.1:80", blocked: true},
		{address: "[ff02::1]:80", blocked: true},
		{address: "100.64.0.1:80", blocked: true},
		{address: "192.0.0.8:80", blocked: true},
		{address: "[2001:db8::1]:80", blocked: true},
		{address: "[fd00:ec2::254]:80", blocked: true},
		{address: "127.0.0.1:8081", blocked: true},
		{address: "[::1]:8081", blocked: true},
	}

	for _, test := range cases {
		err := egress.Check(policy, test.address)
		if test.blocked && !errors.Is(err, egress.ErrBlocked) {
			t.Errorf("%s must be blocked, got %v", test.address, err)
		}
		if !test.blocked && err != nil {
			t.Errorf("%s must be allowed, got %v", test.address, err)
		}
	}
}

func TestPrivateAllowedDialControlRejectsResolvedMetadataAddress(t *testing.T) {
	t.Parallel()

	dialer := egress.Dialer(egress.Policy{Rule: egress.PrivateAllowed, OwnPort: "8081"})
	err := dialer.Control("tcp", "169.254.169.254:5432", nil)
	if !errors.Is(err, egress.ErrBlocked) {
		t.Fatalf("resolved metadata address must be blocked at dial time, got %v", err)
	}
}

func TestTransportReallyRefusesToConnect(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("internal secret"))
	}))
	t.Cleanup(server.Close)

	port := server.Listener.Addr().(*net.TCPAddr).Port //nolint:forcetypeassert // httptest listens on TCP

	cases := map[string]egress.Policy{
		"public only":      {Rule: egress.PublicOnly},
		"this server port": {Rule: egress.LocalOnly, OwnPort: strconv.Itoa(port)},
	}

	for name, policy := range cases {
		client := &http.Client{Transport: egress.Transport(policy)}

		request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
		if response, err := client.Do(request); err == nil {
			_ = response.Body.Close()
			t.Errorf("%s: the connection to %s must be blocked", name, server.URL)
		}
	}

	allowed := &http.Client{Transport: egress.Transport(egress.Policy{Rule: egress.LocalOnly})}

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)

	response, err := allowed.Do(request)
	if err != nil {
		t.Fatalf("a local gateway must be reachable: %v", err)
	}

	_ = response.Body.Close()
}
