package application

import (
	"context"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestSubmittedPlanFreezesAnsweredQuestionsAndAnswers(t *testing.T) {
	base, store, begun := begunWork(t)
	questions := &questionCountFake{workFakeStore: store, answered: []WorkQuestion{{
		State: "answered", Form: []domain.Question{{Text: "Which name?"}}, Answers: [][]string{{"North Star"}},
	}}}
	works := NewWorks(base.deps, questions, store, questions)
	plan, err := works.UpdatePlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		UpdatePlanInput{Steps: []PlanStepInput{{Title: "Build"}}}, "")
	if err != nil {
		t.Fatal(err)
	}
	submitted, err := works.SubmitPlan(context.Background(), "u1", "ChatGPT", "proj", begun.WorkID,
		SubmitPlanInput{BaseRevision: plan.Revision, Summary: "Ready", SmallTask: true})
	if err != nil || submitted.Revision.QA != `[{"question":"Which name?","answer":["North Star"]}]` {
		t.Fatalf("submitted QA snapshot = %q, %v", submitted.Revision.QA, err)
	}
	questions.answered[0].Answers = [][]string{{"Changed later"}}
	stored, found, err := store.LatestPlan(context.Background(), begun.WorkID)
	if err != nil || !found || stored.QA != submitted.Revision.QA {
		t.Fatalf("approved snapshot changed after a later answer edit: %+v, %v", stored, err)
	}
}
