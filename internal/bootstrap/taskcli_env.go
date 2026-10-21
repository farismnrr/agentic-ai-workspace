package bootstrap

import (
	"context"
	"errors"
	"fmt"

	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

var errProjectEnvironmentUnavailable = errors.New("project environment unavailable")

func (p *plans) prepareRun(ctx context.Context, req taskapp.CLIRequest) (
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

	docker, spec, release, err := p.engine.Lease(ctx, dir, p.hosts[req.Program]...)
	if err != nil {
		return nil, sandbox.Spec{}, nil, fmt.Errorf("start program: %w", err)
	}
	spec.WorkspaceReadOnly = req.Limited && !mayChange(req.Tools)
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
	ctx context.Context, req taskapp.CLIRequest, spec sandbox.Spec, window int64,
) ([]string, error) {
	entries := append(windowEnv(req.Program, window), "HOME="+sandbox.Home,
		"CODEX_HOME="+sandbox.Home+"/.codex", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		"DISABLE_AUTOUPDATER=1", fmt.Sprintf("MCP_TOOL_TIMEOUT=%d", bridgeWaitMillis))
	spec.Env = codingrunner.Environment(spec, append(entries, guardEnv(req)...)...)
	if p.env == nil {
		return spec.Env, nil
	}
	values, err := p.env.Values(ctx, req.ProjectID)
	if err != nil {
		return nil, errProjectEnvironmentUnavailable
	}
	return codingrunner.MergeProjectEnvironment(spec.Env, values), nil
}
