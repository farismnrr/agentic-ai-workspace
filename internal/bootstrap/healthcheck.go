package bootstrap

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/healthcheck"
)

// Healthcheck asks the running server on this machine whether it is serving. The container's
// HEALTHCHECK runs it as `agentic-ai-workspace healthcheck`.
func Healthcheck(ctx context.Context) error {
	if err := healthcheck.Probe(ctx, config.Load().Port); err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}

	return nil
}
