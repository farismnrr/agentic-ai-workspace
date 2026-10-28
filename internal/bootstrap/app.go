package bootstrap

import (
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/webassets"
)

// Run wires infrastructure and starts the application process. version is the build's version.
func Run(version string) error {
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

	mounts, stop, err := Wire(db, cfg)
	if err != nil {
		return err
	}

	defer stop()

	defer database.StartSweeper(db, time.Hour)()

	app := httpserver.New(frontend, httpserver.Settings{ClientIPHeader: cfg.ClientIPHeader, Version: version}, mounts...)
	if err := app.Listen(cfg.Address()); err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Address(), err)
	}

	return nil
}
