package sqlite

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestExternalQuestionLimitsCountOpenAndLifetimeQuestions(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-question-limits", "t1", "")
	for index := 1; index <= application.MaxOpenWorkQuestions; index++ {
		askLimitQuestion(t, ctx, store, work, index)
	}
	if _, _, err := store.AskQuestion(ctx, limitQuestion(work, 6)); !errors.Is(err, domain.ErrTooMany) {
		t.Fatalf("sixth open question = %v", err)
	}
	questions, err := store.Questions(ctx, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, question := range questions {
		if _, err := store.AnswerExternalQuestion(ctx, work.UserID, work.ProjectID, work.TaskID,
			work.ID, question.EventID, [][]string{{"Answer"}}, "Answer"); err != nil {
			t.Fatal(err)
		}
	}
	for index := application.MaxOpenWorkQuestions + 1; index <= application.MaxWorkQuestions; index++ {
		created := askLimitQuestion(t, ctx, store, work, index)
		if _, err := store.AnswerExternalQuestion(ctx, work.UserID, work.ProjectID, work.TaskID,
			work.ID, created.EventID, [][]string{{"Answer"}}, "Answer"); err != nil {
			t.Fatal(err)
		}
	}
	if !totalQuestionLimitIsEnforced(ctx, store, work) {
		t.Fatal("41st total question was not refused")
	}
}

func TestQuestionSupersedesSubmittedPlanRevision(t *testing.T) {
	ctx, _, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-question-review", "t1", "")
	revision, err := store.SubmitPlan(ctx, application.PlanSubmission{
		WorkID: work.ID, UserID: work.UserID, ExpectRev: work.Rev,
		Revision: domain.PlanRevision{
			ID: "revision-1", WorkID: work.ID, TaskID: work.TaskID, Steps: []domain.PlanStep{{Title: "Build"}},
			Summary: "Ready", Digest: "digest", SmallTask: true, Status: domain.PlanRevisionSubmitted,
		},
		Event: domain.Event{
			Kind: domain.EventPlanReview, Text: "Ready for approval",
			PlanReview: &domain.PlanReview{Summary: "Ready", Status: domain.PlanRevisionSubmitted},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	question := askLimitQuestion(t, ctx, store, work, 1)
	latest, found, err := store.LatestPlan(ctx, work.ID)
	if err != nil || !found || latest.Status != domain.PlanRevisionSuperseded {
		t.Fatalf("submitted plan revision after question: %+v found=%v err=%v", latest, found, err)
	}
	event, err := store.FindEvent(ctx, work.TaskID, revision.EventID)
	if err != nil || event.PlanReview.Status != domain.PlanRevisionSuperseded {
		t.Fatalf("plan review card after question: %+v err=%v", event.PlanReview, err)
	}
	if _, err := store.AnswerExternalQuestion(ctx, work.UserID, work.ProjectID, work.TaskID,
		work.ID, question.EventID, [][]string{{"Ready"}}, "Ready"); err != nil {
		t.Fatal(err)
	}
}

func totalQuestionLimitIsEnforced(ctx context.Context, store *Store, work domain.ExternalWork) bool {
	_, _, err := store.AskQuestion(ctx, limitQuestion(work, application.MaxWorkQuestions+1))
	return errors.Is(err, domain.ErrTooMany)
}

func askLimitQuestion(t *testing.T, ctx context.Context, store *Store, work domain.ExternalWork,
	index int,
) application.WorkQuestion {
	t.Helper()
	question, replay, err := store.AskQuestion(ctx, limitQuestion(work, index))
	if err != nil || replay {
		t.Fatalf("ask #%d: replay=%v err=%v", index, replay, err)
	}
	return question
}

func limitQuestion(work domain.ExternalWork, index int) application.QuestionCreate {
	form := []domain.Question{{Text: fmt.Sprintf("Question %d?", index), Type: domain.QuestionText}}
	return application.QuestionCreate{
		Question: application.WorkQuestion{ID: fmt.Sprintf("q-%02d", index), Form: form, AskedAt: int64(index)},
		TaskID:   work.TaskID, UserID: work.UserID, ProjectID: work.ProjectID, WorkID: work.ID,
		RequestKey: fmt.Sprintf("request-%03d", index), Event: questionEvent(work.ID, form),
	}
}
