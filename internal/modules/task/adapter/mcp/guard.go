package mcp

import (
	"context"
	"errors"
	"fmt"
	"slices"

	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const clientKey = "client"

func guarded[In, Out any](
	t tools, name, scope string, describe func(In) string, project func(In) string,
	task func(In) string, run func(context.Context, string, In) (Out, error),
) sdk.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error) {
		var empty Out
		if req == nil || req.GetExtra() == nil {
			return failure("Sign in again to use this tool.", nil), empty, nil
		}
		token := req.GetExtra().TokenInfo
		if token == nil || token.UserID == "" {
			return failure("Sign in again to use this tool.", nil), empty, nil
		}
		client, _ := token.Extra[clientKey].(string)
		entry := Entry{
			UserID: token.UserID, Client: client, Tool: name, Summary: describe(in),
			Project: project(in), Task: task(in),
		}
		if result := refuse(ctx, t, name, scope, token.Scopes, &entry); result != nil {
			t.record(ctx, entry)
			return result, empty, nil
		}

		out, err := run(ctx, token.UserID, in)
		if err != nil {
			entry.Outcome, entry.Detail = "failed", errorMessage(err)
			t.record(ctx, entry)
			return failure(entry.Detail, nil), empty, nil
		}
		if result, ok := any(out).(taskOutput); ok {
			entry.Project, entry.Task = result.ProjectID, result.TaskID
		}
		entry.Outcome = "ok"
		t.record(ctx, entry)

		return nil, out, nil
	}
}

func refuse(ctx context.Context, t tools, name, scope string, scopes []string, entry *Entry) *sdk.CallToolResult {
	allowed, err := t.gate.Allowed(ctx, entry.UserID, name)
	if err != nil {
		entry.Outcome, entry.Detail = "failed", "could not check the tool switch"
		return failure("The tool switch could not be checked. Try again in a moment.", nil)
	}
	if !allowed {
		entry.Outcome, entry.Detail = "off", "tool is turned off"
		return failure("This tool is turned off. Ask the person to turn it on in Settings, Spaces & tools.", nil)
	}
	if !slices.Contains(scopes, scope) {
		entry.Outcome, entry.Detail = "forbidden", "missing permission "+scope
		message := "This tool needs the " + scope + " permission. Reconnect and approve it."
		return failure(message, scopeChallenge(t.metadataURL, scope))
	}
	return nil
}

func scopeChallenge(metadataURL, scope string) sdk.Meta {
	value := fmt.Sprintf(`Bearer error="insufficient_scope", scope=%q, resource_metadata=%q`, scope, metadataURL)
	return sdk.Meta{"mcp/www_authenticate": []string{value}}
}

func failure(message string, meta sdk.Meta) *sdk.CallToolResult {
	return &sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: message}}, Meta: meta}
}

func errorMessage(err error) string {
	switch {
	case errors.Is(err, taskdomain.ErrNoProject), errors.Is(err, taskdomain.ErrNotFound):
		return "The project or task was not found. Check the IDs and your access."
	case errors.Is(err, taskdomain.ErrWrongState), errors.Is(err, taskdomain.ErrBusy),
		errors.Is(err, taskdomain.ErrNeedsAnswer):
		return "This task cannot do that right now. Open it in the app to check its status or pending approval."
	case errors.Is(err, taskdomain.ErrBadRequest):
		return "Add a short description of what the team should do."
	case errors.Is(err, errInputTooLong):
		return "Keep the prompt or message to 12,000 characters or fewer."
	case errors.Is(err, taskdomain.ErrTeamUnavailable):
		return "The team is not ready. Check the team setup in the app, then try again."
	default:
		return "The request could not be completed. Try again later."
	}
}

func (t tools) record(ctx context.Context, entry Entry) {
	entry.Project = activityID(entry.Project)
	entry.Task = activityID(entry.Task)
	if t.activity != nil {
		_ = t.activity.Record(ctx, entry)
	}
}

func activityID(value string) string {
	const maxIDRunes = 64
	runes := []rune(value)
	if len(runes) > maxIDRunes {
		return string(runes[:maxIDRunes])
	}

	return value
}

func projectOf(in startInput) string          { return in.ProjectID }
func projectOfTask(in taskInput) string       { return in.ProjectID }
func projectOfMessage(in messageInput) string { return in.ProjectID }
func taskOfTask(in taskInput) string          { return in.TaskID }
func taskOfMessage(in messageInput) string    { return in.TaskID }
