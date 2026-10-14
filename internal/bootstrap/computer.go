package bootstrap

import (
	"context"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
)

// computer tells the agent what its sandbox has installed, as the task image advertises it.
type computer struct{ engine *codingrunner.Engine }

func (c computer) Toolchains(context.Context) []string {
	list := c.engine.Toolchains()
	out := make([]string, 0, len(list))

	for _, tool := range list {
		out = append(out, tool.Name+" "+tool.Version)
	}

	return out
}
