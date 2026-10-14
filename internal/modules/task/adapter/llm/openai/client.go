package openai

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/transport"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Client talks to the OpenAI Chat Completions API or any gateway that speaks it, such as 9router.
type Client struct {
	http    *transport.Client
	baseURL string
	apiKey  string
}

// New builds a client for a base address such as https://api.openai.com/v1.
func New(baseURL, apiKey string) *Client {
	return &Client{http: transport.New(), baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}

type function struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Arguments   string          `json:"arguments,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type toolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function function `json:"function"`
}

type message struct {
	Content    *string    `json:"content"`
	Role       string     `json:"role"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
}

type toolDef struct {
	Type     string   `json:"type"`
	Function function `json:"function"`
}

type request struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Tools    []toolDef `json:"tools,omitempty"`
}

type response struct {
	Choices []struct {
		Message struct {
			Content   *string    `json:"content"`
			ToolCalls []toolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		Prompt     int64 `json:"prompt_tokens"`
		Completion int64 `json:"completion_tokens"`
	} `json:"usage"`
}

// Complete sends one conversation turn and returns the model's text and tool calls.
func (c *Client) Complete(ctx context.Context, in application.Request) (application.Reply, error) {
	var out response

	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}
	if err := c.http.Post(ctx, c.baseURL+"/chat/completions", headers, encode(in), &out); err != nil {
		return application.Reply{}, err //nolint:wrapcheck // transport already returns domain errors
	}

	reply := application.Reply{Usage: domain.Usage{Input: out.Usage.Prompt, Output: out.Usage.Completion}}
	if len(out.Choices) == 0 {
		return reply, nil
	}

	choice := out.Choices[0].Message
	if choice.Content != nil && strings.TrimSpace(*choice.Content) != "" {
		reply.Blocks = append(reply.Blocks, domain.Block{Kind: domain.BlockText, Text: *choice.Content})
	}

	for _, call := range choice.ToolCalls {
		reply.Blocks = append(reply.Blocks, domain.Block{
			Kind: domain.BlockToolUse, ID: call.ID, Name: call.Function.Name, Input: parseArguments(call.Function.Arguments),
		})
	}

	return reply, nil
}

// parseArguments keeps the model's JSON, or wraps unreadable text so the tool can report it.
func parseArguments(raw string) json.RawMessage {
	if strings.TrimSpace(raw) == "" {
		return json.RawMessage(`{}`)
	}

	var object map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &object) == nil && object != nil {
		return json.RawMessage(raw)
	}

	wrapped, _ := json.Marshal(map[string]string{"_unreadable_arguments": raw})

	return wrapped
}

func encode(in application.Request) request {
	out := request{Model: in.Model}

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
