package application

import (
	"context"
	"slices"
	"sync"
)

// CancelState says what Cancel found.
type CancelState int

// Results of Cancel.
const (
	CancelNone CancelState = iota
	CancelRunning
	CancelQueued
)

// Dispatcher runs tasks in the background.
type Dispatcher interface {
	Enqueue(taskID, projectID string)
	Cancel(taskID string) CancelState
}

type job struct {
	taskID    string
	projectID string
}

// Scheduler runs tasks on a small pool of workers. A project runs one task at a
// time so two agents never edit the same folder together.
type Scheduler struct {
	run     func(ctx context.Context, taskID string)
	cond    *sync.Cond
	running map[string]context.CancelFunc
	busy    map[string]bool
	queue   []job
	wg      sync.WaitGroup
	mu      sync.Mutex
	stopped bool
}

// NewScheduler builds a scheduler that calls run for each task.
func NewScheduler(run func(ctx context.Context, taskID string)) *Scheduler {
	s := &Scheduler{run: run, running: map[string]context.CancelFunc{}, busy: map[string]bool{}}
	s.cond = sync.NewCond(&s.mu)

	return s
}

// Start launches the workers; they stop when Stop is called or ctx ends.
func (s *Scheduler) Start(ctx context.Context, workers int) {
	for range max(workers, 1) {
		s.wg.Add(1)

		go s.work(ctx)
	}

	go func() {
		<-ctx.Done()
		s.Stop()
	}()
}

// Enqueue schedules a task unless it is already queued or running.
func (s *Scheduler) Enqueue(taskID, projectID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, running := s.running[taskID]
	queued := slices.ContainsFunc(s.queue, func(j job) bool { return j.taskID == taskID })

	if running || queued {
		return
	}

	s.queue = append(s.queue, job{taskID: taskID, projectID: projectID})
	s.cond.Broadcast()
}

// Cancel stops a running task or drops a queued one.
func (s *Scheduler) Cancel(taskID string) CancelState {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cancel, running := s.running[taskID]; running {
		cancel()

		return CancelRunning
	}

	for i, queued := range s.queue {
		if queued.taskID == taskID {
			s.queue = slices.Delete(s.queue, i, i+1)

			return CancelQueued
		}
	}

	return CancelNone
}

// Stop cancels running tasks and waits for the workers to finish.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	s.stopped = true

	for _, cancel := range s.running {
		cancel()
	}

	s.cond.Broadcast()
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *Scheduler) work(parent context.Context) {
	defer s.wg.Done()

	for {
		ctx, cancel := context.WithCancel(parent)

		next, ok := s.take(cancel)
		if !ok {
			cancel()

			return
		}

		s.run(ctx, next.taskID)
		cancel()
		s.release(next)
	}
}

// take blocks until a job whose project is free exists, and claims it.
func (s *Scheduler) take(cancel context.CancelFunc) (job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		if s.stopped {
			return job{}, false
		}

		if index := slices.IndexFunc(s.queue, func(j job) bool { return !s.busy[j.projectID] }); index >= 0 {
			picked := s.queue[index]
			s.queue = slices.Delete(s.queue, index, index+1)
			s.busy[picked.projectID], s.running[picked.taskID] = true, cancel

			return picked, true
		}

		s.cond.Wait()
	}
}

func (s *Scheduler) release(done job) {
	s.mu.Lock()
	delete(s.busy, done.projectID)
	delete(s.running, done.taskID)
	s.cond.Broadcast()
	s.mu.Unlock()
}
