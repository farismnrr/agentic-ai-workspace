package application

import (
	"context"
	"fmt"
)

// Checks returns the commands the AI may run in one of the person's projects.
func (s *Service) Checks(ctx context.Context, userID, id string) ([]Check, error) {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return nil, err
	}

	checks, err := s.checks.List(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list checks: %w", err)
	}

	return checks, nil
}

// SaveChecks replaces the commands the AI may run in a project.
func (s *Service) SaveChecks(ctx context.Context, userID, id string, checks []Check) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}

	if err := s.checks.Save(ctx, id, checks); err != nil {
		return fmt.Errorf("save checks: %w", err)
	}

	return nil
}
