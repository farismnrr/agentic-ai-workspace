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

// signedInTo is a runner of programs where the person is signed in to the programs named.
type signedInTo map[string]bool

func (s signedInTo) Ready(_ context.Context, _, program string) (bool, error) { return s[program], nil }

func (signedInTo) Run(context.Context, CLIRequest, func(CLIEvent)) (CLIResult, error) {
	return CLIResult{}, nil
}

func TestTeammateRunsOnAPlanTheyAreSignedInTo(t *testing.T) {
	t.Parallel()

	task := domain.Task{ProviderID: "p1", Model: "lead-model"}
	runner := &Runner{deps: Deps{Models: catalog{}, CLI: signedInTo{"chatgpt": true}}}

	role := AgentRole{ProviderID: "plan:chatgpt", Model: "gpt-x", Effort: domain.EffortHigh}

	providerID, model, effort := runner.pickModel(context.Background(), task, role)
	if providerID != "plan:chatgpt" || model != "gpt-x" || effort != domain.EffortHigh {
		t.Fatalf("a teammate was sent to %s %s %s", providerID, model, effort)
	}
}

func TestTeammateFallsBackWhenThePlanIsNotSignedIn(t *testing.T) {
	t.Parallel()

	task := domain.Task{ProviderID: "p1", Model: "lead-model"}
	runner := &Runner{deps: Deps{Models: catalog{providerID: "plan:claude", model: "default"}, CLI: signedInTo{}}}

	role := AgentRole{ProviderID: "plan:chatgpt", Model: "gpt-x"}

	providerID, model, _ := runner.pickModel(context.Background(), task, role)
	if providerID != "p1" || model != "lead-model" {
		t.Fatalf("a teammate was sent to %s %s", providerID, model)
	}
}
