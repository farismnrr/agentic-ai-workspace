package application_test

import (
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func TestStartMakesAPrivateSiteWithAReadableUnguessablePath(t *testing.T) {
	svc, _, store, notify := newService(application.Limits{PerUser: 2, Total: 6})

	site, err := svc.Start(ctx, "p1", "python3 -m http.server 8000", 8000)
	if err != nil {
		t.Fatal(err)
	}

	if site.Visibility != domain.Private || site.Status != domain.Running || site.UserID != "ann" ||
		site.BasePath() != "/run/toko-kopi/uuid-1" || notify.n != 1 || store.sites["p1"].Token != "uuid-1" {
		t.Fatalf("unexpected site %+v", site)
	}
}

func TestStartingAgainReplacesTheAppRotatesTheTokenAndKeepsVisibility(t *testing.T) {
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

	if again.Token != "uuid-2" || again.Visibility != domain.Public ||
		len(runtime.stopped) != 1 || runtime.stopped[0] != "c1" {
		t.Fatalf("got %+v, stopped %v", again, runtime.stopped)
	}

	if _, err = svc.Resolve(ctx, "uuid-1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("the old address must stop working")
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
