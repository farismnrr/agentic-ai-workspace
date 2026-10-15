package openai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/llm/transport"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

// Client talks to the OpenAI Chat Completions API or any gateway that speaks it, such as 9router.
type Client struct {
	http    *transport.Client
	baseURL string
	apiKey  string
}

// New builds a client for a base address such as https://api.openai.com/v1.
func New(baseURL, apiKey string, policy egress.Policy) *Client {
	return &Client{http: transport.New(policy), baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey}
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

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type request struct {
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
	Model         string         `json:"model"`
	// ReasoningEffort is the OpenAI-style effort; gateways such as 9router pass it on.
	ReasoningEffort string    `json:"reasoning_effort,omitempty"`
	Messages        []message `json:"messages"`
	Tools           []toolDef `json:"tools,omitempty"`
	Stream          bool      `json:"stream,omitempty"`
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
	if in.OnText != nil {
		return c.stream(ctx, in, true)
	}

	var out response

	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}
	err := c.http.Post(ctx, c.baseURL+"/chat/completions", headers, encode(in), &out)
	if errors.Is(err, transport.ErrEventStream) {
		return c.stream(ctx, in, false)
	}

	if err != nil {
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
			Kind: domain.BlockToolUse, ID: callID(call.ID), Name: call.Function.Name,
			Input: parseArguments(call.Function.Arguments),
		})
	}

	return reply, nil
}
