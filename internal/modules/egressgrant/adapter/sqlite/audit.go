package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Audit appends expiration and enforcement incidents to the existing activity log.
// Request and decision activity are written by SQLite triggers in the same transaction.
type Audit struct{ db *sql.DB }

func NewAudit(db *sql.DB) *Audit { return &Audit{db: db} }

// Expired records each effective expiry once even across process restarts.
// One grant-enforcing instance holds the global lock, so only one sweep writes at once.
func (a *Audit) Expired(ctx context.Context, now time.Time) (int64, error) {
	result, err := a.db.ExecContext(ctx, `INSERT INTO activity
  (id, user_id, space, client, tool, summary, outcome, detail, at, project_id, task_id)
  SELECT lower(hex(randomblob(16))), p.user_id, 'coding', g.id, 'egress_expired',
   'HTTPS request expired: ' || g.host, 'ok', '', ?, g.project_id, g.task_id
  FROM egress_grants g JOIN projects p ON p.id = g.project_id
  WHERE g.status IN ('pending','approved') AND g.expires_at <= ?
   AND NOT EXISTS (SELECT 1 FROM activity a WHERE a.client=g.id AND a.tool='egress_expired')`,
		now.Unix(), now.Unix())
	if err != nil {
		return 0, fmt.Errorf("audit grant expiration: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count expiration records: %w", err)
	}
	return n, nil
}

// EnforcementFailure records an incident without revealing grant reason, HTTP payload or credentials.
func (a *Audit) EnforcementFailure(ctx context.Context, id string) error {
	_, err := a.db.ExecContext(ctx, `INSERT INTO activity
 (id, user_id, space, client, tool, summary, outcome, detail, at, project_id, task_id)
 SELECT lower(hex(randomblob(16))), p.user_id, 'coding', g.id, 'egress_enforcement_failed',
 'HTTPS access enforcement failed: ' || g.host, 'failed', '', ?, g.project_id, g.task_id
 FROM egress_grants g JOIN projects p ON p.id=g.project_id WHERE g.id=?`,
		time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("audit enforcement failure: %w", err)
	}
	return nil
}
