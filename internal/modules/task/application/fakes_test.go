package application_test

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type fakePersona struct{}

func (fakePersona) Persona(context.Context, string) (application.Persona, error) {
	return application.Persona{Name: "Sari-bot", Rules: []string{"Never change prices"}}, nil
}

// fakeGate has these tools switched on, unless a test switches one off; skip lets the agent act unasked.
type fakeGate struct {
	off  map[string]bool
	skip bool
}

func (g *fakeGate) Enabled(context.Context, string) (map[string]bool, error) {
	on := map[string]bool{}

	known := []string{
		"read_file", "edit_file", "write_file", "delete_file", "run_task", "run_command", "start_site", "screenshot", "test",
	}

	for _, name := range known {
		on[name] = !g.off[name]
	}

	return on, nil
}

func (g *fakeGate) SkipApprovals(context.Context, string) (bool, error) { return g.skip, nil }

type fakeProjects struct{}

func (fakeProjects) Name(_ context.Context, _, projectID string) (string, error) {
	if projectID != "proj" {
		return "", domain.ErrNoProject
	}

	return "Bakery", nil
}

// fakeTools runs no real tools: results come from a table and every call is remembered.
type fakeTools struct {
	risks    map[string]domain.Risk
	results  map[string]application.ToolResult
	executed []string
	undone   [][]byte
}

func (f *fakeTools) Definitions() []domain.ToolDef { return []domain.ToolDef{{Name: "read_file"}} }

func (f *fakeTools) Risk(name string) domain.Risk { return f.risks[name] }

func (f *fakeTools) Describe(c application.Call) application.Described {
	return application.Described{Text: "Did " + c.Name, Ask: "May I " + c.Name + "?", Technical: c.Name}
}

func (f *fakeTools) Execute(_ context.Context, _ string, c application.Call) application.ToolResult {
	f.executed = append(f.executed, c.Name)

	if result, ok := f.results[c.Name]; ok {
		return result
	}

	return application.ToolResult{Output: "ok"}
}

func (f *fakeTools) Undo(_ context.Context, _ string, blob []byte) error {
	f.undone = append(f.undone, blob)

	return nil
}

type fakeDispatcher struct {
	enqueued []string
	cancel   application.CancelState
}

func (f *fakeDispatcher) Enqueue(taskID, _ string) { f.enqueued = append(f.enqueued, taskID) }

func (f *fakeDispatcher) Cancel(string) application.CancelState { return f.cancel }
