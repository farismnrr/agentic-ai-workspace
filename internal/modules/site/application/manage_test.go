package application_test

import (
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func TestOnlyTheOwnerCanStopOrChangeASite(t *testing.T) {
	svc, runtime, _, _ := newService(application.Limits{PerUser: 2, Total: 6})
	_, _ = svc.Start(ctx, "p1", "serve", 3000)

	if err := svc.Stop(ctx, "bob", "p1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("another person stopping: %v", err)
	}

	if err := svc.SetVisibility(ctx, "bob", "p1", "public"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("another person publishing: %v", err)
	}

	if err := svc.SetVisibility(ctx, "ann", "p1", "everyone"); !errors.Is(err, domain.ErrBadChoice) {
		t.Errorf("unknown visibility: %v", err)
	}

	if err := svc.Stop(ctx, "ann", "p1"); err != nil || len(runtime.stopped) != 1 {
		t.Errorf("owner stop: %v %v", err, runtime.stopped)
	}

	if _, err := svc.Resolve(ctx, "uuid-1"); !errors.Is(err, domain.ErrNotFound) {
		t.Error("a stopped site must not resolve")
	}
}

func TestStopUserAndStopProjectAndRestartReset(t *testing.T) {
	svc, runtime, store, _ := newService(application.Limits{PerUser: 2, Total: 6})
	_, _ = svc.Start(ctx, "p1", "serve", 3000)
	_, _ = svc.Start(ctx, "p2", "serve", 3001)
	_, _ = svc.Start(ctx, "q1", "serve", 3002)

	svc.StopProject(ctx, "p1")

	if err := svc.StopUser(ctx, "ann"); err != nil {
		t.Fatal(err)
	}

	sites := store.sites
	if sites["p1"].Status != domain.Stopped || sites["p2"].Status != domain.Stopped ||
		sites["q1"].Status != domain.Running {
		t.Fatalf("statuses %+v", store.sites)
	}

	if len(runtime.stopped) != 2 {
		t.Errorf("containers stopped: %v", runtime.stopped)
	}

	if err := svc.Reconcile(ctx); err != nil || store.sites["q1"].Status != domain.Stopped {
		t.Errorf("a restart leaves nothing running: %v", err)
	}
}

func TestAFailedStartLeavesNothingBehind(t *testing.T) {
	svc, runtime, store, _ := newService(application.Limits{PerUser: 2, Total: 6})
	runtime.failure = domain.ErrNotStarted

	if _, err := svc.Start(ctx, "p1", "serve", 3000); !errors.Is(err, domain.ErrNotStarted) {
		t.Fatalf("got %v", err)
	}

	if _, saved := store.sites["p1"]; saved {
		t.Error("a site that never started must not be saved")
	}
}

func TestAFailedRestartDoesNotLeaveTheOldSiteMarkedRunning(t *testing.T) {
	svc, runtime, store, _ := newService(application.Limits{PerUser: 2, Total: 6})
	_, _ = svc.Start(ctx, "p1", "serve", 3000)
	runtime.failure = domain.ErrNotStarted

	if _, err := svc.Start(ctx, "p1", "serve", 3000); err == nil {
		t.Fatal("the restart should have failed")
	}

	if store.sites["p1"].Status != domain.Stopped {
		t.Fatalf("the old app was stopped, so the site must say so: %+v", store.sites["p1"])
	}
}
