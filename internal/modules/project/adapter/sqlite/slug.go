package sqlite

import (
	"context"
	"fmt"
)

// SlugTaken says whether any project, archived ones too, already has the project ID.
func (s *Store) SlugTaken(ctx context.Context, slug string) (bool, error) {
	var taken bool

	err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM projects WHERE slug = ?)`, slug).Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("select project id: %w", err)
	}

	return taken, nil
}
