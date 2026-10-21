package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	archiveLimit = 10_000
)

// BatchResult reports one task's outcome in a batch action.
type BatchResult struct {
	ID    string `json:"id"`
	Error string `json:"error,omitempty"`
	OK    bool   `json:"ok"`
}

// Archived lists a generous page of the person's archived tasks.
func (s *Service) Archived(ctx context.Context, userID string) ([]domain.Task, error) {
	tasks, err := s.deps.Tasks.Archived(ctx, userID, archiveLimit)
	if err != nil {
		return nil, fmt.Errorf("list archived tasks: %w", err)
	}
	for i := range tasks {
		tasks[i] = s.withWindow(ctx, tasks[i])
	}
	return tasks, nil
}

// ArchiveTasks stops active tasks and archives every owned id independently.
func (s *Service) ArchiveTasks(ctx context.Context, userID string, ids []string) []BatchResult {
	ids = uniqueIDs(ids)
	results := make([]BatchResult, 0, len(ids))
	for _, id := range ids {
		result := BatchResult{ID: id, OK: true}
		task, err := s.deps.Tasks.Find(ctx, userID, id)
		if err == nil {
			if task.Status.Running() {
				s.stopTask(ctx, task)
			}
			err = s.deps.Tasks.Archive(ctx, userID, id, s.deps.Now())
		}
		result.Error = batchError(err)
		result.OK = result.Error == ""
		results = append(results, result)
	}
	return results
}

// RestoreTasks restores every owned id independently and preserves its pin.
func (s *Service) RestoreTasks(ctx context.Context, userID string, ids []string) []BatchResult {
	ids = uniqueIDs(ids)
	results := make([]BatchResult, 0, len(ids))
	for _, id := range ids {
		err := s.deps.Tasks.Restore(ctx, userID, id)
		results = append(results, BatchResult{ID: id, OK: err == nil, Error: batchError(err)})
	}
	return results
}

func uniqueIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func batchError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, domain.ErrNotFound) {
		return "not_found"
	}
	return "internal"
}
