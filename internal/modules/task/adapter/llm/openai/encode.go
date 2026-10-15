package openai

import (
	"encoding/json"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// encode turns a turn of the conversation into a Chat Completions request.
func encode(in application.Request) request {
	out := request{Model: in.Model, ReasoningEffort: string(in.Effort)}

	if in.System != "" {
		out.Messages = append(out.Messages, message{Role: "system", Content: &in.System})
	}

	for _, def := range in.Tools {
		schema := def.Schema
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}

		out.Tools = append(out.Tools, toolDef{
			Type: "function", Function: function{Name: def.Name, Description: def.Description, Parameters: schema},
		})
	}

	for _, msg := range in.Messages {
		out.Messages = append(out.Messages, encodeMessage(msg)...)
	}

	return out
}

// encodeMessage maps one transcript turn to chat messages: tool results become
// "tool" messages, which must directly follow the assistant's calls.
func encodeMessage(msg domain.Message) []message {
	if msg.Role == domain.RoleAssistant {
		return []message{encodeAssistant(msg.Blocks)}
	}

	var out []message

	var texts []string

	for _, item := range msg.Blocks {
		switch item.Kind {
		case domain.BlockToolResult:
			content := item.Text
			out = append(out, message{Role: "tool", ToolCallID: item.ID, Content: &content})
		case domain.BlockText:
			texts = append(texts, item.Text)
		case domain.BlockToolUse:
		}
	}

	if len(texts) > 0 {
		joined := strings.Join(texts, "\n\n")
		out = append(out, message{Role: "user", Content: &joined})
	}

	return out
}

func encodeAssistant(blocks []domain.Block) message {
	out := message{Role: "assistant"}

	var texts []string

	for _, item := range blocks {
		switch item.Kind {
		case domain.BlockText:
			texts = append(texts, item.Text)
		case domain.BlockToolUse:
			out.ToolCalls = append(out.ToolCalls, toolCall{
				ID: item.ID, Type: "function", Function: function{Name: item.Name, Arguments: string(arguments(item.Input))},
			})
		case domain.BlockToolResult:
		}
	}

	if len(texts) > 0 {
		joined := strings.Join(texts, "\n\n")
		out.Content = &joined
	}

	return out
}

func arguments(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}

	return raw
}
