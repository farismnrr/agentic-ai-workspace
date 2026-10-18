package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// FileDiff is one file a task changed: what it held before the task and what it holds now. The page works out
// the lines that differ.
type FileDiff struct {
	Path string
	// OldPath is where the file was before the task moved it.
	OldPath string
	// Status is added, modified, deleted or renamed.
	Status string
	Before string
	After  string
	// Binary files and ones too large to compare carry no text.
	Binary    bool
	Truncated bool
}

// FileDiffs rebuilds what a task changed from the changes it recorded, file by file, against the files as they are.
type FileDiffs interface {
	TaskFiles(ctx context.Context, projectID string, blobs [][]byte) ([]FileDiff, error)
}

// Changes lists the files a task changed, merged over all its steps, the way a pull request shows them.
func (s *Service) Changes(ctx context.Context, userID, taskID string) ([]FileDiff, error) {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}

	events, err := s.deps.Events.EventsAfter(ctx, taskID, 0)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	var blobs [][]byte

	for _, event := range events {
		if event.Kind == domain.EventAction && event.CanUndo() {
			blobs = append(blobs, event.Undo)
		}
	}

	if s.deps.Diffs == nil || len(blobs) == 0 {
		return []FileDiff{}, nil
	}

	diffs, err := s.deps.Diffs.TaskFiles(ctx, task.ProjectID, blobs)
	if err != nil {
		return nil, fmt.Errorf("read changes: %w", err)
	}

	return diffs, nil
}
