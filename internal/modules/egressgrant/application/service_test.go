package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
)

var (
	t0    = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	agent = domain.Actor{Kind: domain.ActorAgent, ID: "agent:t1"}
	errDB = errors.New("disk on fire")
)

// loose is a store that hands back whatever it is told to, to show the service does not trust it.
type loose struct {
	active  []domain.Grant
	err     error
	created []domain.Grant
}

func (l *loose) Create(_ context.Context, g domain.Grant) error {
	l.created = append(l.created, g)

	return l.err
}

func (l *loose) Get(context.Context, string) (domain.Grant, error) { return domain.Grant{}, l.err }

func (l *loose) Decide(context.Context, string, func(domain.Grant) (domain.Grant, error)) (domain.Grant, error) {
	return domain.Grant{}, l.err
}

func (l *loose) Active(context.Context, string, string, time.Time) ([]domain.Grant, error) {
	return l.active, l.err
}

func (l *loose) List(context.Context, string, string) ([]domain.Grant, error) { return l.active, l.err }

func (l *loose) TaskLive(context.Context, string, string) (bool, error) { return l.err == nil, l.err }

func (l *loose) EndInactive(context.Context, time.Time) (int, error) { return 0, l.err }

func (l *loose) LiveApproved(context.Context, time.Time) ([]domain.Grant, error) {
	return l.active, l.err
}

type ids struct{ err error }

func (i ids) UUID() (string, error) { return "g1", i.err }

func service(store application.Store, id application.IDs) *application.Service {
	return application.NewService(store, id, func() time.Time { return t0 })
}

func approved(project, task, host string, ends time.Time) domain.Grant {
	return domain.Grant{
		ID: "x", ProjectID: project, TaskID: task, Host: host, Status: domain.StatusApproved, ExpiresAt: ends,
	}
}

func TestActiveDropsAnythingTheStoreReturnedThatDoesNotQualify(t *testing.T) {
	store := &loose{active: []domain.Grant{
		approved("p1", "t1", "ok.example", t0.Add(time.Hour)),
		approved("p1", "t2", "other-task.example", t0.Add(time.Hour)),
		approved("p2", "t1", "other-project.example", t0.Add(time.Hour)),
		approved("p1", "t1", "expired.example", t0),
		approved("p1", "t1", "10.0.0.1", t0.Add(time.Hour)),
		{
			ID: "y", ProjectID: "p1", TaskID: "t1", Host: "pending.example", Status: domain.StatusPending,
			ExpiresAt: t0.Add(time.Hour),
		},
	}}

	got, err := service(store, ids{}).Active(context.Background(), "p1", "t1")
	if err != nil || len(got) != 1 || got[0].Host != "ok.example" {
		t.Fatalf("got %+v %v", got, err)
	}
}

func TestRequestNeverReachesTheStoreWhenInvalid(t *testing.T) {
	store := &loose{}
	s := service(store, ids{})

	for _, host := range []string{"127.0.0.1", "localhost", "*.example.com", "a.example:443", ""} {
		_, err := s.Request(context.Background(), "p1", "t1", host, "why", agent)
		if !errors.Is(err, domain.ErrInvalidHost) {
			t.Errorf("%q: %v", host, err)
		}
	}

	_, err := s.Request(context.Background(), "p1", "t1", "a.example", "", agent)
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Errorf("no reason: %v", err)
	}

	if len(store.created) != 0 {
		t.Fatalf("the store saw %+v", store.created)
	}
}

func TestFailuresOfTheStoreAndIDsSurface(t *testing.T) {
	ctx := context.Background()
	s := service(&loose{err: errDB}, ids{})

	if _, err := s.Request(ctx, "p1", "t1", "a.example", "why", agent); !errors.Is(err, errDB) {
		t.Errorf("request: %v", err)
	}

	if _, err := s.Approve(ctx, "g", domain.Decider{ID: "u", Admin: true}, 0); !errors.Is(err, errDB) {
		t.Errorf("approve: %v", err)
	}

	if _, err := s.Active(ctx, "p1", "t1"); !errors.Is(err, errDB) {
		t.Errorf("active: %v", err)
	}

	if _, err := s.List(ctx, "p1", "t1"); !errors.Is(err, errDB) {
		t.Errorf("list: %v", err)
	}

	store := &loose{}

	_, err := service(store, ids{err: errDB}).Request(ctx, "p1", "t1", "a.example", "why", agent)
	if !errors.Is(err, errDB) {
		t.Errorf("ids: %v", err)
	}

	if len(store.created) != 0 {
		t.Error("nothing is saved when no id could be made")
	}
}
