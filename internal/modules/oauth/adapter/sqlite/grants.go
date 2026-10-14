package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

const grantColumns = `id, user_id, client_id, client_name, redirect_uri, resource, scope, ` +
	`created_at, last_used_at, revoked_at`

type scanner interface {
	Scan(dest ...any) error
}

// CreateGrant stores a new grant.
func (s *Store) CreateGrant(ctx context.Context, g domain.Grant) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO oauth_grants (id, user_id, client_id, client_name, redirect_uri, resource, scope, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		g.ID, g.UserID, g.ClientID, g.ClientName, g.RedirectURI, g.Resource, domain.JoinScopes(g.Scopes),
		g.CreatedAt.Unix())
	if err != nil {
		return fmt.Errorf("insert grant: %w", err)
	}

	return nil
}

// FindGrant loads a grant, revoked or not.
func (s *Store) FindGrant(ctx context.Context, id string) (domain.Grant, error) {
	grant, err := scanGrant(s.db.QueryRowContext(ctx, `SELECT `+grantColumns+` FROM oauth_grants WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Grant{}, domain.ErrGrantNotFound
	}

	return grant, err
}

// ListGrants returns a person's active grants, newest first.
func (s *Store) ListGrants(ctx context.Context, userID string) ([]domain.Grant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+grantColumns+` FROM oauth_grants WHERE user_id = ? AND revoked_at IS NULL ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("select grants: %w", err)
	}

	defer func() { _ = rows.Close() }()

	grants := []domain.Grant{}

	for rows.Next() {
		grant, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}

		grants = append(grants, grant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read grants: %w", err)
	}

	return grants, nil
}

// RevokeGrant marks a grant revoked; every token it minted stops working.
func (s *Store) RevokeGrant(ctx context.Context, grantID string, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE oauth_grants SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, now.Unix(), grantID)
	if err != nil {
		return fmt.Errorf("revoke grant: %w", err)
	}

	return nil
}

// TouchGrant records recent use.
func (s *Store) TouchGrant(ctx context.Context, grantID string, now time.Time) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE oauth_grants SET last_used_at = ? WHERE id = ?`, now.Unix(), grantID); err != nil {
		return fmt.Errorf("touch grant: %w", err)
	}

	return nil
}

func scanGrant(row scanner) (domain.Grant, error) {
	var (
		g         domain.Grant
		scope     string
		created   int64
		used, rev sql.NullInt64
	)

	err := row.Scan(&g.ID, &g.UserID, &g.ClientID, &g.ClientName, &g.RedirectURI, &g.Resource, &scope,
		&created, &used, &rev)
	if err != nil {
		return domain.Grant{}, fmt.Errorf("scan grant: %w", err)
	}

	g.Scopes = domain.SplitScopes(scope)
	g.CreatedAt = time.Unix(created, 0)
	g.LastUsedAt = optionalTime(used)
	g.RevokedAt = optionalTime(rev)

	return g, nil
}

func optionalTime(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}

	moment := time.Unix(value.Int64, 0)

	return &moment
}
