package mcp

import (
	"context"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

var errBadMessage = errors.New("say who wrote the message (user or assistant) and what it says")

type logInput struct {
	Role string `json:"role" jsonschema:"user for what the person asked, assistant for your answer"`
	Text string `json:"text" jsonschema:"the message, or a short summary of your answer"`
}

type logOutput struct {
	Saved bool `json:"saved"`
}

func (t *tools) addLogTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolLogMessage), guarded(t.guard, domain.ToolLogMessage,
		func(in logInput) string { return "save a message from " + in.Role }, t.logMessage))
}

func (t *tools) logMessage(ctx context.Context, in logInput) (logOutput, report, error) {
	text := strings.TrimSpace(in.Text)

	kind := NoteUser
	if in.Role == string(NoteAssistant) {
		kind = NoteAssistant
	} else if in.Role != string(NoteUser) {
		return logOutput{}, report{}, errBadMessage
	}

	if text == "" {
		return logOutput{}, report{}, errBadMessage
	}

	// The project is not needed here, but asking for it records where the message belongs.
	if _, err := t.services(ctx); err != nil {
		return logOutput{}, report{}, err
	}

	return logOutput{Saved: true}, report{note: &Note{Kind: kind, Text: text}}, nil
}
