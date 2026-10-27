package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const (
	workUndoRecordAttempts = 2
	workCallIDSize         = 16
)

func workIDFrom(ctx context.Context) string {
	who, _ := ctx.Value(identityKey{}).(identity)
	return who.WorkID
}

func (g *guard) beginWork(ctx context.Context, entry *Entry, state *callState, project, meta string) {
	call := WorkCall{Note: Note{Kind: NoteAction, Text: "Running: " + entry.Summary, Technical: meta}, Phase: "running"}
	taskID, eventID, err := g.workCalls.Start(ctx, entry.UserID, entry.Client, project, state.workID, call)
	if err != nil {
		return
	}
	state.taskID, state.eventID = taskID, eventID
	entry.Task = taskID
}

func (g *guard) workThread(ctx context.Context, info domain.ToolInfo, entry *Entry, rep report, workID string) {
	entry.WorkID = workID
	note := workNote(info, entry, rep)
	if note == nil || entry.Project == "" {
		return
	}
	call := workCall(rep, *note)
	state, _ := ctx.Value(stateKey{}).(*callState)
	if state != nil && state.eventID != "" {
		g.finishWorkCall(ctx, entry, state, rep, call)
		return
	}
	if call.Evidence == "" && rep.run != nil {
		call.Evidence = "ran"
	}
	g.recordWorkCall(ctx, entry, workID, call)
}

func workNote(info domain.ToolInfo, entry *Entry, rep report) *Note {
	if rep.note != nil {
		return rep.note
	}
	if slices.Contains(quiet, info.Name) {
		return nil
	}
	note := &Note{Kind: NoteAction, Text: entry.Summary, Technical: entry.Detail, Run: rep.run}
	if entry.Outcome == OutcomeFailed && rep.run == nil {
		note.Text = "Tried to, but it did not work: " + entry.Summary
	}
	return note
}

func (g *guard) finishWorkCall(ctx context.Context, entry *Entry, state *callState, rep report, call WorkCall) {
	call.Phase = "finished"
	if rep.run != nil {
		call.Evidence = "ran"
	}
	if err := g.workCalls.Finish(ctx, state.taskID, state.workID, state.eventID, call); err == nil {
		entry.Task = state.taskID
	}
}

func (g *guard) recordWorkCall(ctx context.Context, entry *Entry, workID string, call WorkCall) {
	callID, err := workCallID()
	if err != nil {
		entry.Detail += "; recorded:false: could not prepare a retry-safe event"
		return
	}
	call.CallID = callID
	for attempt := 0; attempt < recordAttempts(call); attempt++ {
		taskID, err := g.workCalls.Record(ctx, entry.UserID, entry.Client, entry.Project, workID, call)
		if err == nil {
			entry.Task = taskID
			if len(call.Undo) > 0 {
				entry.Undo = nil
				entry.Detail += "; undo from the task's Changes tab"
			}
			return
		}
	}
	entry.Detail += "; recorded:false: undo remains in the activity log"
}

func workCallID() (string, error) {
	bytes := make([]byte, workCallIDSize)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("create connected-app call ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func recordAttempts(call WorkCall) int {
	if len(call.Undo) > 0 {
		return workUndoRecordAttempts
	}
	return 1
}

func workCall(rep report, note Note) WorkCall {
	if rep.run != nil {
		note.Run = rep.run
	}
	evidence := ""
	if len(rep.files) > 0 {
		evidence = "observed"
	}
	return WorkCall{Note: note, Files: rep.files, Undo: rep.undo, Evidence: evidence, Phase: "finished"}
}
