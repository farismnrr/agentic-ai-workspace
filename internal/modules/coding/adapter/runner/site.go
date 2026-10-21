package runner

import (
	"context"
	"fmt"
	"io"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// SiteSpec describes a long-lived command in a workspace, such as a web app: it gets the
// same cache, registry proxy and environment as any command, plus extraEnv. Its proxy also reaches hosts.
func (e *Engine) SiteSpec(
	ctx context.Context, dir string, docker *sandbox.Docker, argv, extraEnv []string, hosts ...string,
) sandbox.Spec {
	spec := sandbox.Spec{Workspace: dir, Argv: argv}
	e.prepare(ctx, dir, &spec, docker, hosts...)
	spec.Env = append(environment(spec), extraEnv...)

	return spec
}

// RunOnce runs a short command to its end in a place of the shared pool, for work the server does itself, such
// as photographing a running app. Output is copied to out.
func (e *Engine) RunOnce(
	ctx context.Context, docker *sandbox.Docker, spec sandbox.Spec, out io.Writer,
) (sandbox.Result, error) {
	release, err := e.acquire(ctx)
	if err != nil {
		return sandbox.Result{}, err
	}

	defer release()

	res, err := docker.Run(ctx, spec, out)
	if err != nil {
		return sandbox.Result{}, fmt.Errorf("run: %w", err)
	}

	return res, nil
}
