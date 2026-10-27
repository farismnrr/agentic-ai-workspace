package application

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestSubmitPlanWaitsForExternalQuestionAnswer(t *testing.T) {
	base, store, begun := begunWork(t)
	questions := &questionCountFake{workFakeStore: store, open: 1}
	works := NewWorks(base.deps, questions, questions, questions)
	plan, err := works.UpdatePlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		UpdatePlanInput{Steps: []PlanStepInput{{Title: "Small fix"}}}, "")
	if err != nil {
		t.Fatal(err)
	}
	input := SubmitPlanInput{BaseRevision: plan.Revision, Summary: "A one line fix", SmallTask: true}
	if !submitPlanErrorIs(t, works, begun.WorkID, input, domain.ErrOpenQuestions) {
		t.Fatal("plan submission did not wait for the open question")
	}
	questions.open = 0
	if _, err := works.SubmitPlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID, input); err != nil {
		t.Fatalf("submitted after questions were answered: %v", err)
	}
}

func submitPlanErrorIs(t *testing.T, works *Works, workID string, input SubmitPlanInput, want error) bool {
	t.Helper()
	_, err := works.SubmitPlan(context.Background(), "u1", "ChatGPT", "proj", workID, input)
	return errors.Is(err, want)
}

type questionCountFake struct {
	*workFakeStore
	answered []WorkQuestion
	open     int
	total    int
}

func (f *questionCountFake) AskQuestion(_ context.Context, in QuestionCreate) (WorkQuestion, bool, error) {
	return in.Question, false, nil
}

func (f *questionCountFake) Questions(context.Context, string) ([]WorkQuestion, error) {
	return f.answered, nil
}

func (f *questionCountFake) OpenQuestionCount(context.Context, string) (int, error) {
	return f.open, nil
}

func (f *questionCountFake) TotalQuestionCount(context.Context, string) (int, error) {
	return f.total, nil
}

func (f *questionCountFake) FindQuestion(context.Context, string, string, string, string, string) (
	WorkQuestion, domain.Event, error,
) {
	return WorkQuestion{}, domain.Event{}, domain.ErrNotFound
}

func (f *questionCountFake) AnswerExternalQuestion(context.Context, string, string, string, string, string,
	[][]string, string,
) (bool, error) {
	return false, nil
}
