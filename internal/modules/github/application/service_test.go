package application_test

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

type memStore struct {
	recs map[string]application.Record
	mu   sync.Mutex
}

func (m *memStore) Load(_ context.Context, id string) (application.Record, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.recs[id]

	return rec, ok, nil
}

func (m *memStore) Save(_ context.Context, rec application.Record, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recs[rec.UserID] = rec

	return nil
}

func (m *memStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.recs, id)

	return nil
}

// fakeGitHub hands out numbered tokens and kills the old refresh token on every refresh.
type fakeGitHub struct {
	now       func() time.Time
	valid     sync.Map
	refreshes atomic.Int32
	reject    atomic.Bool
}

func (f *fakeGitHub) AuthorizeURL(state, challenge string) string {
	return "https://github.test/authorize?" + url.Values{"state": {state}, "code_challenge": {challenge}}.Encode()
}

func (f *fakeGitHub) issue(owner string) domain.Tokens {
	n := strconv.Itoa(int(f.refreshes.Add(1)))
	f.valid.Store("r-"+owner+"-"+n, true)

	return domain.Tokens{
		Access: "ghu_" + owner + "_" + n, Refresh: "r-" + owner + "-" + n, AccessExpires: f.now().Add(8 * time.Hour),
	}
}

func (f *fakeGitHub) Exchange(_ context.Context, code, _ string) (domain.Tokens, error) {
	return f.issue(code), nil
}

func (f *fakeGitHub) Refresh(_ context.Context, refresh string) (domain.Tokens, error) {
	if f.reject.Load() {
		return domain.Tokens{}, domain.ErrRejected
	}

	if _, ok := f.valid.LoadAndDelete(refresh); !ok {
		return domain.Tokens{}, domain.ErrRejected
	}

	owner := refresh[2 : len(refresh)-2]

	return f.issue(owner), nil
}

func (f *fakeGitHub) Profile(_ context.Context, access string) (domain.Profile, error) {
	return domain.Profile{ID: 7, Login: access}, nil
}

func (f *fakeGitHub) Revoke(context.Context, string) error { return nil }

func (f *fakeGitHub) CreateRepo(_ context.Context, _ string, spec domain.NewRepo) (domain.Repo, error) {
	return domain.Repo{FullName: "ann/" + spec.Name, Private: spec.Private}, nil
}

func (f *fakeGitHub) Repos(context.Context, string) ([]domain.Repo, error) {
	return []domain.Repo{{FullName: "ann/shop"}}, nil
}

type plain struct{}

func (plain) Encrypt(b []byte) (string, error) { return "sealed:" + string(b), nil }
func (plain) Decrypt(s string) ([]byte, error) { return []byte(s[len("sealed:"):]), nil }

type counter struct{ n atomic.Int32 }

func (c *counter) New() (string, string, error) {
	n := strconv.Itoa(int(c.n.Add(1)))

	return "secret-" + n, "hash-" + n, nil
}

type rig struct {
	svc   *application.Service
	gh    *fakeGitHub
	clock *time.Time
}

func newRig() rig {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	clock := &now
	gh := &fakeGitHub{now: func() time.Time { return *clock }}
	svc := application.NewService(&memStore{recs: map[string]application.Record{}}, gh, plain{}, &counter{},
		func() time.Time { return *clock }, true)

	return rig{svc: svc, gh: gh, clock: clock}
}

// connect runs a whole sign-in for one person and returns the access token GitHub gave them.
func (r rig) connect(t *testing.T, user, code string) {
	t.Helper()

	target, err := r.svc.Start(user)
	if err != nil {
		t.Fatal(err)
	}

	parsed, _ := url.Parse(target)
	if err = r.svc.Finish(context.Background(), user, code, parsed.Query().Get("state")); err != nil {
		t.Fatal(err)
	}
}

func TestFinishRejectsAnotherPersonsState(t *testing.T) {
	r := newRig()
	target, _ := r.svc.Start("alice")
	state := mustQuery(t, target, "state")

	if err := r.svc.Finish(context.Background(), "bob", "stolen", state); !errors.Is(err, domain.ErrState) {
		t.Fatalf("bob completed alice's sign-in: %v", err)
	}

	if err := r.svc.Finish(context.Background(), "alice", "mine", state); !errors.Is(err, domain.ErrState) {
		t.Fatalf("a state must work only once: %v", err)
	}
}

func TestFinishRejectsExpiredState(t *testing.T) {
	r := newRig()
	target, _ := r.svc.Start("alice")
	*r.clock = r.clock.Add(11 * time.Minute)

	state := mustQuery(t, target, "state")
	if err := r.svc.Finish(context.Background(), "alice", "c", state); !errors.Is(err, domain.ErrState) {
		t.Fatalf("expired state was accepted: %v", err)
	}
}

func TestCredentialsNeverCrossPeople(t *testing.T) {
	r := newRig()
	r.connect(t, "alice", "alice")
	r.connect(t, "bob", "bob")

	a, err := r.svc.Credentials(context.Background(), "alice")
	b, errB := r.svc.Credentials(context.Background(), "bob")

	if err != nil || errB != nil || a.Token == b.Token || a.Token[:9] != "ghu_alice" || b.Token[:7] != "ghu_bob" {
		t.Fatalf("alice %q (%v), bob %q (%v)", a.Token, err, b.Token, errB)
	}

	if _, err = r.svc.Credentials(context.Background(), "carol"); !errors.Is(err, domain.ErrNotConnected) {
		t.Fatalf("a person who never connected got credentials: %v", err)
	}
}

func mustQuery(t *testing.T, target, key string) string {
	t.Helper()

	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}

	return parsed.Query().Get(key)
}
