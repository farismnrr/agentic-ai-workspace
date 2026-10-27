package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestRetryAndAnswerRefuseConnectedAppTasks(t *testing.T) {
	r := newRig(nil)
	ctx := context.Background()
	id, err := r.service.Record(ctx, "u1", application.RecordInput{
		ProjectID: "proj", Source: "ChatGPT", Event: domain.Event{Kind: domain.EventRequest, Text: "Do this"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = r.mem.SetStatus(ctx, id, domain.StatusFailed, time.Now())
	if err := r.service.Retry(ctx, "u1", id); !errors.Is(err, domain.ErrWrongState) || len(r.disp.enqueued) != 0 {
		t.Fatalf("external task was requeued: err=%v queue=%v", err, r.disp.enqueued)
	}
	if err := r.service.Answer(ctx, "u1", id, "missing", application.Answer{}); !errors.Is(err, domain.ErrWrongState) {
		t.Fatalf("external task accepted an internal answer: %v", err)
	}
}

func TestRecoverySkipsWorkAndLegacyCallsDoNotJoinIt(t *testing.T) {
	r := newRig(nil)
	ctx := context.Background()
	workTask := domain.Task{
		ID: "work-task", UserID: "u1", ProjectID: "proj", Source: "ChatGPT",
		WorkID: "work-id", Status: domain.StatusWorking, UpdatedAt: time.Now(), Plan: []domain.PlanStep{},
	}
	if err := r.mem.Create(ctx, workTask); err != nil {
		t.Fatal(err)
	}
	if err := r.service.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if got := r.task(t, workTask.ID).Status; got != domain.StatusWorking {
		t.Fatalf("recovery changed active work state to %s", got)
	}
	first, err := r.service.Record(ctx, "u1", application.RecordInput{
		ProjectID: "proj", Source: "ChatGPT", Event: domain.Event{Kind: domain.EventMessage, Text: "Legacy call"},
	})
	if err != nil || first == workTask.ID {
		t.Fatalf("legacy call joined an explicit work: id=%s err=%v", first, err)
	}
	second, err := r.service.Record(ctx, "u1", application.RecordInput{
		ProjectID: "proj", Source: "ChatGPT", Event: domain.Event{Kind: domain.EventMessage, Text: "Next legacy call"},
	})
	if err != nil || second != first {
		t.Fatalf("legacy 30-minute grouping changed: first=%s second=%s err=%v", first, second, err)
	}
}

type stopRecorder struct{ user, client, project, work string }

func (s *stopRecorder) Stop(_ context.Context, user, client, project, work string) error {
	s.user, s.client, s.project, s.work = user, client, project, work
	return nil
}

func TestCancelDelegatesWorkTasksToWorkStop(t *testing.T) {
	r := newRig(nil)
	stop := &stopRecorder{}
	r.service.UseWorkServices(stop, nil)
	task := domain.Task{
		ID: "work-task", UserID: "u1", ProjectID: "proj", Source: "ChatGPT",
		WorkID: "work-id", Status: domain.StatusWorking, Plan: []domain.PlanStep{},
	}
	if err := r.mem.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if err := r.service.Cancel(context.Background(), "u1", task.ID); err != nil {
		t.Fatal(err)
	}
	if stop.user != "u1" || stop.client != "ChatGPT" || stop.project != "proj" || stop.work != "work-id" {
		t.Fatalf("cancel did not target linked work: %+v", stop)
	}
}
