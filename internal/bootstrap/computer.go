package bootstrap

import (
	"context"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	githubapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/github/application"
)

// computer tells the agent what its sandbox has installed, as the task image advertises it.
type computer struct {
	engine *codingrunner.Engine
	github *githubapp.Service
}

// GitHub names the account the person connected, if any.
func (c computer) GitHub(ctx context.Context, userID string) string {
	if c.github == nil {
		return ""
	}

	conn, err := c.github.Status(ctx, userID)
	if err != nil {
		return ""
	}

	return conn.Profile.Login
}

func (c computer) Toolchains(context.Context) []string {
	list := c.engine.Toolchains()
	out := make([]string, 0, len(list))

	for _, tool := range list {
		out = append(out, tool.Name+" "+tool.Version)
	}

	return out
}
