package main

import (
	"log"

	"github.com/farismnrr/agentic-ai-workspace/internal/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/server"
	webassets "github.com/farismnrr/agentic-ai-workspace/internal/web"
)

func main() {
	cfg := config.Load()

	assets, err := webassets.Assets()
	if err != nil {
		log.Fatalf("load embedded web assets: %v", err)
	}

	app := server.New(assets)

	log.Printf("agentic-ai-workspace listening on %s", cfg.Address())
	log.Fatal(app.Listen(cfg.Address()))
}
