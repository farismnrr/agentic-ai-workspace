package bootstrap

import (
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/webassets"
)

// Run wires infrastructure and starts the application process.
func Run() error {
	cfg := config.Load()

	frontend, err := webassets.Assets()
	if err != nil {
		return fmt.Errorf("load frontend assets: %w", err)
	}

	db, err := database.Open(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	mounts, err := wire(db, cfg)
	if err != nil {
		return err
	}

	app := httpserver.New(frontend, mounts...)
	if err := app.Listen(cfg.Address()); err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Address(), err)
	}

	return nil
}
