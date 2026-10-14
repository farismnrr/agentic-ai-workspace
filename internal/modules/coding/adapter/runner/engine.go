package runner

import (
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

// poolWait is how long a task queues for a free place before the server says it is busy.
const poolWait = 30 * time.Second

// EngineConfig is what every runner shares.
type EngineConfig struct {
	// Open connects to Docker. It is retried until it works, so the server can start
	// before Docker does and while the task image is still being downloaded.
	Open func(context.Context) (*sandbox.Docker, error)
	// Size is how many containers may run at once across every workspace.
	Size int
}

// Engine is the infrastructure all runners share: the Docker connection and the pool
// that bounds how many containers run at once, so one person's checks cannot starve
// everyone else of CPU and memory.
type Engine struct {
	places chan struct{}
	lock   chan struct{}
	docker *sandbox.Docker
	open   func(context.Context) (*sandbox.Docker, error)
}

// NewEngine builds the shared infrastructure.
func NewEngine(cfg EngineConfig) *Engine {
	return &Engine{
		places: make(chan struct{}, max(cfg.Size, 1)), lock: make(chan struct{}, 1), open: cfg.Open,
	}
}

// Docker returns the connection, opening it on first use. A failed attempt is not remembered.
func (e *Engine) Docker(ctx context.Context) (*sandbox.Docker, error) {
	select {
	case e.lock <- struct{}{}:
		defer func() { <-e.lock }()
	case <-ctx.Done():
		return nil, domain.ErrServerBusy
	}

	if e.docker != nil {
		return e.docker, nil
	}

	docker, err := e.open(ctx)
	if err != nil {
		return nil, err
	}

	e.docker = docker

	return docker, nil
}

// acquire waits for a free place and returns the function that gives it back.
func (e *Engine) acquire(ctx context.Context) (func(), error) {
	wait := time.NewTimer(poolWait)
	defer wait.Stop()

	select {
	case e.places <- struct{}{}:
		return func() { <-e.places }, nil
	case <-wait.C:
		return nil, domain.ErrServerBusy
	case <-ctx.Done():
		return nil, domain.ErrServerBusy
	}
}
