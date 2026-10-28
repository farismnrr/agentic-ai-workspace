package codingtool

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	egressapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	egressdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const maxHostWords = 8

type arguments struct {
	Host   string `json:"host"`
	Reason string `json:"reason"`
}

// Tool records an agent's request using scope supplied by the task runner.
type Tool struct{ service *egressapp.Service }

func New(service *egressapp.Service) Tool { return Tool{service: service} }

// Definitions returns the model-facing request tool only while grants are active.
func (t Tool) Definitions() []taskdomain.ToolDef {
	if t.service == nil {
		return nil
	}
	return []taskdomain.ToolDef{{
		Name: codingdomain.ToolRequestEgress.Name, Description: codingdomain.ToolRequestEgress.Description,
		Schema: json.RawMessage(`{"type":"object","properties":{"host":{"type":"string",` +
			`"description":"Only the host name, such as example.com"},"reason":{"type":"string",` +
			`"description":"Why the task needs it, in one or two plain sentences"}},"required":["host","reason"]}`),
	}}
}

// Describe summarizes a request for the task thread without treating model fields as scope.
func (t Tool) Describe(input json.RawMessage) taskapp.Described {
	var in arguments
	_ = json.Unmarshal(input, &in)
	host := strings.Join(strings.Fields(in.Host), " ")
	if words := strings.Fields(host); len(words) > maxHostWords {
		host = strings.Join(words[:maxHostWords], " ") + "…"
	}
	return taskapp.Described{Ask: "Asked to reach " + host, Technical: "request_egress " + in.Host}
}

// Execute records a request under the task's server-created scope and agent identity.
func (t Tool) Execute(ctx context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
	if t.service == nil {
		return failure("There is no tool called " + call.Name + ".")
	}
	var in arguments
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return failure("The tool arguments were not valid JSON.")
	}
	by := egressdomain.Actor{Kind: egressdomain.ActorAgent, ID: "agent:" + scope.TaskID}
	grant, existing, err := t.service.Ask(ctx, scope.ProjectID, scope.TaskID,
		strings.TrimSpace(in.Host), strings.TrimSpace(in.Reason), by)
	if err != nil {
		return failure(explain(err))
	}
	return taskapp.ToolResult{Output: answer(grant, existing)}
}

func failure(message string) taskapp.ToolResult {
	return taskapp.ToolResult{IsError: true, Output: message}
}

func answer(g egressdomain.Grant, existing bool) string {
	switch {
	case g.Status == egressdomain.StatusApproved:
		return g.Host + " is allowed for this task until " + g.ExpiresAt.UTC().Format("15:04 on 2 Jan") +
			" UTC. Try again."
	case existing:
		return "You already asked for " + g.Host + ". The person has not answered yet. " +
			"Carry on without it, or tell them what you need."
	default:
		return "Asked the person to allow " + g.Host + " for this task. It stays blocked until they approve; you " +
			"cannot approve it. Carry on without it for now, or tell them what you need it for."
	}
}

func explain(err error) string {
	switch {
	case errors.Is(err, egressdomain.ErrInvalidHost):
		return "That is not a host name I can ask for. Give only the name, such as example.com, " +
			"without a port, a path or a wildcard."
	case errors.Is(err, egressdomain.ErrInvalidRequest):
		return "Say why the task needs the website, in a sentence or two (under 500 characters)."
	case errors.Is(err, egressapp.ErrTooManyPending):
		return "There are already several requests waiting for the person. Wait for their answer before asking for more."
	case errors.Is(err, egressapp.ErrRateLimited):
		return "You have asked for websites too often for this task. Wait before asking again."
	case errors.Is(err, egressapp.ErrScope):
		return "This task is no longer open, so it cannot ask for websites."
	default:
		return "I could not record the request."
	}
}
