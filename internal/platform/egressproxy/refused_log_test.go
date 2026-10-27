package egressproxy

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestProxyRecordsTheHostItRefuses(t *testing.T) {
	var logged bytes.Buffer

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	path := serve(t, New([]string{"pypi.org"}))

	if got := connect(t, path, "productionresultssa1.blob.core.windows.net:443"); got != http.StatusForbidden {
		t.Fatalf("CONNECT = %d, want %d", got, http.StatusForbidden)
	}

	for _, want := range []string{"egress refused a host", "productionresultssa1.blob.core.windows.net", "443"} {
		if !strings.Contains(logged.String(), want) {
			t.Errorf("the log should say %q, got %q", want, logged.String())
		}
	}
}
