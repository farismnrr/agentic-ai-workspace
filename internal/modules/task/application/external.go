package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// sessionWindow is how long a connected app can stay quiet before its next call starts a new thread.
const sessionWindow = 30 * time.Minute

const maxRecordedText = 4000

// RecordInput is one thing that happened in a connected app's conversation.
type RecordInput struct {
	ProjectID string
	// Source names the connected app, such as ChatGPT.
	Source string
	// Event is a request (what the person asked), a message (what the app answered) or an action.
	Event domain.Event
}

// Record puts a step of a connected app's conversation into the app's current thread, starting a
// thread when there is none or the last one went quiet. The app is its own model, so nothing is queued
// and no agent runs: the thread is only a record. It returns the id of the thread.
func (s *Service) Record(ctx context.Context, userID string, in RecordInput) (string, error) {
	s.recording.Lock()
	defer s.recording.Unlock()

	task, err := s.session(ctx, userID, in)
	if err != nil {
		return "", err
	}

	in.Event.Time = s.deps.Now()
	in.Event.Text = truncate(in.Event.Text, maxRecordedText)

	if _, err := s.deps.Events.AppendEvent(ctx, task.ID, in.Event); err != nil {
		return "", fmt.Errorf("record step: %w", err)
	}

	return task.ID, nil
}

// session finds the app's thread that is still current, or starts one.
func (s *Service) session(ctx context.Context, userID string, in RecordInput) (domain.Task, error) {
	latest, err := s.deps.Tasks.List(ctx, userID, in.ProjectID, 1)
	if err != nil {
		return domain.Task{}, fmt.Errorf("find thread: %w", err)
	}

	now := s.deps.Now()
	if len(latest) == 1 && latest[0].Source == in.Source && now.Sub(latest[0].UpdatedAt) < sessionWindow {
		return latest[0], nil
	}

	id, err := s.deps.IDs.ID()
	if err != nil {
		return domain.Task{}, fmt.Errorf("new task id: %w", err)
	}

	// Done from the start: nothing is running, and a restart must not queue it again.
	task := domain.Task{
		ID: id, ProjectID: in.ProjectID, UserID: userID, Title: externalTitle(in), Source: in.Source,
		Status: domain.StatusDone, CreatedAt: now, UpdatedAt: now, Plan: []domain.PlanStep{},
	}
	if err := s.deps.Tasks.Create(ctx, task); err != nil {
		return domain.Task{}, fmt.Errorf("save task: %w", err)
	}

	return task, nil
}

func externalTitle(in RecordInput) string {
	if in.Event.Kind == domain.EventRequest && in.Event.Text != "" {
		return titleOf(in.Event.Text)
	}

	return "Work done by " + in.Source
}

// Begin writes a step that is under way, such as a command that is still running, and shows the thread as working
// until End. It returns the id of the thread.
func (s *Service) Begin(ctx context.Context, userID string, in RecordInput) (string, error) {
	id, err := s.Record(ctx, userID, in)
	if err != nil {
		return "", err
	}

	s.recording.Lock()
	s.calls[id]++
	first := s.calls[id] == 1
	s.recording.Unlock()

	if first {
		_ = s.deps.Tasks.SetStatus(ctx, id, domain.StatusWorking, s.deps.Now())
	}

	return id, nil
}

// End says one call of the thread is over; the thread is done again when none is left.
func (s *Service) End(ctx context.Context, taskID string) {
	s.recording.Lock()
	s.calls[taskID]--
	last := s.calls[taskID] <= 0

	if last {
		delete(s.calls, taskID)
	}

	s.recording.Unlock()

	if last {
		_ = s.deps.Tasks.SetStatus(ctx, taskID, domain.StatusDone, s.deps.Now())
	}
}
