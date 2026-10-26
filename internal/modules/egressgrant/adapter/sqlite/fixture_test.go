package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/adapter/sqlite"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/egressgrant/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/database"
)

var (
	t0    = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	agent = domain.Actor{Kind: domain.ActorAgent, ID: "agent:t1"}
	p1own = domain.Decider{ID: "u1", ProjectID: "p1", Owner: true}
	p2own = domain.Decider{ID: "u2", ProjectID: "p2", Owner: true}
)

type counter struct{ n atomic.Int64 }

func (c *counter) UUID() (string, error) { return "g" + strconv.FormatInt(c.n.Add(1), 10), nil }

// world is a real database with two users, one project each, two tasks in p1 and one in p2, and a movable clock.
type world struct {
	*application.Service
	db    *sql.DB
	clock *time.Time
}

func newWorld(t *testing.T) world {
	t.Helper()

	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	for _, q := range []string{
		`INSERT INTO users (id, name, email, password_hash, created_at)
			VALUES ('u1', 'A', 'a@example.com', 'x', 1), ('u2', 'B', 'b@example.com', 'x', 1)`,
		`INSERT INTO projects (id, user_id, name, slug, created_at)
			VALUES ('p1', 'u1', 'one', 'one', 1), ('p2', 'u2', 'two', 'two', 1)`,
		`INSERT INTO tasks (id, project_id, user_id, title, status, mode, provider_id, model, created_at, updated_at)
			VALUES ('t1', 'p1', 'u1', 'x', 'working', 'risky', 'p', 'm', 1, 1),
			('t2', 'p1', 'u1', 'x', 'working', 'risky', 'p', 'm', 1, 1),
			('t3', 'p2', 'u2', 'x', 'working', 'risky', 'p', 'm', 1, 1)`,
	} {
		if _, err = db.ExecContext(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}

	clock := t0
	service := application.NewService(sqlite.NewStore(db), &counter{}, func() time.Time { return clock }).
		WithLimits(application.Limits{MaxPending: 1000, MaxByAgent: 1000, Window: time.Hour})

	return world{Service: service, db: db, clock: &clock}
}

func (w world) ask(t *testing.T, project, task, host string) domain.Grant {
	t.Helper()

	g, err := w.Request(context.Background(), project, task, host, "need it", agent)
	if err != nil {
		t.Fatal(err)
	}

	return g
}

func (w world) approve(t *testing.T, g domain.Grant, by domain.Decider, ttl time.Duration) {
	t.Helper()

	if _, err := w.Approve(context.Background(), g.ID, by, ttl); err != nil {
		t.Fatal(err)
	}
}

func (w world) count(t *testing.T, where string) int {
	t.Helper()

	var n int

	query := `SELECT COUNT(*) FROM egress_grants ` + where
	if err := w.db.QueryRowContext(context.Background(), query).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

// only wraps an error from the service so the race helpers return something this package worded.
func only(err error) error {
	if err != nil {
		return fmt.Errorf("call: %w", err)
	}

	return nil
}

type tally struct {
	firstOther                                             error
	ok, notPending, notApproved, duplicate, limited, other int
}

// race runs fn on n goroutines released together and sorts what they returned.
func race(n int, fn func(i int) error) tally {
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		out   tally
		start = make(chan struct{})
	)

	for i := range n {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			err := fn(i)

			mu.Lock()
			defer mu.Unlock()

			out.add(err)
		}()
	}

	close(start)
	wg.Wait()

	return out
}

func (o *tally) add(err error) {
	switch {
	case err == nil:
		o.ok++
	case errors.Is(err, domain.ErrNotPending):
		o.notPending++
	case errors.Is(err, domain.ErrNotApproved):
		o.notApproved++
	case errors.Is(err, application.ErrDuplicate):
		o.duplicate++
	case errors.Is(err, application.ErrTooManyPending), errors.Is(err, application.ErrRateLimited):
		o.limited++
	default:
		o.other++
		o.firstOther = err
	}
}
