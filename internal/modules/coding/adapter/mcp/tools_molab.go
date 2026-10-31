package mcp

import (
	"context"
	"errors"

	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

var errMoLabDeveloperRequired = errors.New("MoLab requires the Developer role; use set_role with role developer on this work_id")

// MoLabExecutor runs one explicitly authorized notebook operation in the same
// isolated sandbox used by AI Code's internal agent.
type MoLabExecutor interface {
	Execute(ctx context.Context, url, token, code string) MoLabResult
}

type MoLabResult struct {
	Output  string `json:"output,omitempty"`
	Problem string `json:"problem,omitempty"`
}

type moLabInput struct {
	Target
	URL   string `json:"url" jsonschema:"HTTPS URL of the active sb-*.sb.molab.run notebook"`
	Token string `json:"token" jsonschema:"temporary MoLab token the user provided in chat; never repeat or log it"`
	Code  string `json:"code" jsonschema:"Python to execute in the running MoLab notebook"`
}

func (t *tools) addMoLabTools(server *sdk.Server) {
	// Deliberately constant description: the access token must never enter
	// activity logs, work threads, error details or shell arguments.
	sdk.AddTool(server, toolOf(domain.ToolMarimoExecute),
		guarded(t.guard, domain.ToolMarimoExecute,
			func(moLabInput) string { return "Execute Python in MoLab" }, t.executeMoLab))
}

func (t *tools) executeMoLab(ctx context.Context, in moLabInput) (MoLabResult, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return MoLabResult{}, report{}, err
	}
	// RequireRole also revalidates the session against the current enabled roster.
	who, _ := ctx.Value(identityKey{}).(identity)
	if err := t.works.RequireRole(ctx, who.User, who.Client, svc.ProjectID, who.WorkID, "developer"); err != nil {
		if errors.Is(err, taskdomain.ErrBadRole) {
			return MoLabResult{}, report{}, errMoLabDeveloperRequired
		}
		return MoLabResult{}, report{}, err
	}
	if svc.MoLab == nil {
		return MoLabResult{Problem: "MoLab sandbox is unavailable."}, report{detail: "unavailable", failed: true}, nil
	}
	result := svc.MoLab.Execute(ctx, in.URL, in.Token, in.Code)
	if result.Problem != "" {
		return result, report{detail: "MoLab execution failed", failed: true}, nil
	}
	return result, report{detail: "MoLab execution succeeded"}, nil
}
