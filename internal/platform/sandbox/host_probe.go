package sandbox

import (
	"context"
	"fmt"
	"time"

	"github.com/moby/moby/client"
)

// ProbeHost checks the Docker daemon reachable by the trusted app process.
// It does not run containers, inspect images, alter daemon state, or expose the socket.
func ProbeHost(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	api, err := client.New(client.FromEnv)
	if err != nil {
		return fmt.Errorf("%w: could not initialize Docker Engine client: %w", ErrUnavailable, err)
	}
	defer api.Close()

	if _, err := api.Ping(ctx, client.PingOptions{}); err != nil {
		return fmt.Errorf("%w: Docker daemon socket or permissions: %w", ErrUnavailable, err)
	}

	return nil
}
