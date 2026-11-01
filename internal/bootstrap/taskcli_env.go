package bootstrap

import (
	"context"
	"errors"
	"fmt"

	cliagentcommand "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/adapter/command"
	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

var errProjectEnvironmentUnavailable = errors.New("project environment unavailable")

func (p *plans) prepareRun(ctx context.Context, req taskapp.CLIRequest, granted codingrunner.Granted, hasGit bool) (
	*sandbox.Docker, sandbox.Spec, func(), error,
) {
	dir, err := p.folders.Dir(req.ProjectID)
	if err != nil {
		return nil, sandbox.Spec{}, nil, fmt.Errorf("find project folder: %w", err)
	}

	home, err := p.box.home(req.UserID)
	if err != nil {
		return nil, sandbox.Spec{}, nil, fmt.Errorf("prepare program home: %w", err)
	}

	// Internal role-limited vendors use the server-side bridge for Git/gh,
	// builds and checks. Their own process receives neither project grants
	// nor GitHub authentication, even when the owner connected GitHub.
	if req.Limited {
		granted = codingrunner.Granted{}
		hasGit = false
	}

	// The program works for one task: the server's own record of it, never anything the program sends. With egress
	// grants on, its container gets that task's socket; a task that is not open gets no run at all.
	if req.TaskID != "" {
		ctx = codingrunner.WithTask(ctx, req.ProjectID, req.TaskID)
	}

	docker, spec, release, err := p.engine.Lease(ctx, dir,
		codingrunner.ProgramHosts(p.hosts[req.Program], granted, hasGit)...)
	if err != nil {
		return nil, sandbox.Spec{}, nil, fmt.Errorf("start program: %w", err)
	}
	granted.Open(&spec)
	// A teammate with check/build permissions never owns writable source,
	// even if the vendor maps the check tool to Bash.
	spec.WorkspaceReadOnly = req.Limited &&
		(req.RoleKey == "qa" || req.RoleKey == "security" || req.RoleKey == "devops" ||
			!cliagentcommand.MayChange(req.Tools))
	stop, err := p.prepareBridge(ctx, req, &spec)
	if err != nil {
		release()
		return nil, sandbox.Spec{}, nil, err
	}
	spec.Home = home
	return docker, spec, combineRelease(stop, release), nil
}

func combineRelease(stop, release func()) func() {
	return func() {
		if stop != nil {
			stop()
		}
		release()
	}
}

func (p *plans) prepareBridge(
	ctx context.Context, req taskapp.CLIRequest, spec *sandbox.Spec,
) (func(), error) {
	if req.Bridge == nil {
		return nil, nil
	}
	stop, bridgeDir, err := p.serveBridge(ctx, req.Bridge)
	if err != nil {
		return nil, fmt.Errorf("start tool bridge: %w", err)
	}
	spec.Bridge = bridgeDir
	return stop, nil
}

func (p *plans) environment(
	ctx context.Context, req taskapp.CLIRequest, spec sandbox.Spec, window int64, granted codingrunner.Granted,
	git *codingrunner.Git,
) ([]string, error) {
	var values map[string]string
	if p.env != nil && !req.Limited {
		var err error
		if values, err = p.env.Values(ctx, req.ProjectID); err != nil {
			return nil, errProjectEnvironmentUnavailable
		}
	}
	if req.Limited {
		// Model-managed vendor processes have broad built-in code execution.
		// They do not get project secrets, GitHub tokens or egress grants;
		// approved VCS and checks instead flow through the server bridge.
		granted = codingrunner.Granted{}
	}
	values, granting := granted.Apply(&spec, values)
	entries := append(cliagentcommand.WindowEnv(req.Program, window), "HOME="+sandbox.Home,
		"CODEX_HOME="+sandbox.Home+"/.codex", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		"DISABLE_AUTOUPDATER=1", fmt.Sprintf("MCP_TOOL_TIMEOUT=%d", bridgeWaitMillis))
	entries = append(append(entries, granting...), guardEnv(req)...)
	if git != nil && spec.Egress != "" && !req.Limited {
		entries = append(entries, codingrunner.GitEnvironment(*git)...)
	}
	spec.Env = codingrunner.Environment(spec, entries...)
	return codingrunner.MergeProjectEnvironment(spec.Env, values), nil
}
