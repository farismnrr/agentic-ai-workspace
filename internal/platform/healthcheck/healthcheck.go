// Package healthcheck lets the binary ask a running copy of itself whether it is serving,
// so a container health check needs no curl or shell in the distroless image.
package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const timeout = 4 * time.Second

var errUnhealthy = errors.New("health endpoint did not answer 200")

// Probe requests /api/health on the loopback address of the given port.
func Probe(ctx context.Context, port string) error {
	return probeURL(ctx, "http://"+net.JoinHostPort("127.0.0.1", port)+"/api/health")
}

func probeURL(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build health request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("reach health endpoint: %w", err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", errUnhealthy, res.StatusCode)
	}

	return nil
}
