package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const maxHandoffBackoffShift = 6

func handoffBackoff(ctx context.Context, db *sql.DB, id string) time.Duration {
	var attempts int
	if err := db.QueryRowContext(ctx, `SELECT attempts FROM external_plan_handoffs WHERE id = ?`, id).
		Scan(&attempts); err != nil {
		return time.Minute
	}
	shift := min(max(attempts-1, 0), maxHandoffBackoffShift)
	return time.Duration(1<<shift) * time.Minute
}

func requireOneRow(result sql.Result, missing error, action string) error {
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count affected rows for %s: %w", action, err)
	}
	if changed != 1 {
		return fmt.Errorf("%s: %w", action, missing)
	}
	return nil
}
