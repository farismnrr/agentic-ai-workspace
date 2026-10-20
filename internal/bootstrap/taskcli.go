package bootstrap

import (
	"context"
	"fmt"
	"strings"

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
	dir, err := p.folders.Dir(req.ProjectID)
	if err != nil {
		return taskapp.CLIResult{}, fmt.Errorf("find project folder: %w", err)
	}

	home, err := p.box.home(req.UserID)
	if err != nil {
		return taskapp.CLIResult{}, err
	}

	docker, spec, release, err := p.engine.Lease(ctx, dir, p.hosts[req.Program]...)
	if err != nil {
		return taskapp.CLIResult{}, fmt.Errorf("start program: %w", err)
	}

	defer release()

	models := p.service.Models(ctx, req.UserID, cliagentdomain.Runtime(req.Program))
	spec.WorkspaceReadOnly = req.Limited && !mayChange(req.Tools)
	spec.Home, spec.Stdin, spec.Argv = home, strings.NewReader(req.Prompt), programArgv(req, models)
	spec.Env = codingrunner.Environment(spec, "HOME="+sandbox.Home, "CODEX_HOME="+sandbox.Home+"/.codex",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1", "DISABLE_AUTOUPDATER=1")

	stream := clistream.New(req.Program, emit)

	res, err := docker.Run(ctx, spec, stream)
	if err != nil {
		return taskapp.CLIResult{}, fmt.Errorf("run program: %w", err)
	}

	result := stream.Result()
	if res.ExitCode != 0 && result.Reason == "" {
		result.Reason = programNames[req.Program] + " stopped before it finished."
	}

	return result, nil
}

// programArgv is the command for one turn. The prompt arrives on standard input. The container is the
// sandbox, so the program is not asked to ask permission for every step. The model and effort are passed only
// when they are ones the program is known to take, so a stray value never becomes an option of its own.
func programArgv(req taskapp.CLIRequest, models []cliagentdomain.Model) []string {
	runtime := cliagentdomain.Runtime(req.Program)
	model, effort := cliagentdomain.Choice(models, req.Model, req.Effort)

	if runtime == cliagentdomain.ChatGPT {
		argv := []string{"codex", "exec"}
		if req.Session != "" {
			argv = append(argv, "resume")
		}

		argv = append(argv, "--json", "--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox")
		argv = append(argv, choice("-m", model, "-c", `model_reasoning_effort="`+effort+`"`, effort)...)

		if req.Session != "" {
			argv = append(argv, req.Session)
		}

		return append(argv, "-")
	}

	argv := []string{
		"claude", "-p", "--output-format", "stream-json", "--verbose", "--include-partial-messages",
		"--permission-mode", "bypassPermissions",
	}
	argv = append(argv, choice("--model", model, "--effort", effort, effort)...)

	if req.Limited {
		argv = append(argv, "--tools", strings.Join(claudeTools(req.Tools), ","))
	}

	if req.Session != "" {
		argv = append(argv, "--resume", req.Session)
	}

	return argv
}

// choice is the options for a model and an effort the person chose; each is left out when empty.
func choice(modelFlag, model, effortFlag, effortOption, effort string) []string {
	var options []string

	if model != "" {
		options = append(options, modelFlag, model)
	}

	if effort != "" {
		options = append(options, effortFlag, effortOption)
	}

	return options
}
