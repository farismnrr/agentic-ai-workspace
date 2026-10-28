package command

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	cliagentdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/mcpbridge"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

const bridgeWait = 3600

// ProgramArgv builds one vendor-program turn using only catalogued model and effort choices.
func ProgramArgv(req taskapp.CLIRequest, models []cliagentdomain.Model, window int64) []string {
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
		argv = append(argv, "--tools", strings.Join(ClaudeTools(req.Tools), ","))
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

// ClaudeTools maps app permissions to the vendor program's tool names.
func ClaudeTools(allowed []string) []string {
	mapping := map[string][]string{
		codingdomain.ToolListFiles.Name:  {"LS", "Glob"},
		codingdomain.ToolFindFiles.Name:  {"Glob"},
		codingdomain.ToolReadFile.Name:   {"Read"},
		codingdomain.ToolSearchCode.Name: {"Grep"},
		codingdomain.ToolEditFile.Name:   {"Edit", "MultiEdit"},
		codingdomain.ToolWriteFile.Name:  {"Write"},
		codingdomain.ToolRunCommand.Name: {"Bash"},
		codingdomain.ToolRunTask.Name:    {"Bash"},
	}
	var names []string
	for _, tool := range allowed {
		for _, own := range mapping[tool] {
			if !slices.Contains(names, own) {
				names = append(names, own)
			}
		}
	}
	slices.Sort(names)
	return names
}

// MayChange reports whether the allowed tools can write or run commands.
func MayChange(allowed []string) bool {
	for _, tool := range []string{
		codingdomain.ToolEditFile.Name, codingdomain.ToolWriteFile.Name, codingdomain.ToolDeleteFile.Name,
		codingdomain.ToolMoveFile.Name, codingdomain.ToolRunCommand.Name, codingdomain.ToolRunTask.Name,
	} {
		if slices.Contains(allowed, tool) {
			return true
		}
	}
	return false
}

// WindowEnv passes Claude Code the selected context window; Codex takes it as an option.
func WindowEnv(program string, window int64) []string {
	if program != string(cliagentdomain.Claude) || window <= 0 {
		return nil
	}
	return []string{"CLAUDE_CODE_AUTO_COMPACT_WINDOW=" + strconv.FormatInt(window, 10)}
}

func codexArgv(req taskapp.CLIRequest, model, effort string, window int64) []string {
	argv := []string{"codex", "exec"}
	if req.Session != "" {
		argv = append(argv, "resume")
	}
	argv = append(argv, "--json", "--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox")
	argv = append(argv, choice("-m", model, "-c", `model_reasoning_effort="`+effort+`"`, effort)...)
	if window > 0 {
		argv = append(argv, "-c", "model_context_window="+strconv.FormatInt(window, 10))
	}
	if req.Bridge != nil {
		argv = append(argv, bridgeArgs(cliagentdomain.ChatGPT)...)
	}
	if req.Session != "" {
		argv = append(argv, req.Session)
	}
	return append(argv, "-")
}

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

func bridgeArgs(runtime cliagentdomain.Runtime) []string {
	socket := "UNIX-CONNECT:" + sandbox.BridgeDir + "/" + mcpbridge.SocketName
	if runtime == cliagentdomain.ChatGPT {
		return []string{
			"-c", `mcp_servers.team.command="socat"`, "-c",
			`mcp_servers.team.args=["STDIO","` + socket + `"]`, "-c",
			fmt.Sprintf("mcp_servers.team.tool_timeout_sec=%d", bridgeWait),
		}
	}
	config := `{"mcpServers":{"team":{"command":"socat","args":["STDIO","` + socket + `"]}}}`
	return []string{"--mcp-config", config, "--strict-mcp-config"}
}

const guardSettings = `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command",` +
	`"command":"node /usr/local/share/guard/guard-command.ts"}]}]}}`
