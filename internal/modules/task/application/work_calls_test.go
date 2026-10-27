package application

import (
	"context"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestFinishCallNotifiesTheTaskSubscribers(t *testing.T) {
	store := &workCallCaptureStore{}
	notified := &callNotificationCapture{}
	works := &Works{
		store: store,
		deps:  Deps{Tasks: callTaskLookup{task: domain.Task{ID: "task", UserID: "user"}}, Notify: notified},
	}
	if err := works.FinishCall(context.Background(), "task", "work", "event", domain.Event{}); err != nil {
		t.Fatal(err)
	}
	if notified.userID != "user" || notified.taskID != "task" {
		t.Fatalf("notification = user %q task %q", notified.userID, notified.taskID)
	}
}

type callTaskLookup struct {
	TaskStore
	task domain.Task
}

func (f callTaskLookup) Find(context.Context, string, string) (domain.Task, error) {
	return f.task, nil
}

type callNotificationCapture struct{ userID, taskID string }

func (f *callNotificationCapture) Changed(userID, taskID string) {
	f.userID, f.taskID = userID, taskID
}

func (*callNotificationCapture) Delta(string, string) {}
func (*callNotificationCapture) FilesChanged(string)  {}

func TestCleanWorkEventRedactsCommandAndRunSummary(t *testing.T) {
	event := domain.Event{
		Text:      "command token=secret-value",
		Technical: "DATABASE_PASSWORD=hunter2",
		Run:       &domain.RunInfo{Name: "go test --token=secret-value", ExitCode: 1},
	}
	cleanWorkEvent(&event)
	for _, value := range []string{event.Text, event.Technical, event.Run.Name} {
		if strings.Contains(value, "secret-value") || strings.Contains(value, "hunter2") {
			t.Errorf("secret remained in stored work event: %q", value)
		}
	}
}

func TestRecordCallPreservesConversationKinds(t *testing.T) {
	store := &workCallCaptureStore{}
	works := &Works{store: store}
	for _, kind := range []domain.EventKind{domain.EventRequest, domain.EventMessage} {
		if _, err := works.RecordCall(context.Background(), "u", "ChatGPT", "p", "w",
			domain.Event{Kind: kind, Text: "conversation"}); err != nil {
			t.Fatal(err)
		}
		if store.event.Kind != kind {
			t.Fatalf("stored kind = %q, want %q", store.event.Kind, kind)
		}
	}
}

type workCallCaptureStore struct {
	WorkStore
	event domain.Event
}

func (s *workCallCaptureStore) AppendWorkCall(
	_ context.Context, _, _, _, _ string, event domain.Event,
) (string, domain.Event, error) {
	s.event = event
	event.ID = "event"
	return "task", event, nil
}

func (*workCallCaptureStore) UpdateWorkCall(
	context.Context, string, string, string, domain.Event,
) (domain.Event, error) {
	return domain.Event{}, nil
}
