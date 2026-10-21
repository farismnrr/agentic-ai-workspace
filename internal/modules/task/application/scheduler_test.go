package application_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
)

func TestAProjectRunsOneTaskAtATimeButProjectsRunTogether(t *testing.T) {
	t.Parallel()

	var live, peak atomic.Int32

	var finished sync.WaitGroup

	finished.Add(3)

	scheduler := application.NewScheduler(func(_ context.Context, _ string) {
		defer finished.Done()

		now := live.Add(1)
		for {
			seen := peak.Load()
			if now <= seen || peak.CompareAndSwap(seen, now) {
				break
			}
		}

		time.Sleep(80 * time.Millisecond)
		live.Add(-1)
	})
	scheduler.Start(context.Background(), 3)
	defer scheduler.Stop()

	scheduler.Enqueue("a1", "projA")
	scheduler.Enqueue("a2", "projA")
	scheduler.Enqueue("b1", "projB")
	finished.Wait()

	if peak.Load() != 2 {
		t.Fatalf("expected at most one task per project (2 in total), peak was %d", peak.Load())
	}
}

func TestCancelStopsARunningTaskAndDropsAQueuedOne(t *testing.T) {
	t.Parallel()

	started, stopped := make(chan struct{}), make(chan struct{})
	scheduler := application.NewScheduler(func(ctx context.Context, _ string) {
		close(started)
		<-ctx.Done()
		close(stopped)
	})
	scheduler.Start(context.Background(), 1)
	defer scheduler.Stop()

	scheduler.Enqueue("t1", "proj")
	<-started
	scheduler.Enqueue("t2", "proj")

	if got := scheduler.Cancel("t2"); got != application.CancelQueued {
		t.Errorf("a waiting task is dropped, got %v", got)
	}

	if got := scheduler.Cancel("t1"); got != application.CancelRunning {
		t.Errorf("a running task is cancelled, got %v", got)
	}

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("the running task should have seen its context cancelled")
	}

	if got := scheduler.Cancel("unknown"); got != application.CancelNone {
		t.Errorf("unknown tasks are ignored, got %v", got)
	}
}

func TestWaitContextGivesUpWhenRequestContextEnds(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	scheduler := application.NewScheduler(func(ctx context.Context, _ string) {
		close(started)
		<-ctx.Done()
		// Simulate a task that ignores cancellation and never returns.
		select {}
	})
	scheduler.Start(context.Background(), 1)
	scheduler.Enqueue("task", "project")
	<-started
	scheduler.Cancel("task")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if scheduler.WaitContext(ctx, "task") {
		t.Fatal("WaitContext should report that it stopped waiting early")
	}
}

func TestAContinuationQueuedWhileThePreviousRunSettlesIsNotLost(t *testing.T) {
	t.Parallel()
	started, finish, continued := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var runs atomic.Int32
	scheduler := application.NewScheduler(func(_ context.Context, _ string) {
		if runs.Add(1) == 1 {
			close(started)
			<-finish
			return
		}
		close(continued)
	})
	scheduler.Start(context.Background(), 2)
	defer scheduler.Stop()
	scheduler.Enqueue("task", "project")
	<-started
	scheduler.Enqueue("task", "project")
	scheduler.Enqueue("task", "project")
	close(finish)
	select {
	case <-continued:
	case <-time.After(2 * time.Second):
		t.Fatal("the continuation was lost while the first run settled")
	}
}
