package runner

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

func (r *Runner) execute(ctx context.Context, task domain.Task, cancel context.CancelFunc) application.RunResult {
	out := &tail{limit: domain.MaxOutputBytes}

	docker, err := r.engine.Docker(ctx)
	if err != nil {
		return finish(ctx, err, sandbox.Result{}, out, 0)
	}

	spec := sandbox.Spec{Workspace: r.dir, Argv: task.Command,
		WorkspaceReadOnly: domain.ReadOnlyWorkspace(ctx)}
	// A one-shot MoLab container receives ONLY its explicit credential, never
	// project-wide variables or Git credentials from unrelated task commands.
	molab, scoped := ctx.Value(oneShotEnvKey{}).(molabRun)
	var projectValues map[string]string
	if !scoped {
		var err error
		projectValues, err = r.projectValues(ctx)
		if err != nil {
			return application.RunResult{ExitCode: -1, Output: "could not load project environment"}
		}
	}
	git, hasGit := Git{}, false
	if !scoped {
		git, hasGit = r.credentials(ctx)
	}

	hosts, granted := r.egressHosts(ctx, hasGit)
	if scoped {
		hosts, granted = []string{molab.host}, Granted{}
	} // No project database/gateway privileges in MoLab containers.
	_ = r.engine.prepare(ctx, r.dir, &spec, docker, hosts...) // without a socket the command just has no network
	projectValues, granting := granted.Apply(&spec, projectValues)
	baseEnv := append(environment(spec), granting...)

	if hasGit && spec.Egress != "" {
		baseEnv = append(baseEnv, gitEnvironment(git)...)
	}
	if scoped {
		// Explicit session-only environment goes into this single disposable container.
		baseEnv = append(baseEnv, "MARIMO_TOKEN="+molab.token)
	}
	spec.Env = MergeProjectEnvironment(baseEnv, projectValues)

	over := watchSize(ctx, r.dir, r.quota, cancel)

	started := time.Now()
	res, err := docker.Run(ctx, spec, out)
	result := finish(ctx, err, res, out, time.Since(started))

	switch {
	case over.Load():
		result.TimedOut, result.ExitCode = false, -1
		result.Output += stoppedBig
	case res.OOMKilled:
		result.ExitCode = -1
		result.Output += stoppedOOM
	}

	return result
}
