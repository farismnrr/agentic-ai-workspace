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
	Usage struct {
		Input int64 `json:"input_tokens"`
	} `json:"usage"`
	LastTokenUsage struct {
		Input int64 `json:"input_tokens"`
	} `json:"last_token_usage"`
	Info struct {
		LastTokenUsage struct {
			Input int64 `json:"input_tokens"`
		} `json:"last_token_usage"`
	} `json:"info"`
	TokenInfo struct {
		LastTokenUsage struct {
			Input int64 `json:"input_tokens"`
		} `json:"last_token_usage"`
	} `json:"token_info"`
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
	case "thread.token_count":
		if used := line.TokenInfo.LastTokenUsage.Input; used > 0 {
			c.out.used = used
			c.out.emit(application.CLIEvent{Kind: application.CLIUsage, Used: used})
		}
	case "turn.completed":
		// Codex turn usage is cumulative and is not the live context occupancy.
		// Only show an occupancy estimate when the stream supplies a last-call value.
		if used := max(line.LastTokenUsage.Input, line.Info.LastTokenUsage.Input); used > 0 {
			c.out.used = used
			c.out.emit(application.CLIEvent{Kind: application.CLIUsage, Used: used})
		}
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
	} else if status == "failed" || status == "declined" {
		code = 1
	}

	return application.CLIEvent{
		Kind: application.CLICommand, Technical: unwrapShell(command), ExitCode: code,
		Text: "Ran " + brief(unwrapShell(command), commandChars),
	}
}

// unwrapShell drops the `bash -lc '...'` that Codex puts around a command.
func unwrapShell(command string) string {
	if _, rest, ok := strings.Cut(command, " -lc "); ok {
		return unquote(strings.TrimSpace(rest))
	}

	return command
}

// unquote drops one pair of quotes around the whole text, and nothing inside it.
func unquote(text string) string {
	if len(text) >= 2 && (text[0] == '"' || text[0] == '\'') && text[len(text)-1] == text[0] {
		return text[1 : len(text)-1]
	}

	return text
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
