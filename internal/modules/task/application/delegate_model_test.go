package application

import (
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var errNone = errors.New("none")

type catalog struct{ providerID, model string }

func (c catalog) Client(context.Context, string, string) (Model, error) {
	return nil, errNone
}

func (c catalog) Default(context.Context, string) (string, string, error) {
	return c.providerID, c.model, nil
}

func (catalog) DefaultEffort(context.Context, string) domain.Effort { return "" }

func TestTeammateNeverRunsOnAPlan(t *testing.T) {
	t.Parallel()

	task := domain.Task{ProviderID: "p1", Model: "lead-model"}
	runner := &Runner{deps: Deps{Models: catalog{providerID: "plan:claude", model: "default"}}}

	role := AgentRole{ProviderID: "plan:chatgpt", Model: "default"}

	providerID, model, _ := runner.pickModel(context.Background(), task, role)
	if providerID != "p1" || model != "lead-model" {
		t.Fatalf("a teammate was sent to %s %s", providerID, model)
	}
}
