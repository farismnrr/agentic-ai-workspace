package main

import (
	"log"
	"os"

	"github.com/farismnrr/agentic-ai-workspace/internal/bootstrap"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

func main() {
	sandbox.RunIfChild() // a sandboxed task re-enters this binary as its init

	run := bootstrap.Run
	if len(os.Args) > 1 && os.Args[1] == "sandbox" {
		run = func() error { return bootstrap.SandboxCommand(os.Args[2:]) }
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}
