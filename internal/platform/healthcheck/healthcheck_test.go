package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeURLAcceptsOnlyOK(t *testing.T) {
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}))

	defer server.Close()

	if err := probeURL(context.Background(), server.URL); err != nil {
		t.Fatalf("200 should be healthy: %v", err)
	}

	status = http.StatusServiceUnavailable
	if err := probeURL(context.Background(), server.URL); err == nil {
		t.Fatal("503 should be unhealthy")
	}
}

func TestProbeURLFailsWhenNothingListens(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()

	if err := probeURL(context.Background(), url); err == nil {
		t.Fatal("a closed port should be unhealthy")
	}
}
