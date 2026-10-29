package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"strings"

	cliagentcommand "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/adapter/command"
	cliagentdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
	codingrunner "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/runner"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/clistream"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

var programNames = map[string]string{"claude": "Claude", "chatgpt": "ChatGPT"}

// Ready says whether the person is signed in to the program.
func (p *plans) Ready(ctx context.Context, userID, program string) (bool, error) {
	runtime, err := cliagentdomain.Parse(program)
	if err != nil {
		return false, fmt.Errorf("find program: %w", err)
	}

	state, err := p.service.Status(ctx, userID, runtime)
	if err != nil {
		return false, fmt.Errorf("check sign-in: %w", err)
	}

	return state.Connected, nil
}

// Run does one turn of a task with the vendor's program, in the project's folder and the person's own home.
// The program works inside the container; what it prints is read, never what it keeps in the home.
func (p *plans) Run(
	ctx context.Context, req taskapp.CLIRequest, emit func(taskapp.CLIEvent),
) (taskapp.CLIResult, error) {
	granted := codingrunner.GrantedFor(ctx, p.reach, req.ProjectID)
	git, hasGit := p.gitOf(ctx, req.ProjectID)
	docker, spec, release, err := p.prepareRun(ctx, req, granted, hasGit)
	if err != nil {
		return taskapp.CLIResult{}, err
	}
	defer release()

	models := p.service.Models(ctx, req.UserID, cliagentdomain.Runtime(req.Program))
	window := p.chosenWindow(ctx, req)
	spec.Stdin, spec.Argv = strings.NewReader(req.Prompt), cliagentcommand.ProgramArgv(req, models, window)
	spec.Env, err = p.environment(ctx, req, spec, window, granted, git)
	if err != nil {
		return taskapp.CLIResult{}, err
	}

	// The chosen override controls vendor flags; the catalog window is only
	// used for display when no override was set.
	measuredWindow := shownWindow(window, catalogWindow(models, req.Model))
	return p.execute(ctx, req, emit, docker, spec, window, measuredWindow)
}

func (p *plans) execute(
	ctx context.Context, req taskapp.CLIRequest, emit func(taskapp.CLIEvent),
	docker *sandbox.Docker, spec sandbox.Spec, window, measuredWindow int64,
) (taskapp.CLIResult, error) {
	stream := clistream.New(req.Program, func(event taskapp.CLIEvent) {
		if event.Kind == taskapp.CLIUsage {
			event.Window = shownWindow(measuredWindow, event.Window)
		}
		emit(event)
	})

	res, err := docker.Run(ctx, spec, stream)
	if err != nil {
		return taskapp.CLIResult{}, fmt.Errorf("run program: %w", err)
	}

	result := stream.Result()
	result.Window = shownWindow(measuredWindow, result.Window)
	if res.ExitCode != 0 && result.Reason == "" {
		result.Reason = programNames[req.Program] + " stopped before it finished."
	}

	return result, nil
}

// guardEnv is what guard-command needs to apply the same rule as the app's own loop: the pattern of a command that
// starts a server and the words that say why not, which depend on whether this teammate may start the app.
func guardEnv(req taskapp.CLIRequest) []string {
	if !req.Limited {
		return nil
	}

	canStart := slices.Contains(req.Tools, "start_site")

	return []string{
		"GUARD_SERVER_PATTERN=" + taskapp.ServerStartPattern,
		"GUARD_SERVER_MESSAGE=" + taskapp.ServerHoldText(canStart),
	}
}
