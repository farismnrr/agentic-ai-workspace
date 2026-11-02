package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// ApproveExternalPlanEvent makes the event answer and task transition one
// transaction. Only the latest submitted Orchestrator PRD question is eligible.
// A retry with the same hashed idempotency key is a read-only success.
func (s *Store) ApproveExternalPlanEvent(
	ctx context.Context, userID, client, taskID, eventID, requestHash string, at time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start external plan approval: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var status, source, prd, plan string
	var archived sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT status, source, COALESCE(prd,''), COALESCE(plan,'[]'), archived_at
        FROM tasks WHERE id = ? AND user_id = ?`, taskID, userID).
		Scan(&status, &source, &prd, &plan, &archived)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find external approval task: %w", err)
	}
	if archived.Valid || source != "orchestrator:"+client || prd == "" || plan == "" || plan == "[]" {
		return domain.ErrWrongState
	}

	var latestPRD, latestRevision int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0) FROM task_events
        WHERE task_id = ? AND kind = 'prd'`, taskID).Scan(&latestPRD); err != nil {
		return fmt.Errorf("read latest PRD revision: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0) FROM task_events
        WHERE task_id = ? AND kind IN ('prd','plan')`, taskID).Scan(&latestRevision); err != nil {
		return fmt.Errorf("read latest plan revision: %w", err)
	}
	if latestPRD == 0 {
		return domain.ErrWrongState
	}

	event, err := scanEvent(tx.QueryRowContext(ctx, `SELECT `+eventColumns+`
        FROM task_events WHERE task_id = ? AND id = ?`, taskID, eventID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("read submitted approval event: %w", err)
	}

	// Match the exact question created by SubmitExternalPlan, not another
	// arbitrary question in the same task's thread.
	const question = "The external orchestrator submitted the PRD. Start building?"
	const answer = "Start building"
	marker := "MCP approval request sha256=" + requestHash
	if event.Kind != domain.EventQuestion || event.Agent != source || event.Text != question ||
		len(event.Questions) != 1 || len(event.Questions[0].Options) == 0 ||
		event.Questions[0].Options[0].Label != answer ||
		event.Seq <= latestRevision {
		return domain.ErrWrongState
	}
	if event.Answered == answer && event.Technical == marker && status == "working" {
		return nil // Exactly the same event and idempotency key; no extra audit write.
	}
	if event.Answered != "" || status != "needs-input" {
		return domain.ErrWrongState
	}

	event.Answered = answer
	event.Technical = marker // Hashed idempotency key, never a raw secret.
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode approved event: %w", err)
	}
	// CAS also rechecks current plan revision and question state at write time.
	updated, err := tx.ExecContext(ctx, `UPDATE task_events SET body = ?, version =
        (SELECT COALESCE(MAX(version),0)+1 FROM task_events WHERE task_id = ?)
        WHERE task_id = ? AND id = ? AND version = ? AND kind = 'question'
        AND COALESCE(json_extract(body,'$.answered'),'') = ''
        AND seq > (SELECT COALESCE(MAX(seq),0) FROM task_events
                   WHERE task_id = ? AND kind IN ('prd','plan'))
        AND EXISTS (SELECT 1 FROM tasks WHERE id = ? AND user_id = ?
                    AND source = ? AND status = 'needs-input' AND archived_at IS NULL)`,
		string(body), taskID, taskID, eventID, event.Version, taskID,
		taskID, userID, source)
	if err != nil {
		return fmt.Errorf("record external approval: %w", err)
	}
	count, err := updated.RowsAffected()
	if err != nil {
		return fmt.Errorf("count approval changes: %w", err)
	}
	if count != 1 {
		return domain.ErrWrongState
	}

	changed, err := tx.ExecContext(ctx, `UPDATE tasks SET status = 'working', updated_at = ?
        WHERE id = ? AND user_id = ? AND source = ? AND status = 'needs-input'
        AND archived_at IS NULL`, at.Unix(), taskID, userID, source)
	if err != nil {
		return fmt.Errorf("resume approved external task: %w", err)
	}
	count, err = changed.RowsAffected()
	if err != nil || count != 1 {
		return domain.ErrWrongState
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit external approval: %w", err)
	}
	return nil
}
