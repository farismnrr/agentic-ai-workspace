package openai

import (
	"encoding/base64"
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

	var out, pictures []message

	var texts []string

	for _, item := range msg.Blocks {
		switch item.Kind {
		case domain.BlockToolResult:
			content := item.Text
			out = append(out, message{Role: "tool", ToolCallID: item.ID, Content: &content})

			if len(item.Data) > 0 {
				pictures = append(pictures, picture(item))
			}
		case domain.BlockText:
			texts = append(texts, item.Text)
		case domain.BlockToolUse:
		}
	}

	out = append(out, pictures...)

	if len(texts) > 0 {
		joined := strings.Join(texts, "\n\n")
		out = append(out, message{Role: "user", Content: &joined})
	}

	return out
}

// part is one piece of a message that holds a picture.
type part struct {
	ImageURL *imageURL `json:"image_url,omitempty"`
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

// picture carries a screenshot to the model. Chat Completions takes pictures in user messages only, not in the
// "tool" message that answers the call, so it follows the tool messages.
func picture(item domain.Block) message {
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(item.Data)

	return message{Role: "user", Content: []part{
		{Type: "text", Text: "The screenshot that the tool call " + item.ID + " returned:"},
		{Type: "image_url", ImageURL: &imageURL{URL: url}},
	}}
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
