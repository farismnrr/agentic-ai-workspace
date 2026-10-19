package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"

// program is how the server asks one vendor's CLI to do the three things it needs.
type program struct {
	login, status, logout []string
	env                   []string
}

func programOf(rt domain.Runtime) program {
	if rt == domain.ChatGPT {
		return program{
			login:  []string{"codex", "login", "--device-auth"},
			status: []string{"codex", "login", "status"},
			logout: []string{"codex", "logout"},
		}
	}

	return program{
		login:  []string{"claude", "auth", "login", "--claudeai"},
		status: []string{"claude", "auth", "status"},
		logout: []string{"claude", "auth", "logout"},
		env:    []string{"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1", "DISABLE_AUTOUPDATER=1"},
	}
}
