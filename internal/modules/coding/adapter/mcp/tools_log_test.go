package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"
)

var errLogPersistenceFailure = errors.New("write failed")

func TestLogMessagePersistsWorkKindsBeforeReportingSaved(t *testing.T) {
	calls := &fakeWorkCalls{}
	legacy := &fakeTaskLog{}
	tools := logTestTools(calls, legacy)
	ctx := context.WithValue(t.Context(), identityKey{}, identity{
		User: "u1", Client: "ChatGPT", WorkID: "work-1",
	})
	for _, role := range []string{"user", "assistant"} {
		result, rep, err := tools.logMessage(ctx, logInput{Role: role, Text: "hello"})
		if err != nil || !result.Saved || rep.note != nil {
			t.Fatalf("%s result=%+v report=%+v error=%v", role, result, rep, err)
		}
	}
	if len(calls.recorded) != 2 || calls.recorded[0].Note.Kind != NoteUser ||
		calls.recorded[1].Note.Kind != NoteAssistant || legacy.logs != 0 {
		t.Fatalf("work calls=%+v legacy logs=%d", calls.recorded, legacy.logs)
	}
}

func TestLogMessageDoesNotClaimSavedAfterPersistenceFailure(t *testing.T) {
	calls := &fakeWorkCalls{recordErr: errLogPersistenceFailure}
	tools := logTestTools(calls, &fakeTaskLog{})
	ctx := context.WithValue(t.Context(), identityKey{}, identity{
		User: "u1", Client: "ChatGPT", WorkID: "work-1",
	})
	result, _, err := tools.logMessage(ctx, logInput{Role: "user", Text: "hello"})
	if err == nil || result.Saved {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

func TestLogMessageKeepsLegacyThreadPath(t *testing.T) {
	legacy := &fakeTaskLog{}
	tools := logTestTools(&fakeWorkCalls{}, legacy)
	ctx := context.WithValue(t.Context(), identityKey{}, identity{User: "u1", Client: "ChatGPT"})
	result, _, err := tools.logMessage(ctx, logInput{Role: "assistant", Text: "answer"})
	if err != nil || !result.Saved || legacy.logs != 1 || legacy.begins != 0 ||
		!strings.Contains(result.Message, "begin_work with a role") ||
		!strings.Contains(result.Message, "developer, researcher") {
		t.Fatalf("result=%+v logs=%d error=%v", result, legacy.logs, err)
	}
}

func logTestTools(calls *fakeWorkCalls, legacy *fakeTaskLog) *tools {
	return &tools{
		workspaces: recordingWorkspace{services: Services{ProjectID: "project-1"}},
		guard:      &guard{tasks: legacy, workCalls: calls},
		works:      &fakeWorkSessions{},
	}
}
