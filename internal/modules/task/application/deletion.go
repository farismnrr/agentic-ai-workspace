package application

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const deleteChunkSize = 500

// DeleteTasks stops and removes owned tasks, then cleans files for deleted rows.
func (s *Service) DeleteTasks(ctx context.Context, userID string, ids []string) []BatchResult {
	ids = uniqueIDs(ids)
	owned, results := s.tasksForDeletion(ctx, userID, ids)
	projects := s.deleteTasksInChunks(ctx, userID, owned, results)
	for projectID := range projects {
		s.rebuildNotes(ctx, userID, projectID)
	}
	return results
}

func (s *Service) tasksForDeletion(ctx context.Context, userID string, ids []string) ([]domain.Task, []BatchResult) {
	owned := make([]domain.Task, 0, len(ids))
	results := make([]BatchResult, 0, len(ids))
	for _, id := range ids {
		task, err := s.deps.Tasks.Find(ctx, userID, id)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			results = append(results, BatchResult{ID: id, OK: true})
		case err != nil:
			results = append(results, BatchResult{ID: id, Error: "internal"})
		default:
			if task.Status.Running() {
				s.stopTask(ctx, task)
			}
			owned = append(owned, task)
			results = append(results, BatchResult{ID: id, OK: true})
		}
	}
	return owned, results
}

func (s *Service) deleteTasksInChunks(
	ctx context.Context,
	userID string,
	owned []domain.Task,
	results []BatchResult,
) map[string]struct{} {
	projects := map[string]struct{}{}
	for start := 0; start < len(owned); start += deleteChunkSize {
		end := min(start+deleteChunkSize, len(owned))
		chunk := owned[start:end]
		deleted, err := s.deps.Tasks.DeleteOwned(ctx, userID, taskIDs(chunk))
		if err != nil {
			markDeleteFailure(results, chunk)
		}
		for _, task := range tasksByID(chunk, deleted) {
			s.revokeAgentSession(task)
			projects[task.ProjectID] = struct{}{}
			s.removeTaskFiles(ctx, task)
		}
	}
	return projects
}

func tasksByID(tasks []domain.Task, ids []string) []domain.Task {
	byID := make(map[string]domain.Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}
	deleted := make([]domain.Task, 0, len(ids))
	for _, id := range ids {
		if task, ok := byID[id]; ok {
			deleted = append(deleted, task)
		}
	}
	return deleted
}

func taskIDs(tasks []domain.Task) []string {
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return ids
}

func markDeleteFailure(results []BatchResult, tasks []domain.Task) {
	failed := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		failed[task.ID] = struct{}{}
	}
	for i, result := range results {
		if _, ok := failed[result.ID]; ok {
			results[i] = BatchResult{ID: result.ID, Error: "internal"}
		}
	}
}

func (s *Service) stopTask(ctx context.Context, task domain.Task) {
	state := s.dispatcher.Cancel(task.ID)
	if state != CancelRunning {
		_ = s.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusFailed, s.deps.Now())
		return
	}
	if settling, ok := s.dispatcher.(interface {
		WaitContext(context.Context, string) bool
	}); ok {
		if !settling.WaitContext(ctx, task.ID) {
			slog.Warn("task did not stop before deletion cleanup", "task_id", task.ID)
		}
		return
	}
	if settling, ok := s.dispatcher.(interface{ Wait(string) }); ok {
		settling.Wait(task.ID)
	}
}

func (s *Service) removeTaskFiles(ctx context.Context, task domain.Task) {
	if s.deps.Shots != nil {
		if err := s.deps.Shots.Remove(ctx, task.ID); err != nil {
			slog.Error("remove task screenshots", "task_id", task.ID, "error", err)
		}
	}
	if s.deps.Notes != nil {
		if err := s.deps.Notes.RemoveTask(ctx, task.ProjectID, task.ID); err != nil {
			slog.Error("remove task documents", "task_id", task.ID, "error", err)
		}
	}
}

func (s *Service) rebuildNotes(ctx context.Context, userID, projectID string) {
	if s.deps.Notes == nil {
		return
	}
	tasks, err := s.deps.Tasks.All(ctx, userID, projectID, unlimited)
	if err != nil {
		slog.Warn("list tasks to rebuild project notes", "project_id", projectID, "error", err)
		return
	}
	var index strings.Builder
	index.WriteString("# Project documents\n\nReference material from tasks in this project. " +
		"Check dates and status; prior approval does not approve new work.\n")
	for _, task := range tasks {
		files, syncErr := syncNotes(ctx, s.deps, task)
		if syncErr != nil {
			slog.Warn("rebuild task documents", "task_id", task.ID, "error", syncErr)
			continue
		}
		for _, file := range files {
			index.WriteString("\n- [" + task.Title + "](" + strings.TrimPrefix(file.Path, notesFolder) + ") — " +
				string(task.Status) + "; " + task.UpdatedAt.UTC().Format(time.RFC3339))
		}
	}
	files := []NoteFile{{Path: notesFolder + "index.md", Content: index.String()}}
	if err := s.deps.Notes.Write(ctx, projectID, files); err != nil {
		slog.Warn("rebuild project notes index", "project_id", projectID, "error", err)
	}
}
