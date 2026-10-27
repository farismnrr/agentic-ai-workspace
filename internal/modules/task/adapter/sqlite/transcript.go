package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// AppendBlock adds one block to the orchestrator's model-facing conversation.
func (s *Store) AppendBlock(ctx context.Context, taskID string, role domain.Role, block domain.Block) error {
	return s.appendIn(ctx, taskID, "", role, block)
}

// LoadTranscript returns the orchestrator's conversation, joining consecutive blocks of one role into messages.
func (s *Store) LoadTranscript(ctx context.Context, taskID string) ([]domain.Message, error) {
	return s.loadIn(ctx, taskID, "")
}

func (s *Store) appendIn(ctx context.Context, taskID, scope string, role domain.Role, block domain.Block) error {
	return appendTranscript(ctx, s.db, taskID, role, block, scope)
}

func appendTranscript(
	ctx context.Context,
	exec taskExecutor,
	taskID string,
	role domain.Role,
	block domain.Block,
	scope string,
) error {
	encoded, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("encode block: %w", err)
	}

	_, err = exec.ExecContext(ctx,
		`INSERT INTO task_transcript (task_id, seq, role, block, scope)
		 VALUES (?, (SELECT COALESCE(MAX(seq), 0) + 1 FROM task_transcript WHERE task_id = ?), ?, ?, ?)`,
		taskID, taskID, string(role), string(encoded), scope)
	if err != nil {
		return fmt.Errorf("insert block: %w", err)
	}

	return nil
}

func (s *Store) loadIn(ctx context.Context, taskID, scope string) ([]domain.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT role, block FROM task_transcript WHERE task_id = ? AND scope = ? ORDER BY seq`, taskID, scope)
	if err != nil {
		return nil, fmt.Errorf("select transcript: %w", err)
	}

	defer func() { _ = rows.Close() }()

	messages := []domain.Message{}

	for rows.Next() {
		var role, encoded string
		if err := rows.Scan(&role, &encoded); err != nil {
			return nil, fmt.Errorf("scan block: %w", err)
		}

		var block domain.Block
		if err := json.Unmarshal([]byte(encoded), &block); err != nil {
			return nil, fmt.Errorf("decode block: %w", err)
		}

		if n := len(messages); n > 0 && messages[n-1].Role == domain.Role(role) {
			messages[n-1].Blocks = append(messages[n-1].Blocks, block)
		} else {
			messages = append(messages, domain.Message{Role: domain.Role(role), Blocks: []domain.Block{block}})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read transcript: %w", err)
	}

	return messages, nil
}

// AddUsage records the tokens of one model call.
func (s *Store) AddUsage(ctx context.Context, r application.UsageRecord) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO task_usage (task_id, user_id, provider_id, model, input_tokens, output_tokens, at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.TaskID, r.UserID, r.ProviderID, r.Model, r.Usage.Input, r.Usage.Output, r.At.Unix())
	if err != nil {
		return fmt.Errorf("insert usage: %w", err)
	}

	return nil
}

// UsageSummary sums a person's usage per model since a date.
func (s *Store) UsageSummary(ctx context.Context, userID string, since time.Time) ([]application.UsageRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT provider_id, model, COUNT(DISTINCT task_id), SUM(input_tokens), SUM(output_tokens)
		 FROM task_usage WHERE user_id = ? AND at >= ? GROUP BY provider_id, model ORDER BY model`,
		userID, since.Unix())
	if err != nil {
		return nil, fmt.Errorf("select usage: %w", err)
	}

	defer func() { _ = rows.Close() }()

	summary := []application.UsageRow{}

	for rows.Next() {
		var row application.UsageRow
		if err := rows.Scan(&row.ProviderID, &row.Model, &row.Tasks, &row.Input, &row.Output); err != nil {
			return nil, fmt.Errorf("scan usage: %w", err)
		}

		summary = append(summary, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read usage: %w", err)
	}

	return summary, nil
}
