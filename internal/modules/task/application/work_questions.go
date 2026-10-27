package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	MaxOpenWorkQuestions = 5
	MaxWorkQuestions     = 40
)

// WorkQuestion is a persisted request for information from the project owner.
type WorkQuestion struct {
	ID      string            `json:"id"`
	EventID string            `json:"event_id"`
	State   string            `json:"state"`
	Summary string            `json:"summary,omitempty"`
	Answers [][]string        `json:"answers,omitempty"`
	Form    []domain.Question `json:"form"`
	AskedAt int64             `json:"asked_at"`
}

// QuestionCreate contains the already-authorized identity and the question card to persist atomically.
type QuestionCreate struct {
	Question   WorkQuestion
	TaskID     string
	UserID     string
	ProjectID  string
	WorkID     string
	RequestKey string
	Event      domain.Event
}

// QuestionStore owns durable questions and their task-thread events.
type QuestionStore interface {
	AskQuestion(context.Context, QuestionCreate) (WorkQuestion, bool, error)
	Questions(context.Context, string) ([]WorkQuestion, error)
	OpenQuestionCount(context.Context, string) (int, error)
	TotalQuestionCount(context.Context, string) (int, error)
	FindQuestion(context.Context, string, string, string, string, string) (WorkQuestion, domain.Event, error)
	AnswerExternalQuestion(context.Context, string, string, string, string, string, [][]string, string) (bool, error)
}

// FinishResearch checks saved research without changing documents or phase.
func (w *Works) FinishResearch(ctx context.Context, userID, client, projectID, workID string) (bool, []string, error) {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return false, nil, err
	}
	work, task, _, err := w.mutable(ctx, userID, client, projectID, workID)
	if err != nil {
		return false, nil, err
	}
	events, err := w.deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return false, nil, fmt.Errorf("read research history: %w", err)
	}
	ready, missing := domain.ResearchEnough(researchRounds(events, work.ID))
	return ready, missing, nil
}

// Questions returns persistent open and answered questions for a work session.
func (w *Works) Questions(ctx context.Context, workID string) ([]WorkQuestion, error) {
	if w.questions == nil {
		return []WorkQuestion{}, nil
	}
	questions, err := w.questions.Questions(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("load work questions: %w", err)
	}
	return questions, nil
}

// AskQuestion persists one form and updates the work phase and task status in the same transaction.
func (w *Works) AskQuestion(ctx context.Context, userID, client, projectID, workID, requestID string,
	questions []domain.Question,
) (WorkQuestion, error) {
	if err := w.limits.allow(userID, false, w.now()); err != nil {
		return WorkQuestion{}, err
	}
	if err := validateQuestionRequest(requestID, questions); err != nil {
		return WorkQuestion{}, err
	}
	work, _, _, err := w.mutable(ctx, userID, client, projectID, workID)
	if err != nil {
		return WorkQuestion{}, err
	}
	return w.persistQuestion(ctx, work, userID, projectID, requestID, questions)
}

func validateQuestionRequest(requestID string, questions []domain.Question) error {
	if strings.TrimSpace(requestID) == "" || !validKey(requestID) || len(questions) == 0 || len(questions) > 4 {
		return domain.ErrBadRequest
	}
	return nil
}

func (w *Works) persistQuestion(ctx context.Context, work domain.ExternalWork, userID, projectID, requestID string,
	questions []domain.Question,
) (WorkQuestion, error) {
	if w.questions == nil {
		return WorkQuestion{}, domain.ErrWrongState
	}
	questionID, err := w.deps.IDs.ID()
	if err != nil {
		return WorkQuestion{}, fmt.Errorf("new work question id: %w", err)
	}
	event := workEvent(work, domain.EventQuestion, questionTitle(questions), "", w.now())
	event.Questions = questions
	event.Options = nil
	created := QuestionCreate{
		Question: WorkQuestion{ID: questionID, Form: questions, State: "open", AskedAt: w.now().Unix()},
		TaskID:   work.TaskID, UserID: userID, ProjectID: projectID, WorkID: work.ID,
		RequestKey: requestID, Event: event,
	}
	persisted, replay, err := w.questions.AskQuestion(ctx, created)
	if err != nil {
		return WorkQuestion{}, fmt.Errorf("save work question: %w", err)
	}
	if !replay {
		w.notify(ctx, domain.Task{ID: work.TaskID, UserID: userID})
	}
	return persisted, nil
}

func questionTitle(questions []domain.Question) string {
	if len(questions) == 1 {
		return strings.TrimSpace(questions[0].Text)
	}
	return fmt.Sprintf("The assistant has %d questions", len(questions))
}
