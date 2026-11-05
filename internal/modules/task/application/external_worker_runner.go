package application

import (
	"context"
	"encoding/json"
	"slices"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type cancellationAwareTools struct {
	ToolSet
}

func (t cancellationAwareTools) Execute(ctx context.Context, scope ToolScope, call Call) ToolResult {
	if err := ctx.Err(); err != nil {
		return ToolResult{Output: err.Error(), IsError: true}
	}
	return t.ToolSet.Execute(ctx, scope, call)
}

type approvalResearcherTools struct {
	ToolSet
}

func (t approvalResearcherTools) Execute(ctx context.Context, scope ToolScope, call Call) ToolResult {
	if refusal := approvalResearcherRefusal(scope.RoleKey, call); refusal != "" {
		return ToolResult{Output: refusal, IsError: true}
	}
	return t.ToolSet.Execute(ctx, scope, call)
}

func approvalResearcherRefusal(roleKey string, call Call) string {
	if roleKey != "researcher" {
		return "Pre-approval Researcher access is limited to its assigned read-only role."
	}
	if call.Name == "git" || call.Name == "gh" {
		var input struct {
			Args []string `json:"args"`
		}
		if json.Unmarshal(call.Input, &input) != nil || !codingdomain.VCSReadOnly(call.Name, input.Args) {
			return "Git and GitHub changes require the person's PRD approval."
		}
	}
	if slices.Contains(writeToolNames, call.Name) {
		return "Changes and execution require the person's PRD approval."
	}
	return ""
}

func externalWorkerRunner(r *Runner) *Runner {
	worker := *r
	worker.externalFacts = true
	worker.externalWorker = true
	tools := ToolSet(cancellationAwareTools{ToolSet: r.deps.Tools})
	if r.approvalResearcher {
		tools = approvalResearcherTools{ToolSet: tools}
	}
	worker.deps.Tools = tools
	return &worker
}
