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
