package clistream

import (
	"encoding/json"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

// claudeLine is one line of `claude -p --output-format stream-json`.
type claudeLine struct {
	Event struct {
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	} `json:"event"`
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	Message   struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
	IsError bool `json:"is_error"`
}

type claudeBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	ToolUseID string          `json:"tool_use_id"`
	Input     json.RawMessage `json:"input"`
	IsError   bool            `json:"is_error"`
}

type claudeInput struct {
	FilePath    string `json:"file_path"`
	NotebookRef string `json:"notebook_path"`
	Command     string `json:"command"`
	Description string `json:"description"`
}

type claude struct {
	out   *Stream
	calls map[string]claudeBlock
}

func newClaude(out *Stream) *claude {
	return &claude{out: out, calls: map[string]claudeBlock{}}
}

func (c *claude) line(raw []byte) {
	var line claudeLine
	if json.Unmarshal(raw, &line) != nil {
		return
	}

	switch line.Type {
	case "stream_event":
		if line.Event.Delta.Type == "text_delta" && line.Event.Delta.Text != "" {
			c.out.emit(application.CLIEvent{Kind: application.CLIDelta, Text: line.Event.Delta.Text})
		}
	case "assistant", "user":
		c.blocks(line.Message.Content)
	case "result":
		c.finish(line)
	}
}

func (c *claude) blocks(content json.RawMessage) {
	var blocks []claudeBlock
	if json.Unmarshal(content, &blocks) != nil {
		return // user messages may carry plain text instead of blocks
	}

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				c.out.emit(application.CLIEvent{Kind: application.CLIText, Text: block.Text})
			}
		case "tool_use":
			c.calls[block.ID] = block
		case "tool_result":
			c.report(block)
		}
	}
}

// report describes a finished tool call. Only steps that change something are shown, as with the agent's own tools.
func (c *claude) report(result claudeBlock) {
	call, ok := c.calls[result.ToolUseID]
	if !ok {
		return
	}

	delete(c.calls, result.ToolUseID)

	var in claudeInput
	_ = json.Unmarshal(call.Input, &in)

	switch call.Name {
	case "Write", "Edit", "MultiEdit", "NotebookEdit":
		c.out.emit(changeStep(call.Name, in, result.IsError))
	case "Bash":
		c.out.emit(commandStep(in, result.IsError))
	}
}

func changeStep(tool string, in claudeInput, failed bool) application.CLIEvent {
	path := relative(in.FilePath + in.NotebookRef)
	event := application.CLIEvent{Kind: application.CLIChange, Technical: tool + " " + path}

	if failed {
		event.Text = "Tried to change " + path + ", but it did not work"

		return event
	}

	event.Files = []string{path}
	event.Text = "Changed " + path

	if tool == "Write" {
		event.Text = "Wrote " + path
	}

	return event
}

func commandStep(in claudeInput, failed bool) application.CLIEvent {
	event := application.CLIEvent{Kind: application.CLICommand, Technical: in.Command, Text: in.Description}
	if event.Text == "" {
		event.Text = "Ran " + brief(in.Command, commandChars)
	}

	if failed {
		event.ExitCode = 1
	}

	return event
}

func (c *claude) finish(line claudeLine) {
	c.out.session = line.SessionID
	c.out.summary = line.Result

	if line.IsError {
		c.out.reason = "Claude stopped: " + brief(line.Result, errorChars)
	}
}
