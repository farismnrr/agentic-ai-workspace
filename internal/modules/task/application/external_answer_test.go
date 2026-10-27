package application

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestAnswerRoutesExternalTaskToOwnedQuestionStore(t *testing.T) {
	store := newWorkFakeStore()
	store.tasks["task-external"] = domain.Task{
		ID: "task-external", UserID: "u1", ProjectID: "proj", Source: "ChatGPT", WorkID: "work-1",
	}
	answers := &externalAnswerFake{
		userID: "u1", projectID: "proj", taskID: "task-external", workID: "work-1", eventID: "event-1",
		event: domain.Event{Kind: domain.EventQuestion, Questions: []domain.Question{
			{Text: "Ready?", Type: domain.QuestionText},
		}},
	}
	notify := &answerNotifier{}
	service := NewService(Deps{Tasks: workTaskFake{store: store}, Notify: notify}, nil)
	service.UseExternalQuestionStore(answers)
	if err := service.Answer(context.Background(), "u1", "task-external", "event-1",
		Answer{Answers: [][]string{{" Ready "}}}); err != nil {
		t.Fatal(err)
	}
	if answers.calls != 1 || answers.summary != "Ready" || answers.answers[0][0] != "Ready" ||
		notify.taskID != "task-external" {
		t.Fatalf("external answer = %+v notification=%q", answers, notify.taskID)
	}
	if err := service.Answer(context.Background(), "other", "task-external", "event-1",
		Answer{Answers: [][]string{{"No"}}}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user answer = %v", err)
	}
	if answers.calls != 1 {
		t.Fatalf("cross-user call reached question store: %d", answers.calls)
	}
	answers.projectID = "another-project"
	if err := service.Answer(context.Background(), "u1", "task-external", "event-1",
		Answer{Answers: [][]string{{"No"}}}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-project answer = %v", err)
	}
	if answers.calls != 1 {
		t.Fatalf("cross-project call reached answer store: %d", answers.calls)
	}
}

type externalAnswerFake struct {
	summary   string
	userID    string
	projectID string
	taskID    string
	workID    string
	eventID   string
	answers   [][]string
	event     domain.Event
	calls     int
}

func (f *externalAnswerFake) FindQuestion(_ context.Context, user, project, task, work, event string) (
	WorkQuestion, domain.Event, error,
) {
	if user != f.userID || project != f.projectID || task != f.taskID || work != f.workID || event != f.eventID {
		return WorkQuestion{}, domain.Event{}, domain.ErrNotFound
	}
	return WorkQuestion{ID: "q1", State: "open"}, f.event, nil
}

func (f *externalAnswerFake) AnswerExternalQuestion(_ context.Context, user, project, task, work, event string,
	answers [][]string, summary string,
) (bool, error) {
	if user != f.userID || project != f.projectID || task != f.taskID || work != f.workID || event != f.eventID {
		return false, domain.ErrNotFound
	}
	f.calls++
	f.answers, f.summary = answers, summary
	return true, nil
}

type answerNotifier struct{ taskID string }

func (n *answerNotifier) Changed(_, taskID string) { n.taskID = taskID }
func (*answerNotifier) Delta(string, string)       {}
func (*answerNotifier) FilesChanged(string)        {}
