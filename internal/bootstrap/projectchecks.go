package bootstrap

import (
	"context"
	"errors"
	"fmt"

	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	projectdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

// projectChecks stores a project's checks as the tasks file the coding tools understand.
type projectChecks struct {
	tools *agentTools
}

func (p projectChecks) List(_ context.Context, projectID string) ([]projectapp.Check, error) {
	services, err := p.tools.services(projectID)
	if err != nil {
		return nil, err
	}

	tasks, err := services.Tasks.List()
	if errors.Is(err, codingdomain.ErrNoTasks) {
		return []projectapp.Check{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read checks: %w", err)
	}

	checks := make([]projectapp.Check, 0, len(tasks))
	for _, task := range tasks {
		checks = append(checks, projectapp.Check{
			Name: task.Name, Description: task.Description, Command: task.Command, TimeoutSeconds: task.TimeoutSeconds,
		})
	}

	return checks, nil
}

func (p projectChecks) Save(_ context.Context, projectID string, checks []projectapp.Check) error {
	services, err := p.tools.services(projectID)
	if err != nil {
		return err
	}

	tasks := make([]codingdomain.Task, 0, len(checks))
	for _, check := range checks {
		tasks = append(tasks, codingdomain.Task{
			Name: check.Name, Description: check.Description, Command: check.Command, TimeoutSeconds: check.TimeoutSeconds,
		})
	}

	if err := services.Tasks.Save(tasks); err != nil {
		if errors.Is(err, codingdomain.ErrBadTasks) {
			return fmt.Errorf("%w: %w", projectdomain.ErrInvalidChecks, err)
		}

		return fmt.Errorf("save checks: %w", err)
	}

	return nil
}
