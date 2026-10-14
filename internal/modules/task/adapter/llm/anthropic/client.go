package anthropic

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/transport"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	version   = "2023-06-01"
	maxTokens = 8192
)

// Client talks to the Anthropic Messages API (or a gateway that speaks it).
type Client struct {
	http    *transport.Client
	baseURL string
	apiKey  string
}

// New builds a client for a base address such as https://api.anthropic.com.
func New(baseURL, apiKey string) *Client {
	return &Client{http: transport.New(), baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
}

type block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type message struct {
	Role    string  `json:"role"`
	Content []block `json:"content"`
}

type tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type request struct {
	Model     string    `json:"model"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
	Tools     []tool    `json:"tools,omitempty"`
	MaxTokens int       `json:"max_tokens"`
	Stream    bool      `json:"stream,omitempty"`
}

type response struct {
	Content []block `json:"content"`
	Usage   struct {
		Input  int64 `json:"input_tokens"`
		Output int64 `json:"output_tokens"`
	} `json:"usage"`
}

// Complete sends one conversation turn and returns the model's text and tool calls.
func (c *Client) Complete(ctx context.Context, in application.Request) (application.Reply, error) {
	if in.OnText != nil {
		return c.stream(ctx, in)
	}

	var out response

	headers := map[string]string{"x-api-key": c.apiKey, "anthropic-version": version}
	if err := c.http.Post(ctx, c.baseURL+"/v1/messages", headers, encode(in), &out); err != nil {
		return application.Reply{}, err //nolint:wrapcheck // transport already returns domain errors
	}

	reply := application.Reply{Usage: domain.Usage{Input: out.Usage.Input, Output: out.Usage.Output}}

	for _, item := range out.Content {
		switch item.Type {
		case "text":
			reply.Blocks = append(reply.Blocks, domain.Block{Kind: domain.BlockText, Text: item.Text})
		case "tool_use":
			reply.Blocks = append(reply.Blocks, domain.Block{
				Kind: domain.BlockToolUse, ID: item.ID, Name: item.Name, Input: objectOrEmpty(item.Input),
			})
		}
	}

	return reply, nil
}

func encode(in application.Request) request {
	out := request{Model: in.Model, System: in.System, MaxTokens: maxTokens}

	for _, def := range in.Tools {
		out.Tools = append(out.Tools, tool{
			Name: def.Name, Description: def.Description, InputSchema: schemaOrEmpty(def.Schema),
		})
	}

	for _, msg := range in.Messages {
		blocks := encodeBlocks(msg.Blocks)
		if len(blocks) > 0 {
			out.Messages = append(out.Messages, message{Role: string(msg.Role), Content: blocks})
		}
	}

	return out
}

func encodeBlocks(blocks []domain.Block) []block {
	out := make([]block, 0, len(blocks))

	for _, item := range blocks {
		switch item.Kind {
		case domain.BlockText:
			if strings.TrimSpace(item.Text) != "" {
				out = append(out, block{Type: "text", Text: item.Text})
			}
		case domain.BlockToolUse:
			out = append(out, block{Type: "tool_use", ID: item.ID, Name: item.Name, Input: objectOrEmpty(item.Input)})
		case domain.BlockToolResult:
			out = append(out, block{Type: "tool_result", ToolUseID: item.ID, Content: item.Text, IsError: item.IsError})
		}
	}

	return out
}

func objectOrEmpty(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || raw[0] != '{' {
		return json.RawMessage(`{}`)
	}

	return raw
}

func schemaOrEmpty(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}

	return raw
}
