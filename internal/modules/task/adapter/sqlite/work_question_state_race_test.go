package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestQuestionStateUpdatesDoNotUndoPlanApproval(t *testing.T) {
	ctx, db, store := newWorkStore(t)
	work := createTestWork(t, ctx, store, "w-question-race", "t-question-race", "")
	revision := submitStorePlan(t, ctx, store, work)
	input := approvalWrite(work, revision.RevNo, time.Unix(20, 0))
	if _, err := store.DecidePlan(ctx, input); err != nil {
		t.Fatal(err)
	}

	question := application.QuestionCreate{WorkID: work.ID, UserID: work.UserID, TaskID: work.TaskID}
	if err := applyQuestionWorkUpdate(ctx, db, func(tx *sql.Tx) error {
		return advanceQuestionWork(ctx, tx, question)
	}); !errors.Is(err, domain.ErrWrongState) {
		t.Fatalf("advance approved question work = %v, want wrong state", err)
	}
	if err := applyQuestionWorkUpdate(ctx, db, func(tx *sql.Tx) error {
		_, err := resumeWhenQuestionsClear(ctx, tx, work.ID, work.TaskID)
		return err
	}); !errors.Is(err, domain.ErrWrongState) {
		t.Fatalf("resume approved question work = %v, want wrong state", err)
	}

	var workState, planState, taskStatus string
	if err := db.QueryRowContext(ctx, `SELECT state, plan_state FROM external_works WHERE id = ?`, work.ID).
		Scan(&workState, &planState); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, work.TaskID).Scan(&taskStatus); err != nil {
		t.Fatal(err)
	}
	if workState != "completed" || planState != "approved" || taskStatus != "done" {
		t.Fatalf("question update changed approval: work=%s/%s task=%s", workState, planState, taskStatus)
	}
}

func applyQuestionWorkUpdate(ctx context.Context, db *sql.DB, update func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin question state update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	return update(tx)
}
