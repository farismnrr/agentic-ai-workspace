// Package sqlite keeps which of a person's own plans they switched off.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Switches persists the on and off switch of each plan.
type Switches struct{ db *sql.DB }

// NewSwitches wraps an open database.
func NewSwitches(db *sql.DB) *Switches { return &Switches{db: db} }

// Enabled says whether the plan is on; a plan nobody switched is on.
func (s *Switches) Enabled(ctx context.Context, userID, program string) (bool, error) {
	var enabled bool

	err := s.db.QueryRowContext(ctx,
		`SELECT enabled FROM plan_switches WHERE user_id = ? AND program = ?`, userID, program).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("read plan switch: %w", err)
	}

	return enabled, nil
}

// Set switches the plan on or off.
func (s *Switches) Set(ctx context.Context, userID, program string, enabled bool) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO plan_switches (user_id, program, enabled) VALUES (?, ?, ?)
		 ON CONFLICT (user_id, program) DO UPDATE SET enabled = excluded.enabled`, userID, program, enabled)
	if err != nil {
		return fmt.Errorf("save plan switch: %w", err)
	}

	return nil
}
