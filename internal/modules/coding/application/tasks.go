package application

import (
	"context"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// Tasks lists and runs the named commands the person approved.
type Tasks struct {
	file   TaskFile
	runner Runner
}

// NewTasks wires the task use cases.
func NewTasks(file TaskFile, runner Runner) *Tasks {
	return &Tasks{file: file, runner: runner}
}

// List returns the approved tasks, re-reading the file so edits apply at once.
func (t *Tasks) List() ([]domain.Task, error) {
	data, err := t.file.ReadTasks()
	if err != nil {
		return nil, fmt.Errorf("read tasks: %w", err)
	}

	tasks, err := domain.ParseTasks(data)
	if err != nil {
		return nil, fmt.Errorf("parse tasks: %w", err)
	}

	return tasks, nil
}

// Run executes one approved task by name.
func (t *Tasks) Run(ctx context.Context, name string) (RunResult, error) {
	tasks, err := t.List()
	if err != nil {
		return RunResult{}, err
	}

	for _, task := range tasks {
		if task.Name == name {
			result, err := t.runner.Run(ctx, task)
			if err != nil {
				return RunResult{}, fmt.Errorf("run %s: %w", name, err)
			}

			return result, nil
		}
	}

	return RunResult{}, domain.ErrUnknownTask
}

// Save replaces the approved tasks after checking them.
func (t *Tasks) Save(tasks []domain.Task) error {
	if err := domain.ValidateTasks(tasks); err != nil {
		return fmt.Errorf("check tasks: %w", err)
	}

	data, err := domain.EncodeTasks(tasks)
	if err != nil {
		return fmt.Errorf("encode tasks: %w", err)
	}

	if err := t.file.WriteTasks(data); err != nil {
		return fmt.Errorf("write tasks: %w", err)
	}

	return nil
}
