package application_test

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type fakeModels struct {
	model  application.Model
	tiers  map[string]string
	effort domain.Effort
	none   bool
}

func (f *fakeModels) Client(context.Context, string, string) (application.Model, error) {
	if f.none {
		return nil, errBoom
	}

	return f.model, nil
}

func (f *fakeModels) DefaultEffort(context.Context, string) domain.Effort { return f.effort }

func (f *fakeModels) Tier(_ context.Context, _, tier string) (string, string, domain.Effort, bool) {
	if got, ok := f.tiers[tier]; ok {
		return "prov", got, domain.EffortHigh, true
	}

	return "", "", "", false
}

func (f *fakeModels) Default(context.Context, string) (string, string, error) {
	if f.none {
		return "", "", errBoom
	}

	return "prov", "model-x", nil
}
