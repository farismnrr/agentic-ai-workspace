package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

func (r *Runner) execute(ctx context.Context, task domain.Task, cancel context.CancelFunc) application.RunResult {
	out := &tail{limit: domain.MaxOutputBytes}

	spec := sandbox.Spec{Workspace: r.dir, Argv: task.Command,
		WorkspaceReadOnly: domain.ReadOnlyWorkspace(ctx)}
	if task.Kind == "build" || task.Kind == "artifact" {
		// The source stays read-only; each build or document/media run gets
		// a fresh isolated output folder for review and download.
		prefix := ".build-output-"
		if task.Kind == "artifact" {
			prefix = ".artifact-output-"
		}
		path, err := os.MkdirTemp(r.dir, prefix)
		if err != nil {
			return application.RunResult{ExitCode: -1, Output: "could not create build output: " + err.Error()}
		}
		spec.Artifacts = path
	}
	// A one-shot MoLab container receives ONLY its explicit credential, never
	// project-wide variables or Git credentials from unrelated task commands.
	molab, scoped := ctx.Value(oneShotEnvKey{}).(molabRun)
	// Checks and builds must not inherit credentials, project secrets or
	// privileged network routes even if their commands try to obtain them.
	// Read-only QA checks never receive GitHub credentials. A server-
	// authorized gh operation is different: GitOnly gives it GitHub
	// authentication while retaining a read-only project mount.
	gitOnly := domain.GitOnly(ctx)
	locked := domain.ReadOnlyWorkspace(ctx) && !gitOnly
	var projectValues map[string]string
	if !scoped && !locked && !gitOnly {
		var err error
		projectValues, err = r.projectValues(ctx)
		if err != nil {
			return application.RunResult{ExitCode: -1, Output: "could not load project environment"}
		}
	}
	git, hasGit := Git{}, false
	if !scoped && !locked {
		git, hasGit = r.credentials(ctx)
	}
	if gitOnly && task.Kind == "git" && !hasGit && requiresGitHubConnection(task.Command) {
		return application.RunResult{
			ExitCode: -1,
			Output:   "CONNECT_GITHUB_REQUIRED: The project owner has not connected GitHub. Connect GitHub in Settings before using this remote Git operation. No network request was made.",
		}
	}

	hosts, granted := r.egressHosts(ctx, hasGit)
	if gitOnly {
		// Git and gh need GitHub, not project databases or gateway grants.
		granted = Granted{}
		hosts = nil
		if hasGit {
			hosts = githubHosts
		}
	}
	if scoped {
		hosts, granted = []string{molab.host}, Granted{}
	} else if locked {
		hosts, granted = nil, Granted{}
	}
	docker, err := r.engine.Docker(ctx)
	if err != nil {
		return finish(ctx, err, sandbox.Result{}, out, 0)
	}
	// No project database/gateway privileges in isolated check/build containers.
	_ = r.engine.prepare(ctx, r.dir, &spec, docker, hosts...) // without a socket the command just has no network
	projectValues, granting := granted.Apply(&spec, projectValues)
	baseEnv := append(environment(spec), granting...)
	if spec.Artifacts != "" {
		baseEnv = append(baseEnv, "BUILD_OUT="+sandbox.BuildOutput, "ARTIFACT_OUT="+sandbox.BuildOutput)
	}

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
	if spec.Artifacts != "" {
		result.Output += fmt.Sprintf("\nGenerated outputs: %s/ (BUILD_OUT/ARTIFACT_OUT=/output)", filepath.Base(spec.Artifacts))
	}

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

func requiresGitHubConnection(argv []string) bool {
	for index, arg := range argv {
		if arg != "--no-pager" || index+1 >= len(argv) {
			continue
		}
		switch argv[index+1] {
		case "clone", "fetch", "fetch-pack", "ls-remote", "pull", "push":
			return true
		}
	}
	return false
}
