package anthropic

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type streamEvent struct {
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		PartialJSON string `json:"partial_json"`
	} `json:"delta"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
	ContentBlock block `json:"content_block"`
	Message      struct {
		Usage struct {
			Input int64 `json:"input_tokens"`
		} `json:"usage"`
	} `json:"message"`
	Usage struct {
		Output int64 `json:"output_tokens"`
	} `json:"usage"`
	Index int `json:"index"`
}

// assembly collects the pieces of a streamed answer into whole blocks.
type assembly struct {
	text    map[int]*strings.Builder
	args    map[int]*strings.Builder
	calls   map[int]block
	onText  func(string)
	indexes map[int]struct{}
	usage   domain.Usage
}

func newAssembly(onText func(string)) *assembly {
	return &assembly{
		text: map[int]*strings.Builder{}, args: map[int]*strings.Builder{}, calls: map[int]block{},
		indexes: map[int]struct{}{}, onText: onText,
	}
}

// complete streams one turn and returns the whole answer.
func (c *Client) stream(ctx context.Context, in application.Request) (application.Reply, error) {
	body := encode(in)
	body.Stream = true
	assembled := newAssembly(in.OnText)
	headers := map[string]string{"x-api-key": c.apiKey, "anthropic-version": version}

	err := c.http.Stream(ctx, c.baseURL+"/v1/messages", headers, body, assembled.handle)
	if err != nil {
		return application.Reply{}, err //nolint:wrapcheck // transport already returns domain errors
	}

	return application.Reply{Blocks: assembled.blocks(), Usage: assembled.usage}, nil
}

func (a *assembly) handle(event, data string) error {
	var parsed streamEvent
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil //nolint:nilerr // pings and unknown events are skipped
	}

	switch event {
	case "message_start":
		a.usage.Input = parsed.Message.Usage.Input
	case "content_block_start":
		a.indexes[parsed.Index] = struct{}{}
		a.calls[parsed.Index] = parsed.ContentBlock
	case "content_block_delta":
		a.delta(parsed)
	case "message_delta":
		a.usage.Output = parsed.Usage.Output
	case "error":
		return &domain.ModelError{
			Message: parsed.Error.Message,
			Retryable: parsed.Error.Type == "overloaded_error" || parsed.Error.Type == "rate_limit_error" ||
				parsed.Error.Type == "api_error",
		}
	}

	return nil
}

func (a *assembly) delta(parsed streamEvent) {
	switch parsed.Delta.Type {
	case "text_delta":
		if a.text[parsed.Index] == nil {
			a.text[parsed.Index] = &strings.Builder{}
		}

		_, _ = a.text[parsed.Index].WriteString(parsed.Delta.Text)

		if a.onText != nil {
			a.onText(parsed.Delta.Text)
		}
	case "input_json_delta":
		if a.args[parsed.Index] == nil {
			a.args[parsed.Index] = &strings.Builder{}
		}

		_, _ = a.args[parsed.Index].WriteString(parsed.Delta.PartialJSON)
	}
}

func (a *assembly) blocks() []domain.Block {
	order := make([]int, 0, len(a.indexes))
	for index := range a.indexes {
		order = append(order, index)
	}

	sort.Ints(order)

	var out []domain.Block

	for _, index := range order {
		start := a.calls[index]

		switch start.Type {
		case "text":
			if text := a.text[index]; text != nil && strings.TrimSpace(text.String()) != "" {
				out = append(out, domain.Block{Kind: domain.BlockText, Text: text.String()})
			}
		case "tool_use":
			input := json.RawMessage(`{}`)
			if args := a.args[index]; args != nil && args.Len() > 0 {
				input = objectOrEmpty(json.RawMessage(args.String()))
			}

			out = append(out, domain.Block{Kind: domain.BlockToolUse, ID: start.ID, Name: start.Name, Input: input})
		}
	}

	return out
}
