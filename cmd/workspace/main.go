package main

import (
	"context"
	"log"
	"os"

	"github.com/farismnrr/agentic-ai-workspace/internal/bootstrap"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := bootstrap.Healthcheck(context.Background()); err != nil {
			log.Print(err)
			os.Exit(1)
		}

		return
	}

	if err := bootstrap.Run(); err != nil {
		log.Fatal(err)
	}
}
