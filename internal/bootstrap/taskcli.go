package bootstrap

import (
	"context"
	"fmt"
	"slices"
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
	granted := codingrunner.GrantedFor(ctx, p.reach, req.ProjectID)
	docker, spec, release, err := p.prepareRun(ctx, req, granted)
	if err != nil {
		return taskapp.CLIResult{}, err
	}
	defer release()

	models := p.service.Models(ctx, req.UserID, cliagentdomain.Runtime(req.Program))
	window := p.chosenWindow(ctx, req)
	spec.Stdin, spec.Argv = strings.NewReader(req.Prompt), programArgv(req, models, window)
	spec.Env, err = p.environment(ctx, req, spec, window, granted)
	if err != nil {
		return taskapp.CLIResult{}, err
	}

	return p.execute(ctx, req, emit, docker, spec, window)
}

func (p *plans) execute(
	ctx context.Context, req taskapp.CLIRequest, emit func(taskapp.CLIEvent),
	docker *sandbox.Docker, spec sandbox.Spec, window int64,
) (taskapp.CLIResult, error) {
	stream := clistream.New(req.Program, emit)

	res, err := docker.Run(ctx, spec, stream)
	if err != nil {
		return taskapp.CLIResult{}, fmt.Errorf("run program: %w", err)
	}

	result := stream.Result()
	result.Window = shownWindow(window, result.Window)
	if res.ExitCode != 0 && result.Reason == "" {
		result.Reason = programNames[req.Program] + " stopped before it finished."
	}

	return result, nil
}

// programArgv is the command for one turn. The prompt arrives on standard input. The container is the
// sandbox, so the program is not asked to ask permission for every step. The model and effort are passed only
// when they are ones the program is known to take, so a stray value never becomes an option of its own.
func programArgv(req taskapp.CLIRequest, models []cliagentdomain.Model, window int64) []string {
	runtime := cliagentdomain.Runtime(req.Program)
	model, effort := cliagentdomain.Choice(models, req.Model, req.Effort)

	if runtime == cliagentdomain.ChatGPT {
		return codexArgv(req, model, effort, window)
	}

	argv := []string{
		"claude", "-p", "--output-format", "stream-json", "--verbose", "--include-partial-messages",
		"--permission-mode", "bypassPermissions",
	}
	argv = append(argv, choice("--model", model, "--effort", effort, effort)...)

	if req.Limited {
		argv = append(argv, "--tools", strings.Join(claudeTools(req.Tools), ","))
	}

	if req.Bridge != nil {
		argv = append(argv, bridgeArgs(runtime)...)
	}

	if req.Limited {
		argv = append(argv, "--settings", guardSettings)
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

// guardSettings makes the program ask guard-command before every shell command of a teammate, because its own shell
// does not pass through the server's checks. The command refuses one that starts a server.
const guardSettings = `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command",` +
	`"command":"node /usr/local/share/guard/guard-command.ts"}]}]}}`

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
