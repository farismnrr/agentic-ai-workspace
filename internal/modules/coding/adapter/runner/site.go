package runner

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// SiteSpec describes a long-lived command in a workspace, such as a web app: it gets the
// same cache, registry proxy and environment as any command, plus extraEnv.
func (e *Engine) SiteSpec(
	ctx context.Context, dir string, docker *sandbox.Docker, argv, extraEnv []string,
) sandbox.Spec {
	spec := sandbox.Spec{Workspace: dir, Argv: argv}
	e.prepare(ctx, dir, &spec, docker)
	spec.Env = append(environment(spec), extraEnv...)

	return spec
}
