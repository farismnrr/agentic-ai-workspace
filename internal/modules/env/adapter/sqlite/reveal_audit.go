package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// RevealAudit stores metadata for browser reveal attempts, never secret content.
type RevealAudit struct{ db *sql.DB }

func NewRevealAudit(db *sql.DB) *RevealAudit { return &RevealAudit{db: db} }

func (a *RevealAudit) RecordReveal(
	ctx context.Context, userID, projectID, name, outcome string, at time.Time,
) error {
	_, err := a.db.ExecContext(ctx, `INSERT INTO project_env_reveal_audit
		(user_id, project_id, variable_name, outcome, created_at) VALUES (?, ?, ?, ?, ?)`,
		userID, projectID, name, outcome, at.Unix())
	if err != nil {
		return fmt.Errorf("record project env reveal metadata: %w", err)
	}
	return nil
}
