package clistream

import (
	"encoding/json"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

// claudeLine is one line of `claude -p --output-format stream-json`.
type claudeLine struct {
	ModelUsage map[string]struct {
		Window int64 `json:"contextWindow"`
	} `json:"modelUsage"`
	Event struct {
		Type         string `json:"type"`
		ContentBlock struct {
			Type string `json:"type"`
		} `json:"content_block"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	} `json:"event"`
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
	Message   struct {
		Model   string          `json:"model"`
		Content json.RawMessage `json:"content"`
		Usage   claudeUsage     `json:"usage"`
	} `json:"message"`
	IsError bool `json:"is_error"`
}

// claudeUsage is what one call to the model carried: the new input, what was read from its cache, and the answer.
type claudeUsage struct {
	Input, Output, CacheRead, CacheWrite int64
}

// UnmarshalJSON reads the program's field names.
func (u *claudeUsage) UnmarshalJSON(raw []byte) error {
	var in struct {
		Input      int64 `json:"input_tokens"`
		Output     int64 `json:"output_tokens"`
		CacheRead  int64 `json:"cache_read_input_tokens"`
		CacheWrite int64 `json:"cache_creation_input_tokens"`
	}

	if err := json.Unmarshal(raw, &in); err != nil {
		return nil //nolint:nilerr // a usage that cannot be read only leaves the meter empty
	}

	*u = claudeUsage{Input: in.Input, Output: in.Output, CacheRead: in.CacheRead, CacheWrite: in.CacheWrite}

	return nil
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
	out       *Stream
	calls     map[string]claudeBlock
	lastModel string
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
		if line.Event.Type == "content_block_start" && line.Event.ContentBlock.Type == "tool_use" {
			c.out.emit(application.CLIEvent{Kind: application.CLIActivity})
		}
		if line.Event.Delta.Type == "text_delta" && line.Event.Delta.Text != "" {
			c.out.emit(application.CLIEvent{Kind: application.CLIDelta, Text: line.Event.Delta.Text})
		}
	case "assistant", "user":
		if u := line.Message.Usage; line.Type == "assistant" && u.Input+u.CacheRead+u.CacheWrite > 0 {
			if line.Message.Model != "" {
				c.lastModel = line.Message.Model
			}
			c.out.used = u.Input + u.CacheRead + u.CacheWrite + u.Output
			c.out.emit(application.CLIEvent{Kind: application.CLIUsage, Used: c.out.used, Window: c.out.window})
		}

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
			c.out.emit(application.CLIEvent{Kind: application.CLIActivity})
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

	// A result can list multiple models. Never use another model's larger
	// window to make the last model's context appear less full.
	if usage, ok := line.ModelUsage[c.lastModel]; ok {
		c.out.window = usage.Window
	} else if len(line.ModelUsage) == 1 {
		for _, usage := range line.ModelUsage {
			c.out.window = usage.Window
		}
	}

	if c.out.used > 0 && c.out.window > 0 {
		c.out.emit(application.CLIEvent{Kind: application.CLIUsage, Used: c.out.used, Window: c.out.window})
	}
	if line.IsError {
		c.out.reason = "Claude stopped: " + brief(line.Result, errorChars)
	}
}
