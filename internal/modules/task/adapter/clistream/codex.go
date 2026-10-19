package clistream

import (
	"encoding/json"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

// codexLine is one line of `codex exec --json`.
type codexLine struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
	Message  string `json:"message"`
	Error    struct {
		Message string `json:"message"`
	} `json:"error"`
	Item struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Command  string `json:"command"`
		Status   string `json:"status"`
		ExitCode *int   `json:"exit_code"`
		Changes  []struct {
			Path string `json:"path"`
		} `json:"changes"`
	} `json:"item"`
}

type codex struct{ out *Stream }

func newCodex(out *Stream) *codex { return &codex{out: out} }

func (c *codex) line(raw []byte) {
	var line codexLine
	if json.Unmarshal(raw, &line) != nil {
		return
	}

	switch line.Type {
	case "thread.started":
		c.out.session = line.ThreadID
	case "item.completed":
		c.item(line)
	case "turn.failed":
		c.out.reason = "ChatGPT stopped: " + brief(line.Error.Message, errorChars)
	}
}

func (c *codex) item(line codexLine) {
	item := line.Item

	switch item.Type {
	case "agent_message", "assistant_message":
		c.out.summary = item.Text
		c.out.emit(application.CLIEvent{Kind: application.CLIText, Text: item.Text})
	case "command_execution":
		c.out.emit(codexCommand(item.Command, item.ExitCode, item.Status))
	case "file_change":
		c.out.emit(codexChange(line))
	}
}

func codexCommand(command string, exit *int, status string) application.CLIEvent {
	code := 0
	if exit != nil {
		code = *exit
	} else if status == "failed" {
		code = 1
	}

	return application.CLIEvent{
		Kind: application.CLICommand, Technical: command, ExitCode: code,
		Text: "Ran " + brief(unwrapShell(command), commandChars),
	}
}

// unwrapShell drops the `bash -lc '...'` that Codex puts around a command.
func unwrapShell(command string) string {
	if _, rest, ok := strings.Cut(command, " -lc "); ok {
		return strings.Trim(rest, `'"`)
	}

	return command
}

func codexChange(line codexLine) application.CLIEvent {
	files := make([]string, 0, len(line.Item.Changes))
	for _, change := range line.Item.Changes {
		files = append(files, relative(change.Path))
	}

	return application.CLIEvent{
		Kind: application.CLIChange, Files: files, Technical: strings.Join(files, ", "),
		Text: "Changed " + strings.Join(files, ", "),
	}
}
