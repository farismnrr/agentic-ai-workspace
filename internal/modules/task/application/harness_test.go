package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type step struct {
	err   error
	reply application.Reply
}

// scriptModel answers with a fixed sequence of replies and remembers every request.
type scriptModel struct {
	steps    []step
	requests []application.Request
}

func (s *scriptModel) Complete(ctx context.Context, request application.Request) (application.Reply, error) {
	if err := ctx.Err(); err != nil {
		return application.Reply{}, err //nolint:wrapcheck // test double
	}

	s.requests = append(s.requests, request)

	if len(s.requests) > len(s.steps) {
		return application.Reply{}, &domain.ModelError{Message: "script ran out"}
	}

	next := s.steps[len(s.requests)-1]

	return next.reply, next.err
}

// blockingModel waits until the run is cancelled.
type blockingModel struct{}

func (blockingModel) Complete(ctx context.Context, _ application.Request) (application.Reply, error) {
	<-ctx.Done()

	return application.Reply{}, ctx.Err() //nolint:wrapcheck // test double
}

func call(id, name string, input any) step {
	raw, _ := json.Marshal(input)

	return step{reply: application.Reply{
		Blocks: []domain.Block{{Kind: domain.BlockToolUse, ID: id, Name: name, Input: raw}},
		Usage:  domain.Usage{Input: 10, Output: 5},
	}}
}

func failing(message string, retryable bool) step {
	return step{err: &domain.ModelError{Message: message, Retryable: retryable}}
}

type fakeModels struct {
	model application.Model
	none  bool
}

func (f *fakeModels) Client(context.Context, string, string) (application.Model, error) {
	if f.none {
		return nil, errBoom
	}

	return f.model, nil
}

func (f *fakeModels) Default(context.Context, string) (string, string, error) {
	if f.none {
		return "", "", errBoom
	}

	return "prov", "model-x", nil
}

type rig struct {
	mem     *memory
	models  *fakeModels
	model   *scriptModel
	tools   *fakeTools
	gate    *fakeGate
	runner  *application.Runner
	service *application.Service
	disp    *fakeDispatcher
}

func newRig(model application.Model, steps ...step) *rig {
	mem := newMemory()
	script := &scriptModel{steps: steps}

	if model == nil {
		model = script
	}

	models := &fakeModels{model: model}
	gate := &fakeGate{off: map[string]bool{}}
	tools := &fakeTools{risks: map[string]domain.Risk{}, results: map[string]application.ToolResult{}}
	deps := application.Deps{
		Tasks: mem, Events: mem, Transcript: mem, Usage: mem, Models: models, Persona: fakePersona{},
		Projects: fakeProjects{}, Tools: tools, Gate: gate, IDs: mem, Now: time.Now,
	}
	limits := application.Limits{
		Backoff: func(int) time.Duration { return 0 }, WallTime: 5 * time.Second, TokenCap: 1000, MaxSteps: 10,
		Retries: 2, MaxResult: 24000,
	}
	disp := &fakeDispatcher{}

	return &rig{
		mem: mem, models: models, model: script, tools: tools, gate: gate, runner: application.NewRunner(deps, limits),
		service: application.NewService(deps, disp), disp: disp,
	}
}

func (r *rig) start(t *testing.T, prompt string) domain.Task {
	t.Helper()

	task, err := r.service.Create(context.Background(), "u1", application.CreateInput{
		ProjectID: "proj", Prompt: prompt,
	})
	if err != nil {
		t.Fatal(err)
	}

	return task
}

func (r *rig) run(taskID string) { r.runner.Run(context.Background(), taskID) }

func (r *rig) task(t *testing.T, id string) domain.Task {
	t.Helper()

	task, err := r.mem.Find(context.Background(), "", id)
	if err != nil {
		t.Fatal(err)
	}

	return task
}

func (r *rig) kinds(taskID string) []domain.EventKind {
	events, _ := r.mem.EventsAfter(context.Background(), taskID, 0)
	kinds := make([]domain.EventKind, 0, len(events))

	for _, event := range events {
		kinds = append(kinds, event.Kind)
	}

	return kinds
}

func (r *rig) lastEvent(taskID string, kind domain.EventKind) domain.Event {
	events, _ := r.mem.EventsAfter(context.Background(), taskID, 0)

	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Kind == kind {
			return events[i]
		}
	}

	return domain.Event{}
}

var errBoom = errors.New("boom")
