package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

type taskInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type tasksOutput struct {
	Tasks []taskInfo `json:"tasks"`
}

func (t *tools) listTasks(ctx context.Context, _ Target) (tasksOutput, report, error) {
	svc, err := t.services(ctx)
	if err != nil {
		return tasksOutput{}, report{}, err
	}

	tasks, err := svc.Tasks.List()
	if errors.Is(err, domain.ErrNoTasks) {
		return tasksOutput{Tasks: []taskInfo{}}, report{}, nil
	}
	if err != nil {
		return tasksOutput{}, report{}, fmt.Errorf("list tasks: %w", err)
	}

	out := tasksOutput{Tasks: make([]taskInfo, 0, len(tasks))}
	for _, task := range tasks {
		kind := task.Kind
		if kind == "" {
			kind = "check"
		}
		out.Tasks = append(out.Tasks, taskInfo{Name: task.Name, Description: task.Description, Kind: kind})
	}

	return out, report{}, nil
}
