package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

var (
	errBadMessage            = errors.New("say who wrote the message (user or assistant) and what it says")
	errWorkThreadUnavailable = errors.New("work-session thread storage is unavailable")
	errThreadUnavailable     = errors.New("thread storage is unavailable")
)

type logInput struct {
	Target
	Role string `json:"role" jsonschema:"user for what the person asked, assistant for your answer"`
	Text string `json:"text" jsonschema:"the message, or a short summary of your answer"`
}

type logOutput struct {
	Message string `json:"message,omitempty"`
	Saved   bool   `json:"saved"`
}

func (t *tools) addLogTools(server *sdk.Server) {
	sdk.AddTool(server, toolOf(domain.ToolLogMessage), guarded(t.guard, domain.ToolLogMessage,
		func(in logInput) string { return "save a message from " + in.Role }, t.logMessage))
}

func (t *tools) logMessage(ctx context.Context, in logInput) (logOutput, report, error) {
	text := strings.TrimSpace(in.Text)

	kind := NoteUser
	switch in.Role {
	case string(NoteAssistant):
		kind = NoteAssistant
	case string(NoteUser):
	default:
		return logOutput{}, report{}, errBadMessage
	}

	if text == "" {
		return logOutput{}, report{}, errBadMessage
	}

	services, err := t.services(ctx)
	if err != nil {
		return logOutput{}, report{}, err
	}

	who, _ := ctx.Value(identityKey{}).(identity)
	note := Note{Kind: kind, Text: text}
	message := ""
	if who.WorkID == "" {
		message = legacyWorkHint(ctx, t.works, who.User)
	}
	if err := t.saveNote(ctx, who, services, note); err != nil {
		return logOutput{}, report{}, fmt.Errorf("save conversation message: %w", err)
	}

	return logOutput{Saved: true, Message: message}, report{}, nil
}

func legacyWorkHint(ctx context.Context, works WorkSessions, userID string) string {
	if works == nil {
		return "This legacy message is saved as a record only. For tracked work, call begin_work with a role; " +
			"enabled role keys are unavailable right now."
	}
	roles, err := works.AvailableRoles(ctx, userID)
	if err != nil {
		return "This legacy message is saved as a record only. For tracked work, call begin_work with a role; " +
			"enabled role keys are unavailable right now."
	}
	keys := make([]string, 0, len(roles))
	for _, role := range roles {
		keys = append(keys, role.Key)
	}
	if len(keys) == 0 {
		return "This legacy message is saved as a record only. For tracked work, call begin_work with a role; " +
			"enabled role keys: none."
	}
	return "This legacy message is saved as a record only. For tracked work, call begin_work with a role; " +
		"enabled role keys: " + strings.Join(keys, ", ") + "."
}

func (t *tools) saveNote(ctx context.Context, who identity, services Services, note Note) error {
	if who.WorkID != "" {
		if t.guard.workCalls == nil {
			return errWorkThreadUnavailable
		}
		callID, idErr := workCallID()
		if idErr != nil {
			return idErr
		}
		_, err := t.guard.workCalls.Record(ctx, who.User, who.Client, services.ProjectID,
			who.WorkID, WorkCall{CallID: callID, Note: note, Phase: "finished"})
		if err != nil {
			return fmt.Errorf("record work-session message: %w", err)
		}
		return nil
	}
	if t.guard.tasks == nil {
		return errThreadUnavailable
	}
	_, err := t.guard.tasks.Log(ctx, who.User, who.Client, services.ProjectID, note)
	if err != nil {
		return fmt.Errorf("record task message: %w", err)
	}
	return nil
}
