package application_test

import (
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func TestStartMakesAPrivateSiteAtTheProjectName(t *testing.T) {
	svc, _, store, notify := newService(application.Limits{PerUser: 2, Total: 6})

	site, err := svc.Start(ctx, "p1", "python3 -m http.server 8000", 8000)
	if err != nil {
		t.Fatal(err)
	}

	if site.Visibility != domain.Private || site.Status != domain.Running || site.UserID != "ann" ||
		site.BasePath() != "/run/toko-kopi" || notify.n != 1 || store.sites["p1"].Token != "uuid-1" {
		t.Fatalf("unexpected site %+v", site)
	}
}

func TestStartingAgainReplacesTheAppAndKeepsTheAddressAndVisibility(t *testing.T) {
	svc, runtime, _, _ := newService(application.Limits{PerUser: 1, Total: 1})

	if _, err := svc.Start(ctx, "p1", "serve", 3000); err != nil {
		t.Fatal(err)
	}

	if err := svc.SetVisibility(ctx, "ann", "p1", "public"); err != nil {
		t.Fatal(err)
	}

	again, err := svc.Start(ctx, "p1", "serve", 3000)
	if err != nil {
		t.Fatalf("restarting must not count against the limit: %v", err)
	}

	if again.Token != "uuid-1" || again.Visibility != domain.Public ||
		len(runtime.stopped) != 1 || runtime.stopped[0] != "c1" {
		t.Fatalf("got %+v, stopped %v", again, runtime.stopped)
	}

	if _, err = svc.Resolve(ctx, "toko-kopi"); err != nil {
		t.Fatalf("the address must keep working: %v", err)
	}

	if _, err = svc.ResolveOld(ctx, "uuid-1"); err != nil {
		t.Fatalf("the old address with the token must still lead here: %v", err)
	}
}

func TestTwoProjectsWithTheSameNameGetDifferentAddresses(t *testing.T) {
	svc, _, _, _ := newService(application.Limits{PerUser: 5, Total: 6})

	first, err := svc.Start(ctx, "p1", "serve", 3000)
	if err != nil {
		t.Fatal(err)
	}

	second, err := svc.Start(ctx, "r1", "serve", 3001)
	if err != nil {
		t.Fatal(err)
	}

	if first.Slug != "toko-kopi" || second.Slug != "toko-kopi-2" {
		t.Fatalf("addresses %q and %q", first.Slug, second.Slug)
	}

	again, err := svc.Start(ctx, "p1", "serve", 3000)
	if err != nil || again.Slug != "toko-kopi" {
		t.Fatalf("a project keeps its own address: %+v %v", again, err)
	}
}

func TestAStoppedSiteStartsAgainFromTheSitesPageAtTheSameAddress(t *testing.T) {
	svc, runtime, store, _ := newService(application.Limits{PerUser: 2, Total: 6})

	first, err := svc.Start(ctx, "p1", "python3 -m http.server 8000", 8000)
	if err != nil {
		t.Fatal(err)
	}

	if err = svc.Stop(ctx, "ann", "p1"); err != nil {
		t.Fatal(err)
	}

	again, err := svc.Restart(ctx, "ann", "p1")
	if err != nil {
		t.Fatal(err)
	}

	if again.Token != first.Token || again.Status != domain.Running || again.Port != 8000 ||
		store.sites["p1"].Command != "python3 -m http.server 8000" || runtime.started != 2 {
		t.Fatalf("same address, same command, running again: %+v", again)
	}

	if _, err = svc.Restart(ctx, "bob", "p1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("someone else cannot start it: %v", err)
	}
}

func TestASiteNeverStartedBeforeCannotBeStartedByHand(t *testing.T) {
	svc, _, store, _ := newService(application.Limits{PerUser: 2, Total: 6})
	store.sites["p1"] = domain.Site{ProjectID: "p1", UserID: "ann", Token: "old", Status: domain.Stopped}

	if _, err := svc.Restart(ctx, "ann", "p1"); !errors.Is(err, domain.ErrNeverStarted) {
		t.Fatalf("an app that was started before commands were remembered: %v", err)
	}
}

func TestLimitsPerPersonAndPerServer(t *testing.T) {
	svc, _, _, _ := newService(application.Limits{PerUser: 2, Total: 2})

	for _, id := range []string{"p1", "p2"} {
		if _, err := svc.Start(ctx, id, "serve", 3000); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := svc.Start(ctx, "p3", "serve", 3000); !errors.Is(err, domain.ErrTooMany) {
		t.Errorf("third site of one person: %v", err)
	}

	if _, err := svc.Start(ctx, "q1", "serve", 3000); !errors.Is(err, domain.ErrServerBusy) {
		t.Errorf("server full: %v", err)
	}
}

func TestStartRefusesBadInputAndUnknownProjects(t *testing.T) {
	svc, _, _, _ := newService(application.Limits{PerUser: 2, Total: 6})

	attempts := map[string]struct {
		project, command string
		port             int
	}{
		"low port":   {"p1", "x", 80},
		"no command": {"p1", "  ", 3000},
		"no project": {"nope", "x", 3000},
	}

	for name, a := range attempts {
		if _, err := svc.Start(ctx, a.project, a.command, a.port); err == nil {
			t.Errorf("%s must be refused", name)
		}
	}
}

func TestARefusedStartLeavesTheRunningAppAlone(t *testing.T) {
	t.Parallel()

	svc, runtime, store, _ := newService(application.Limits{PerUser: 1, Total: 4})

	if _, err := svc.Start(ctx, "p1", "serve", 3000); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Start(ctx, "p2", "serve", 3001); !errors.Is(err, domain.ErrTooMany) {
		t.Fatalf("a second app is over the cap of one: %v", err)
	}

	// A start in the project that already runs is a replacement, not a second app.
	if _, err := svc.Start(ctx, "p1", "serve", 3002); err != nil {
		t.Fatalf("replacing the project's own app: %v", err)
	}

	if len(runtime.stopped) != 1 || store.sites["p1"].Status != domain.Running {
		t.Fatalf("only the replaced app was stopped: %v", runtime.stopped)
	}
}

func TestEveryProjectCanRunItsOwnAppWhenThereIsNoCap(t *testing.T) {
	t.Parallel()

	svc, _, _, _ := newService(application.Limits{})

	for project, port := range map[string]int{"p1": 3000, "p2": 3001, "p3": 3002} {
		if _, err := svc.Start(ctx, project, "serve", port); err != nil {
			t.Fatalf("%s: with no cap each project runs its own app: %v", project, err)
		}
	}
}
