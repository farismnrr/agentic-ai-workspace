package bootstrap

import (
	"context"

	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
)

func (a appWorks) SaveResearch(ctx context.Context, user, client, project, workID string,
	in codingmcp.ResearchWorkInput,
) (codingmcp.WorkMutation, error) {
	return a.saveMutation(ctx, user, client, project, workID, "save work research", "research.md",
		func() error { return a.works.SaveResearch(ctx, user, client, project, workID, in.Markdown, in.Sources) },
		append([]string{in.Markdown}, in.Sources...)...)
}

func (a appWorks) SaveRequirements(
	ctx context.Context, user, client, project, workID, markdown string,
) (codingmcp.WorkMutation, error) {
	if err := a.works.SaveRequirements(ctx, user, client, project, workID, markdown); err != nil {
		return codingmcp.WorkMutation{}, workBridgeError("save work requirements", err)
	}
	return a.mutationState(ctx, user, client, project, workID, "requirements.md", markdown)
}

func (a appWorks) Progress(ctx context.Context, user, client, project, workID string,
	in codingmcp.ProgressWorkInput,
) (codingmcp.WorkMutation, error) {
	if err := a.works.Progress(ctx, user, client, project, workID, in.Message, in.Claims); err != nil {
		return codingmcp.WorkMutation{}, workBridgeError("report work progress", err)
	}
	return a.mutationState(ctx, user, client, project, workID, "", append([]string{in.Message}, in.Claims...)...)
}

func (a appWorks) saveMutation(ctx context.Context, user, client, project, workID, action, path string,
	save func() error, texts ...string,
) (codingmcp.WorkMutation, error) {
	if err := save(); err != nil {
		return codingmcp.WorkMutation{}, workBridgeError(action, err)
	}
	return a.mutationState(ctx, user, client, project, workID, path, texts...)
}

func (a appWorks) mutationState(ctx context.Context, user, client, project, workID string,
	path string, texts ...string,
) (codingmcp.WorkMutation, error) {
	state, err := a.works.State(ctx, user, client, project, workID)
	if err != nil {
		return codingmcp.WorkMutation{}, workBridgeError("read updated work", err)
	}
	exportPath := ""
	if path != "" {
		exportPath = "project-notes/tasks/" + state.TaskID + "/" + path
	}
	return codingmcp.WorkMutation{
		WorkID: workID, Path: exportPath, Revision: state.Revision, State: string(state.State),
		Warnings: RedactionWarnings(texts...),
	}, nil
}
