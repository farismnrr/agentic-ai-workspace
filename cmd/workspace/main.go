package main

import (
	"log"

	"github.com/farismnrr/agentic-ai-workspace/internal/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		log.Fatal(err)
	}
}
