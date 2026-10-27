package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func (m *memory) CreateLinkedTask(
	_ context.Context,
	externalID string,
	task domain.Task,
	event domain.Event,
	prompt string,
) (domain.Task, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	external, ok := m.tasks[externalID]
	if !ok || external.UserID != task.UserID || external.ProjectID != task.ProjectID || external.ArchivedAt != nil {
		return domain.Task{}, false, domain.ErrNotFound
	}
	if external.LinkedTaskID != "" {
		linked := m.tasks[external.LinkedTaskID]
		if linked.ArchivedAt == nil {
			return linked, false, nil
		}
	}
	for _, existing := range m.tasks {
		if existing.OriginTaskID == externalID && existing.ArchivedAt == nil {
			return existing, false, nil
		}
	}
	external.LinkedTaskID = task.ID
	m.tasks[externalID] = external
	m.tasks[task.ID] = task
	event.ID, event.Seq, event.Version = "linked-request", 1, 1
	m.events[task.ID] = []domain.Event{event}
	m.transcript[task.ID] = []row{{role: domain.RoleUser, block: domain.Block{Kind: domain.BlockText, Text: prompt}}}
	return task, true, nil
}

func TestChatOnExternalCreatesOneInternalChatAndKeepsExternalTaskUntouched(t *testing.T) {
	r, external := externalChatRig(t)
	linkedID, err := r.service.ChatOnExternal(context.Background(), "u1", external.ID, "Check the search behavior")
	if err != nil {
		t.Fatal(err)
	}
	assertLinkedChat(t, r, external, linkedID)
	assertExternalContext(t, r, linkedID)
	assertExternalTaskUnchanged(t, r, external, linkedID)
	assertExternalFollowUpBlocked(t, r, external.ID)
}

func TestChatOnExternalRejectsOwnerAndArchivedTaskAndNoModel(t *testing.T) {
	tests := []rejectedChatCase{
		{
			name: "owner", user: "intruder", want: domain.ErrNotFound,
		},
		{
			name: "archived", user: "u1", want: domain.ErrWrongState,
			prep: func(r *rig, task domain.Task) {
				now := time.Now()
				task.ArchivedAt = &now
				_ = r.mem.Create(context.Background(), task)
			},
		},
		{
			name: "no model", user: "u1", want: domain.ErrNoModel,
			prep: func(r *rig, _ domain.Task) { r.models.none = true },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertRejectedChatCase(t, test)
		})
	}
}

type rejectedChatCase struct {
	prep func(*rig, domain.Task)
	want error
	name string
	user string
}

func assertRejectedChatCase(t *testing.T, test rejectedChatCase) {
	t.Helper()
	r := newRig(nil)
	r.service = newServiceWithLinkedTasks(r)
	task := externalTask()
	if test.name == "archived" {
		task.ArchivedAt = nil
	}
	if err := r.mem.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if test.prep != nil {
		test.prep(r, task)
	}
	err := rChatOnExternal(r, test.user, task.ID)
	if !errors.Is(err, test.want) {
		t.Fatalf("got %v, want %v", err, test.want)
	}
	if len(r.disp.enqueued) != 0 {
		t.Fatalf("rejected chat enqueued work: %v", r.disp.enqueued)
	}
}

func rChatOnExternal(r *rig, user, taskID string) error {
	_, err := r.service.ChatOnExternal(context.Background(), user, taskID, "Please help")
	if err != nil {
		return fmt.Errorf("chat on external task: %w", err)
	}
	return nil
}

func TestChatOnExternalReusesLinkedTaskWithNormalFollowUpGuards(t *testing.T) {
	r := newRig(nil)
	r.service = newServiceWithLinkedTasks(r)
	external := externalTask()
	_ = r.mem.Create(context.Background(), external)
	linked := domain.Task{
		ID: "linked", UserID: "u1", ProjectID: "proj", Status: domain.StatusWorking,
		OriginTaskID: external.ID,
	}
	_ = r.mem.Create(context.Background(), linked)
	external.LinkedTaskID = linked.ID
	_ = r.mem.Create(context.Background(), external)
	_, err := r.service.ChatOnExternal(context.Background(), "u1", external.ID, "Again")
	if !errors.Is(err, domain.ErrBusy) {
		t.Fatalf("running linked chat should use FollowUp busy guard, got %v", err)
	}
	_ = r.mem.SetStatus(context.Background(), linked.ID, domain.StatusFailed, time.Now())
	if _, err := r.service.ChatOnExternal(context.Background(), "u1", external.ID, "Continue"); err != nil {
		t.Fatalf("failed linked chat should follow normal continuation rules: %v", err)
	}
	_ = r.mem.SetStatus(context.Background(), linked.ID, domain.StatusDone, time.Now())
	if _, err := r.service.ChatOnExternal(context.Background(), "u1", external.ID, "One last question"); err != nil {
		t.Fatalf("done linked chat should follow normal continuation rules: %v", err)
	}
}

func TestChatOnExternalReplacesArchivedLinkedConversation(t *testing.T) {
	r := newRig(nil)
	r.service = newServiceWithLinkedTasks(r)
	external := externalTask()
	_ = r.mem.Create(context.Background(), external)
	archivedAt := time.Now()
	old := domain.Task{
		ID: "old-linked", UserID: "u1", ProjectID: "proj", OriginTaskID: external.ID,
		Status: domain.StatusDone, ArchivedAt: &archivedAt,
	}
	_ = r.mem.Create(context.Background(), old)
	external.LinkedTaskID = old.ID
	_ = r.mem.Create(context.Background(), external)
	newID, err := r.service.ChatOnExternal(context.Background(), "u1", external.ID, "Start fresh")
	if err != nil {
		t.Fatalf("archived linked chat should be replaced: %v", err)
	}
	if newID == old.ID {
		t.Fatalf("archived linked task was reused: %q", newID)
	}
	updated, err := r.mem.Find(context.Background(), "u1", external.ID)
	if err != nil || updated.LinkedTaskID != newID {
		t.Fatalf("external task link = %q, error = %v; want %q", updated.LinkedTaskID, err, newID)
	}
	if len(r.disp.enqueued) != 1 || r.disp.enqueued[0] != newID {
		t.Fatalf("fresh linked task was not enqueued once: %v", r.disp.enqueued)
	}
}

func externalTask() domain.Task {
	return domain.Task{
		ID: "external", UserID: "u1", ProjectID: "proj", Source: "ChatGPT", WorkID: "work",
		Status: domain.StatusWorking, Plan: []domain.PlanStep{{Title: "Search", State: domain.PlanDone}},
		Research: "Research notes", PRD: "Requirements notes",
	}
}

func newServiceWithLinkedTasks(r *rig) *application.Service {
	return application.NewService(application.Deps{
		Tasks: r.mem, LinkedTasks: r.mem, Events: r.mem, Transcript: r.mem,
		Models: r.models, Projects: fakeProjects{}, IDs: r.mem, Now: time.Now,
	}, r.disp)
}
