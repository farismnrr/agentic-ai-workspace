package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
)

// AddEraser registers a feature that must clear its data before an account is deleted.
// Erasers run in the order they were added.
func (s *Service) AddEraser(e Eraser) {
	s.erasers = append(s.erasers, e)
}

// Delete removes an account and everything it owns, after checking the password.
func (s *Service) Delete(ctx context.Context, userID, password string) error {
	hash, err := s.users.PasswordHash(ctx, userID)
	if err != nil {
		return fmt.Errorf("find account: %w", err)
	}

	if !s.hasher.Verify(password, hash) {
		return domain.ErrWrongPassword
	}

	for _, eraser := range s.erasers {
		if err := eraser.Erase(ctx, userID); err != nil {
			return fmt.Errorf("erase data: %w", err)
		}
	}

	if err := s.users.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}

	return nil
}
