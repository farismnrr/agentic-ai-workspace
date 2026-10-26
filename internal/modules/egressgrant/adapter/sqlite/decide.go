package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

// attempts is how many times a decision is tried again when another decision changed the grant first.
const attempts = 5

// Decide works out the next state inside a transaction and saves it only where the row still has the state it was
// read in: the update names the old status and end, so a second decision made from the same state changes nothing
// and tries again from what the first one left. That second try is refused by the domain rules (already decided).
func (s *Store) Decide(
	ctx context.Context, id string, change func(domain.Grant) (domain.Grant, error),
) (domain.Grant, error) {
	for range attempts {
		next, retry, err := s.decideOnce(ctx, id, change)
		if !retry {
			return next, err
		}
	}

	return domain.Grant{}, application.ErrConflict
}

func (s *Store) decideOnce(
	ctx context.Context, id string, change func(domain.Grant) (domain.Grant, error),
) (next domain.Grant, retry bool, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Grant{}, false, fmt.Errorf("begin decision: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	// The first statement is a write that changes nothing: it takes SQLite's write lock before anything is read, so
	// decisions wait for each other and each one reads what the last committed. Reading first and writing later would
	// fail with a stale snapshot whenever another decision committed in between.
	locked, err := tx.ExecContext(ctx, `UPDATE egress_grants SET id = id WHERE id = ?`, id)
	if err != nil {
		return domain.Grant{}, busy(err), fmt.Errorf("lock grant: %w", err)
	}

	if n, _ := locked.RowsAffected(); n == 0 {
		return domain.Grant{}, false, application.ErrNotFound
	}

	current, err := scan(tx.QueryRowContext(ctx, `SELECT `+columns+` FROM egress_grants WHERE id = ?`, id))
	if err != nil {
		return domain.Grant{}, false, err
	}

	if err = s.taskStillOpen(ctx, tx, current); err != nil {
		return domain.Grant{}, false, err
	}

	if next, err = change(current); err != nil {
		return domain.Grant{}, false, err
	}

	result, err := tx.ExecContext(ctx, `UPDATE egress_grants SET status = ?, decided_by = ?, decided_at = ?,
		expires_at = ? WHERE id = ? AND status = ? AND expires_at = ? AND project_id = ? AND task_id = ?`,
		next.Status, next.DecidedBy, next.DecidedAt.Unix(), next.ExpiresAt.Unix(),
		id, current.Status, current.ExpiresAt.Unix(), current.ProjectID, current.TaskID)
	if err != nil {
		return domain.Grant{}, busy(err), fmt.Errorf("update grant: %w", err)
	}

	if n, _ := result.RowsAffected(); n != 1 {
		return domain.Grant{}, true, nil
	}

	if err = tx.Commit(); err != nil {
		return domain.Grant{}, busy(err), fmt.Errorf("commit decision: %w", err)
	}

	return next, false, nil
}

// busy reports a lock conflict with another writer, the one failure worth trying a decision again for.
func busy(err error) bool {
	text := err.Error()

	return strings.Contains(text, "SQLITE_BUSY") || strings.Contains(text, "database is locked")
}

// taskStillOpen prevents decisions after task closure inside the same locked transaction.
func (s *Store) taskStillOpen(ctx context.Context, tx *sql.Tx, current domain.Grant) error {
	if current.Status != domain.StatusPending {
		return nil
	}
	var open int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks t WHERE t.id = ? AND t.project_id = ? AND `+liveTask,
		current.TaskID, current.ProjectID).Scan(&open)
	if err != nil {
		return fmt.Errorf("check grant task: %w", err)
	}
	if open != 1 {
		return application.ErrScope
	}
	return nil
}
