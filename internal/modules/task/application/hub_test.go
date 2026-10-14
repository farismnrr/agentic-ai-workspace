package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func receive(t *testing.T, sub *application.Subscription) application.Update {
	t.Helper()

	select {
	case update, open := <-sub.C:
		if !open {
			t.Fatal("subscription closed")
		}

		return update
	case <-time.After(time.Second):
		t.Fatal("nothing arrived")
	}

	return application.Update{}
}

func TestOnlyFollowersOfATopicHearAboutIt(t *testing.T) {
	t.Parallel()

	hub := application.NewHub()
	mine := hub.Subscribe(application.TaskTopic("t1"))
	other := hub.Subscribe(application.TaskTopic("t2"))

	hub.Delta("t1", "hi")

	if got := receive(t, mine); got.Delta != "hi" {
		t.Fatalf("got %+v", got)
	}

	select {
	case update := <-other.C:
		t.Fatalf("another task's follower must hear nothing, got %+v", update)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestASlowFollowerNeverBlocksThePublisher(t *testing.T) {
	t.Parallel()

	hub := application.NewHub()
	_ = hub.Subscribe(application.TaskTopic("t1"))
	done := make(chan struct{})

	go func() {
		for range 1000 {
			hub.Delta("t1", "x")
		}

		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publishing must not wait for followers")
	}
}

func TestClosingTheHubEndsEveryStream(t *testing.T) {
	t.Parallel()

	hub := application.NewHub()
	sub := hub.Subscribe(application.UserTopic("u1"))
	hub.Close()

	if _, open := <-sub.C; open {
		t.Fatal("followers must be released when the hub closes")
	}

	if _, open := <-hub.Subscribe(application.UserTopic("u1")).C; open {
		t.Fatal("following a closed hub ends at once")
	}
}

func TestEveryWriteWakesTheTaskAndItsOwner(t *testing.T) {
	t.Parallel()

	r := newRig(nil)
	task := r.start(t, "Hello")
	hub := application.NewHub()
	watched := application.Observe(application.Deps{Tasks: r.mem, Events: r.mem}, hub)
	task1, owner := hub.Subscribe(application.TaskTopic(task.ID)), hub.Subscribe(application.UserTopic("u1"))

	if err := watched.Tasks.SetStatus(context.Background(), task.ID, domain.StatusWorking, time.Now()); err != nil {
		t.Fatal(err)
	}

	event := domain.Event{Kind: domain.EventMessage}
	if _, err := watched.Events.AppendEvent(context.Background(), task.ID, event); err != nil {
		t.Fatal(err)
	}

	for _, sub := range []*application.Subscription{task1, owner} {
		if got := receive(t, sub); got.TaskID != task.ID || got.Delta != "" {
			t.Errorf("expected a plain change for %s, got %+v", task.ID, got)
		}
	}
}
