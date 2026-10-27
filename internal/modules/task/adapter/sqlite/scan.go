package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func collectTasks(rows *sql.Rows) ([]domain.Task, error) {
	defer func() { _ = rows.Close() }()

	tasks := []domain.Task{}

	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tasks: %w", err)
	}

	return tasks, nil
}

func scanTask(row scanner) (domain.Task, error) {
	var (
		t                domain.Task
		status, plan     string
		effort           string
		proof            sql.NullString
		pinned           sql.NullInt64
		archived         sql.NullInt64
		linkedTask       sql.NullString
		originTask       sql.NullString
		created, updated int64
	)

	err := row.Scan(&t.ID, &t.ProjectID, &t.UserID, &t.Title, &t.Summary, &status, &t.ProviderID, &t.Model,
		&plan, &proof, &created, &updated, &t.Source, &t.ContextUsed, &t.ContextWindow, &effort, &t.Team, &t.PRD, &t.Research,
		&t.PlanFirst, &t.FollowSettings, &pinned, &archived, &t.WorkID, &linkedTask, &originTask)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}

	t.Status, t.Effort = domain.Status(status), domain.Effort(effort)
	t.LinkedTaskID, t.OriginTaskID = linkedTask.String, originTask.String
	t.CreatedAt, t.UpdatedAt = time.Unix(created, 0).UTC(), time.Unix(updated, 0).UTC()
	if pinned.Valid {
		at := time.Unix(pinned.Int64, 0).UTC()
		t.PinnedAt = &at
	}
	if archived.Valid {
		at := time.Unix(archived.Int64, 0).UTC()
		t.ArchivedAt = &at
	}

	if err := json.Unmarshal([]byte(plan), &t.Plan); err != nil {
		return domain.Task{}, fmt.Errorf("decode plan: %w", err)
	}

	if proof.Valid && proof.String != "null" {
		t.Proof = &domain.Proof{}
		if err := json.Unmarshal([]byte(proof.String), t.Proof); err != nil {
			return domain.Task{}, fmt.Errorf("decode proof: %w", err)
		}
	}

	return t, nil
}
