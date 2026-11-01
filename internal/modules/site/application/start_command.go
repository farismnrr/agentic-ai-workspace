package application

import (
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// SiteArgv accepts only a single recognized application server invocation.
// Runtime executes argv directly, never through bash -c. Operators, variable
// expansion, and embedded commands are unsupported.
func SiteArgv(command string) ([]string, error) {
	if strings.ContainsAny(command, "\n\r;&|<>\x60$()\\\"'") {
		return nil, domain.ErrBadCommand
	}

	argv := strings.Fields(command)
	if len(argv) < 2 {
		return nil, domain.ErrBadCommand
	}

	allowed := false
	switch argv[0] {
	case "npm", "pnpm", "yarn", "bun":
		action := argv[1]
		if action == "run" && len(argv) > 2 {
			action = argv[2]
		}
		allowed = action == "start" || action == "preview" || action == "serve"
	case "node", "nodejs":
		allowed = strings.HasSuffix(argv[1], ".js") || strings.HasSuffix(argv[1], ".mjs") ||
			strings.HasSuffix(argv[1], ".cjs")
	case "python", "python3":
		allowed = len(argv) >= 3 && argv[1] == "-m" &&
			(argv[2] == "http.server" || argv[2] == "uvicorn")
	case "go":
		allowed = argv[1] == "run" && len(argv) >= 3
	case "next":
		allowed = argv[1] == "start"
	case "vite", "astro":
		allowed = argv[1] == "preview"
	case "npx":
		allowed = len(argv) >= 3 &&
			((argv[1] == "vite" && argv[2] == "preview") ||
				(argv[1] == "next" && argv[2] == "start") ||
				(argv[1] == "astro" && argv[2] == "preview"))
	case "uvicorn", "gunicorn", "http-server":
		allowed = true
	case "php":
		allowed = argv[1] == "-S"
	}

	if !allowed {
		return nil, domain.ErrBadCommand
	}

	for _, arg := range argv[1:] {
		// No shell syntax is evaluated, but reject obvious attempts to
		// repurpose a site start as a test operation.
		if arg == "test" || arg == "pytest" {
			return nil, domain.ErrBadCommand
		}
	}

	return argv, nil
}
