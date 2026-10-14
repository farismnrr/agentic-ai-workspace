package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/domain"
)

// Store persists chats and messages in SQLite.
type Store struct {
	db  *sql.DB
	ids application.IDs
}

// NewStore wraps a database and id generator.
func NewStore(db *sql.DB, ids application.IDs) *Store {
	return &Store{db: db, ids: ids}
}

// FindOrCreate retrieves the active chat for a project or creates one.
func (s *Store) FindOrCreate(ctx context.Context, userID, projectID string) (domain.Chat, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, user_id, title, created_at, updated_at `+
			`FROM project_chats WHERE user_id = ? AND project_id = ? ORDER BY updated_at DESC LIMIT 1`,
		userID, projectID)

	var c domain.Chat
	var createdAt, updatedAt int64
	err := row.Scan(&c.ID, &c.ProjectID, &c.UserID, &c.Title, &createdAt, &updatedAt)
	if err == nil {
		c.CreatedAt = time.Unix(createdAt, 0).UTC()
		c.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		return c, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Chat{}, fmt.Errorf("select chat: %w", err)
	}

	id, err := s.ids.ID()
	if err != nil {
		return domain.Chat{}, fmt.Errorf("generate chat id: %w", err)
	}

	now := time.Now().UTC()
	c = domain.Chat{
		ID: id, ProjectID: projectID, UserID: userID, Title: "Project Chat",
		CreatedAt: now, UpdatedAt: now,
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO project_chats (id, project_id, user_id, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.ProjectID, c.UserID, c.Title, now.Unix(), now.Unix())
	if err != nil {
		return domain.Chat{}, fmt.Errorf("insert chat: %w", err)
	}

	return c, nil
}

// Get loads a chat by id and checks ownership.
func (s *Store) Get(ctx context.Context, userID, chatID string) (domain.Chat, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, user_id, title, created_at, updated_at `+
			`FROM project_chats WHERE user_id = ? AND id = ?`,
		userID, chatID)

	var c domain.Chat
	var createdAt, updatedAt int64
	err := row.Scan(&c.ID, &c.ProjectID, &c.UserID, &c.Title, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Chat{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Chat{}, fmt.Errorf("select chat: %w", err)
	}

	c.CreatedAt = time.Unix(createdAt, 0).UTC()
	c.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return c, nil
}

// Messages returns all messages in a chat ordered by timestamp.
func (s *Store) Messages(ctx context.Context, chatID string) ([]domain.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, chat_id, role, content, created_at `+
			`FROM chat_messages WHERE chat_id = ? ORDER BY created_at ASC`,
		chatID)
	if err != nil {
		return nil, fmt.Errorf("select messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var msgs []domain.Message
	for rows.Next() {
		var m domain.Message
		var role string
		var createdAt int64
		if err := rows.Scan(&m.ID, &m.ChatID, &role, &m.Content, &createdAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.Role = domain.Role(role)
		m.CreatedAt = time.Unix(createdAt, 0).UTC()
		msgs = append(msgs, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read messages: %w", err)
	}

	return msgs, nil
}

// Append inserts a new message into the chat.
func (s *Store) Append(ctx context.Context, msg domain.Message) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO chat_messages (id, chat_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)`,
		msg.ID, msg.ChatID, string(msg.Role), msg.Content, msg.CreatedAt.Unix())
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}

// Clear deletes all messages in a chat.
func (s *Store) Clear(ctx context.Context, chatID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM chat_messages WHERE chat_id = ?`, chatID)
	if err != nil {
		return fmt.Errorf("clear messages: %w", err)
	}
	return nil
}

// Touch updates the updated_at timestamp on a chat.
func (s *Store) Touch(ctx context.Context, chatID string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE project_chats SET updated_at = ? WHERE id = ?`, at.Unix(), chatID)
	if err != nil {
		return fmt.Errorf("touch chat: %w", err)
	}
	return nil
}
