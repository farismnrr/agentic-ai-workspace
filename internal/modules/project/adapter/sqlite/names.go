package sqlite

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

// ActiveNames lists every active project that a person made themselves.
func (s *Store) ActiveNames(ctx context.Context) ([]domain.Named, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM projects WHERE archived_at IS NULL AND source = ''`)
	if err != nil {
		return nil, fmt.Errorf("select names: %w", err)
	}

	defer func() { _ = rows.Close() }()

	names := []domain.Named{}

	for rows.Next() {
		var named domain.Named
		if err = rows.Scan(&named.ID, &named.Name); err != nil {
			return nil, fmt.Errorf("read name: %w", err)
		}

		names = append(names, named)
	}

	return names, rows.Err() //nolint:wrapcheck // a failed iteration is reported by the caller
}
