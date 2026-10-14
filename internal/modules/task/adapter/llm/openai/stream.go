package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type streamChunk struct {
	Usage *struct {
		Prompt     int64 `json:"prompt_tokens"`
		Completion int64 `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Choices []struct {
		Delta struct {
			Content   *string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
				Index int `json:"index"`
			} `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

type pendingCall struct {
	id, name string
	args     strings.Builder
}

// assembly collects the pieces of a streamed answer into whole blocks.
type assembly struct {
	calls  map[int]*pendingCall
	onText func(string)
	text   strings.Builder
	usage  domain.Usage
	sent   bool
}

func newAssembly(onText func(string)) *assembly {
	return &assembly{calls: map[int]*pendingCall{}, onText: onText}
}

// stream asks for a streamed answer. Gateways that do not support streaming
// answer with a client error before any text, and then the turn is asked again without it.
func (c *Client) stream(ctx context.Context, in application.Request) (application.Reply, error) {
	body := encode(in)
	body.Stream = true
	body.StreamOptions = &streamOptions{IncludeUsage: true}
	assembled := newAssembly(in.OnText)
	headers := map[string]string{"Authorization": "Bearer " + c.apiKey}

	err := c.http.Stream(ctx, c.baseURL+"/chat/completions", headers, body, assembled.handle)
	if modelErr, ok := domain.AsModelError(err); ok && !assembled.sent && unsupported(modelErr.Status) {
		in.OnText = nil

		return c.Complete(ctx, in)
	}

	if err != nil {
		return application.Reply{}, err //nolint:wrapcheck // transport already returns domain errors
	}

	return application.Reply{Blocks: assembled.blocks(), Usage: assembled.usage}, nil
}

func unsupported(status int) bool {
	return status == http.StatusBadRequest || status == http.StatusNotFound || status == http.StatusUnprocessableEntity
}

func (a *assembly) handle(_, data string) error {
	if strings.TrimSpace(data) == "[DONE]" {
		return nil
	}

	var parsed streamChunk
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil //nolint:nilerr // unreadable keep-alive lines are skipped
	}

	if parsed.Error != nil {
		return &domain.ModelError{Message: parsed.Error.Message}
	}

	if parsed.Usage != nil {
		a.usage = domain.Usage{Input: parsed.Usage.Prompt, Output: parsed.Usage.Completion}
	}

	for _, choice := range parsed.Choices {
		a.add(choice.Delta.Content)

		for _, call := range choice.Delta.ToolCalls {
			a.addCall(call.Index, call.ID, call.Function.Name, call.Function.Arguments)
		}
	}

	return nil
}

// addCall merges one fragment of a tool call into the call it belongs to.
func (a *assembly) addCall(index int, id, name, arguments string) {
	pending := a.calls[index]
	if pending == nil {
		pending = &pendingCall{}
		a.calls[index] = pending
	}

	if id != "" {
		pending.id = id
	}

	if name != "" {
		pending.name = name
	}

	_, _ = pending.args.WriteString(arguments)
}

func (a *assembly) add(content *string) {
	if content == nil || *content == "" {
		return
	}

	_, _ = a.text.WriteString(*content)

	if a.onText != nil {
		a.sent = true
		a.onText(*content)
	}
}

func (a *assembly) blocks() []domain.Block {
	var out []domain.Block

	if strings.TrimSpace(a.text.String()) != "" {
		out = append(out, domain.Block{Kind: domain.BlockText, Text: a.text.String()})
	}

	order := make([]int, 0, len(a.calls))
	for index := range a.calls {
		order = append(order, index)
	}

	sort.Ints(order)

	for _, index := range order {
		call := a.calls[index]
		out = append(out, domain.Block{
			Kind: domain.BlockToolUse, ID: callID(call.id), Name: call.name, Input: parseArguments(call.args.String()),
		})
	}

	return out
}
