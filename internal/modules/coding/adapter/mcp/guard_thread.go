package mcp

import (
	"context"
	"slices"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// newState starts what a call learns about itself. A call that takes a while writes its step into the thread as
// soon as its project is known.
func (g *guard) newState(
	ctx context.Context, info domain.ToolInfo, entry *Entry, req *sdk.CallToolRequest, workID string,
) *callState {
	state := &callState{workID: workID}
	if info.Group == domain.GroupExec {
		switch {
		case workID != "" && g.workCalls != nil:
			state.begin = func(project string) { g.beginWork(ctx, entry, state, project, metaKeys(req)) }
		case workID == "" && g.tasks != nil:
			state.begin = func(project string) { g.begin(ctx, entry, state, project, metaKeys(req)) }
		}
	}

	return state
}

// failed records a call that returned an error and gives the person's model the readable form of it.
func (g *guard) failed(ctx context.Context, info domain.ToolInfo, entry *Entry, err error) error {
	entry.Outcome, entry.Detail = OutcomeFailed, UserFacing(err).Error()
	g.thread(ctx, info, entry, report{})
	g.record(ctx, *entry)

	return UserFacing(err)
}

// succeeded records a call that ran, which may still have failed in its own terms (rep.failed).
func (g *guard) succeeded(ctx context.Context, info domain.ToolInfo, entry *Entry, rep report) {
	entry.Outcome, entry.Detail, entry.Undo = OutcomeOK, rep.detail, rep.undo
	if rep.project != "" {
		entry.Project = rep.project
	}

	if rep.failed {
		entry.Outcome = OutcomeFailed
	}

	g.thread(ctx, info, entry, rep)
	g.record(ctx, *entry)
}

// begin writes a call that takes a while into the thread as soon as it starts, and shows the thread as working,
// so the person sees a long command while it runs and not only after it ends. meta lists what the app's request
// carried besides the arguments, which shows what an app tells us about its conversation.
func (g *guard) begin(ctx context.Context, entry *Entry, state *callState, project, meta string) {
	note := Note{Kind: NoteAction, Text: "Running: " + entry.Summary, Technical: meta}
	if id, err := g.tasks.Begin(ctx, entry.UserID, entry.Client, project, note); err == nil {
		state.taskID = id
	}
}

// end says a call that begin started is over.
func (g *guard) end(ctx context.Context, state *callState) {
	if state.eventID != "" {
		return
	}
	if state.taskID != "" {
		g.tasks.End(context.WithoutCancel(ctx), state.taskID)
	}
}

// metaKeys names what the request carried in _meta, without the values.
func metaKeys(req *sdk.CallToolRequest) string {
	if req == nil || req.Params == nil {
		return ""
	}

	keys := make([]string, 0, len(req.Params.GetMeta()))
	for key := range req.Params.GetMeta() {
		keys = append(keys, key)
	}

	slices.Sort(keys)

	if len(keys) == 0 {
		return ""
	}

	return "the app sent: " + strings.Join(keys, ", ")
}

// quiet are the tools whose calls say nothing about the work, so they are not written into the thread.
var quiet = []string{domain.ToolWhoami.Name, domain.ToolListProjects.Name, domain.ToolLogMessage.Name}

// thread writes the call into the app's thread, so the person can read what the app did, reading included. A
// failure to write never turns a finished call into an error.
func (g *guard) thread(ctx context.Context, info domain.ToolInfo, entry *Entry, rep report) {
	if workID := workIDFrom(ctx); workID != "" {
		if g.workCalls != nil {
			g.workThread(ctx, info, entry, rep, workID)
		}
		return
	}
	g.legacyThread(ctx, info, entry, rep)
}

func (g *guard) legacyThread(ctx context.Context, info domain.ToolInfo, entry *Entry, rep report) {
	note := rep.note
	if note == nil && !slices.Contains(quiet, info.Name) {
		note = &Note{Kind: NoteAction, Text: entry.Summary, Technical: entry.Detail, Run: rep.run}
		if entry.Outcome == OutcomeFailed && rep.run == nil {
			note.Text = "Tried to, but it did not work: " + entry.Summary
		}
	}

	if note == nil || g.tasks == nil || entry.Project == "" {
		return
	}

	if id, err := g.tasks.Log(ctx, entry.UserID, entry.Client, entry.Project, *note); err == nil {
		entry.Task = id
	}
}
