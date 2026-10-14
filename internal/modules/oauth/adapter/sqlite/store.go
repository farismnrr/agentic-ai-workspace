package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// Store persists OAuth clients, requests, grants and credentials in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore wraps an open database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// SaveClient inserts or replaces a client (CIMD clients are refreshed on re-fetch).
func (s *Store) SaveClient(ctx context.Context, client domain.Client) error {
	uris, err := json.Marshal(client.RedirectURIs)
	if err != nil {
		return fmt.Errorf("encode redirect uris: %w", err)
	}

	authMethod := client.AuthMethod
	if authMethod == "" {
		authMethod = domain.AuthMethodNone
	}

	var secretHash sql.NullString
	if client.SecretHash != "" {
		secretHash = sql.NullString{String: client.SecretHash, Valid: true}
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO oauth_clients (id, name, redirect_uris, source, auth_method, secret_hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (id) DO UPDATE SET name = excluded.name, redirect_uris = excluded.redirect_uris,
		 auth_method = excluded.auth_method, secret_hash = excluded.secret_hash`,
		client.ID, client.Name, string(uris), string(client.Source), authMethod, secretHash, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("save client: %w", err)
	}

	return nil
}

// FindClient loads a client by id.
func (s *Store) FindClient(ctx context.Context, id string) (domain.Client, error) {
	var (
		client     domain.Client
		uris       string
		source     string
		authMethod string
		secretHash sql.NullString
	)

	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, redirect_uris, source, auth_method, secret_hash FROM oauth_clients WHERE id = ?`, id).
		Scan(&client.ID, &client.Name, &uris, &source, &authMethod, &secretHash)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Client{}, domain.ErrClientNotFound
	}

	if err != nil {
		return domain.Client{}, fmt.Errorf("select client: %w", err)
	}

	if err := json.Unmarshal([]byte(uris), &client.RedirectURIs); err != nil {
		return domain.Client{}, fmt.Errorf("decode redirect uris: %w", err)
	}

	client.Source = domain.ClientSource(source)
	client.AuthMethod = authMethod
	if secretHash.Valid {
		client.SecretHash = secretHash.String
	}

	return client, nil
}
