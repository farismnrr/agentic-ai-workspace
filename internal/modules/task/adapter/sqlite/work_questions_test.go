package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestExternalQuestionRetryOwnerBindingAndAnswerRace(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-question", "t1", "")
	form := []domain.Question{{Header: "Name", Text: "What should it be called?", Type: domain.QuestionText}}
	in := application.QuestionCreate{
		Question: application.WorkQuestion{ID: "q1", Form: form, AskedAt: time.Now().Unix()},
		TaskID:   work.TaskID, UserID: work.UserID, ProjectID: work.ProjectID, WorkID: work.ID,
		RequestKey: "question-001", Event: questionEvent(work.ID, form),
	}
	first := askQuestionAndConfirmReplay(t, ctx, store, in)
	if !questionLookupIsHidden(ctx, store, "other", work.ProjectID, work, first.EventID) {
		t.Fatal("cross-user question lookup exposed the question")
	}
	if !questionLookupIsHidden(ctx, store, work.UserID, "other-project", work, first.EventID) {
		t.Fatal("cross-project question lookup exposed the question")
	}
	if !changedQuestionRetryIsRefused(ctx, store, work, in.RequestKey, form) {
		t.Fatal("changed retry was accepted")
	}
	assertWorkQuestionState(t, ctx, store, work, 1, 0)
	answerQuestionRace(t, ctx, store, work, first.EventID)
	assertWorkQuestionState(t, ctx, store, work, 0, 1)
	replay, replayed, err := store.AskQuestion(ctx, in)
	if err != nil || !replayed || replay.ID != first.ID || replay.State != "answered" {
		t.Fatalf("answered question retry: %+v replay=%v err=%v", replay, replayed, err)
	}
	assertQuestionAnswered(t, ctx, store, db, work, first.ID, first.EventID)
}

func assertWorkQuestionState(t *testing.T, ctx context.Context, store *Store, work domain.ExternalWork,
	wantOpen, wantAnswered int,
) {
	t.Helper()
	works := application.NewWorks(application.Deps{Tasks: store, Events: store}, store, store, store)
	state, err := works.State(ctx, work.UserID, work.Client, work.ProjectID, work.ID)
	if err != nil || len(state.Questions.Open) != wantOpen || len(state.Questions.Answered) != wantAnswered {
		t.Fatalf("reloaded work questions: %+v err=%v", state.Questions, err)
	}
	if wantOpen > 0 && state.NextAction != "wait_for_answers" {
		t.Fatalf("next action while question open = %q", state.NextAction)
	}
}

func askQuestionAndConfirmReplay(t *testing.T, ctx context.Context, store *Store,
	in application.QuestionCreate,
) application.WorkQuestion {
	t.Helper()
	first, replay, err := store.AskQuestion(ctx, in)
	if err != nil || replay || first.ID != in.Question.ID || first.EventID == "" {
		t.Fatalf("first ask: %+v replay=%v err=%v", first, replay, err)
	}
	second, replay, err := store.AskQuestion(ctx, in)
	if err != nil || !replay || second.ID != first.ID || second.EventID != first.EventID {
		t.Fatalf("retry ask: %+v replay=%v err=%v", second, replay, err)
	}
	return first
}

func questionLookupIsHidden(ctx context.Context, store *Store, user, project string,
	work domain.ExternalWork, eventID string,
) bool {
	_, _, err := store.FindQuestion(ctx, user, project, work.TaskID, work.ID, eventID)
	return errors.Is(err, domain.ErrNotFound)
}

func changedQuestionRetryIsRefused(ctx context.Context, store *Store, work domain.ExternalWork,
	key string, original []domain.Question,
) bool {
	form := []domain.Question{{Text: "Different?", Type: domain.QuestionText}}
	_, _, err := store.AskQuestion(ctx, application.QuestionCreate{
		Question: application.WorkQuestion{ID: "q2", Form: form},
		TaskID:   work.TaskID, UserID: work.UserID, ProjectID: work.ProjectID, WorkID: work.ID,
		RequestKey: key, Event: questionEvent(work.ID, original),
	})
	return errors.Is(err, domain.ErrKeyReused)
}

func answerQuestionRace(t *testing.T, ctx context.Context, store *Store, work domain.ExternalWork, eventID string) {
	t.Helper()
	start := make(chan struct{})
	var group sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := store.AnswerExternalQuestion(ctx, work.UserID, work.ProjectID, work.TaskID,
				work.ID, eventID, [][]string{{"Menu"}}, "Menu")
			results <- err
		}()
	}
	close(start)
	group.Wait()
	close(results)
	success, rejected := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, domain.ErrNotWaiting) {
			rejected++
		}
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("answer race: success=%d rejected=%d", success, rejected)
	}
}

func assertQuestionAnswered(t *testing.T, ctx context.Context, store *Store, db *sql.DB,
	work domain.ExternalWork, id, eventID string,
) {
	t.Helper()
	assertStoredQuestion(t, ctx, store, db, work, id, eventID)
	assertAnsweredThreadAndLifecycle(t, ctx, store, db, work, eventID)
}

func assertStoredQuestion(t *testing.T, ctx context.Context, store *Store, db *sql.DB,
	work domain.ExternalWork, id, eventID string,
) {
	t.Helper()
	questions, err := store.Questions(ctx, work.ID)
	if err != nil || len(questions) != 1 || questions[0].ID != id || questions[0].State != "answered" ||
		questions[0].Summary != "Menu" {
		t.Fatalf("persisted question = %+v, %v", questions, err)
	}
	loaded, err := NewStore(db, &workTestIDs{}).Questions(ctx, work.ID)
	if err != nil || len(loaded) != 1 || loaded[0].EventID != eventID || loaded[0].Answers[0][0] != "Menu" {
		t.Fatalf("reloaded question = %+v, %v", loaded, err)
	}
}

func assertAnsweredThreadAndLifecycle(t *testing.T, ctx context.Context, store *Store, db *sql.DB,
	work domain.ExternalWork, eventID string,
) {
	t.Helper()
	event, err := store.FindEvent(ctx, work.TaskID, eventID)
	if err != nil || event.Answered != "Menu" {
		t.Fatalf("answered thread card = %+v, %v", event, err)
	}
	var phase, status string
	if err := db.QueryRowContext(ctx, `SELECT plan_state FROM external_works WHERE id = ?`, work.ID).
		Scan(&phase); err != nil || phase != "draft" {
		t.Fatalf("work phase = %q, %v", phase, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, work.TaskID).
		Scan(&status); err != nil || status != string(domain.StatusWorking) {
		t.Fatalf("task status = %q, %v", status, err)
	}
}

func questionEvent(workID string, form []domain.Question) domain.Event {
	return domain.Event{
		Kind: domain.EventQuestion, Text: "What should it be called?", Time: time.Now(),
		Questions: form, Work: &domain.WorkRef{ID: workID},
	}
}
