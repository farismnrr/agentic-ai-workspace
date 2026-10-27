package bootstrap

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

var toolRequestEgress = codingdomain.ToolRequestEgress.Name

type egressArgs struct {
	Host   string `json:"host"`
	Reason string `json:"reason"`
}

// egressDefinitions describes request_egress. It exists only when the server has egress grants running, so with the
// flag off the model is never told about it.
func (a *agentTools) egressDefinitions() []taskdomain.ToolDef {
	if a.egress == nil {
		return nil
	}

	return []taskdomain.ToolDef{{
		Name: toolRequestEgress, Description: codingdomain.ToolRequestEgress.Description,
		Schema: schema(`"host":{"type":"string","description":"Only the host name, such as example.com"},`+
			`"reason":{"type":"string","description":"Why the task needs it, in one or two plain sentences"}`,
			`"host","reason"`),
	}}
}

func describeEgress(input json.RawMessage) taskapp.Described {
	var in egressArgs

	_ = json.Unmarshal(input, &in)

	return said("Asked to reach "+clipWords(in.Host), "", "request_egress "+in.Host)
}

// otherTools are the tools that take their own path instead of a handler of the coding space.
func (a *agentTools) otherTools() map[string]func(context.Context, taskapp.ToolScope, taskapp.Call) taskapp.ToolResult {
	return map[string]func(context.Context, taskapp.ToolScope, taskapp.Call) taskapp.ToolResult{
		toolRequestEgress:     a.executeEgress,
		toolScreenshot:        a.executeShot,
		toolScreenshotSession: a.executeScreenshotSession,
		toolDeployExtension: func(ctx context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
			return a.executeExtension(ctx, scope.ProjectID, call)
		},
		toolCILogs: a.executeCILogs,
		toolViewImage: func(_ context.Context, scope taskapp.ToolScope, call taskapp.Call) taskapp.ToolResult {
			return a.executeView(scope.ProjectID, call)
		},
	}
}

// executeEgress records a request. The project and the task come from the scope the task loop built from its own
// record, never from the arguments, and the actor is always the agent, which can ask and nothing more: there is no
// path from here to approve, deny or change a grant.
func (a *agentTools) executeEgress(
	ctx context.Context, scope taskapp.ToolScope, call taskapp.Call,
) taskapp.ToolResult {
	if a.egress == nil {
		return problem("There is no tool called " + call.Name + ".")
	}

	var in egressArgs
	if err := json.Unmarshal(call.Input, &in); err != nil {
		return problem("The tool arguments were not valid JSON.")
	}

	by := egressdomain.Actor{Kind: egressdomain.ActorAgent, ID: "agent:" + scope.TaskID}

	grant, existing, err := a.egress.service.Ask(ctx, scope.ProjectID, scope.TaskID, strings.TrimSpace(in.Host),
		strings.TrimSpace(in.Reason), by)
	if err != nil {
		return problem(egressProblem(err))
	}

	return taskapp.ToolResult{Output: egressAnswer(grant, existing)}
}

func egressAnswer(g egressdomain.Grant, existing bool) string {
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

// egressProblem is what the model is told when a request is refused. None of it names an internal.
func egressProblem(err error) string {
	switch {
	case errors.Is(err, egressdomain.ErrInvalidHost):
		return "That is not a host name I can ask for. Give only the name, such as example.com, " +
			"without a port, a path or a wildcard."
	case errors.Is(err, egressdomain.ErrInvalidRequest):
		return "Say why the task needs the website, in a sentence or two (under 500 characters)."
	case errors.Is(err, egressapp.ErrTooManyPending):
		return "There are already several requests waiting for the person. " +
			"Wait for their answer before asking for more."
	case errors.Is(err, egressapp.ErrRateLimited):
		return "You have asked for websites too often for this task. Wait before asking again."
	case errors.Is(err, egressapp.ErrScope):
		return "This task is no longer open, so it cannot ask for websites."
	}

	return "I could not record the request."
}
