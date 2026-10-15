package application_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

func TestCredentialsRefreshOnceWhenCalledTogether(t *testing.T) {
	r := newRig()
	r.connect(t, "alice", "alice")
	before := r.gh.refreshes.Load()
	*r.clock = r.clock.Add(7*time.Hour + 55*time.Minute)

	var wg sync.WaitGroup

	tokens := make([]string, 8)
	for i := range tokens {
		wg.Add(1)

		go func() {
			defer wg.Done()

			cred, err := r.svc.Credentials(context.Background(), "alice")
			if err != nil {
				t.Error(err)
			}

			tokens[i] = cred.Token
		}()
	}

	wg.Wait()

	if got := r.gh.refreshes.Load() - before; got != 1 {
		t.Fatalf("refreshed %d times; a refresh token works once", got)
	}

	for _, token := range tokens {
		if token != tokens[0] {
			t.Fatalf("callers got different tokens: %v", tokens)
		}
	}

	again, err := r.svc.Credentials(context.Background(), "alice")
	if err != nil || again.Token != tokens[0] {
		t.Fatalf("new token was not kept: %q %v", again.Token, err)
	}
}

func TestRejectedRefreshDisconnects(t *testing.T) {
	r := newRig()
	r.connect(t, "alice", "alice")
	r.gh.reject.Store(true)
	*r.clock = r.clock.Add(7*time.Hour + 55*time.Minute)

	if _, err := r.svc.Credentials(context.Background(), "alice"); !errors.Is(err, domain.ErrNotConnected) {
		t.Fatalf("revoked grant still gave credentials: %v", err)
	}

	if _, err := r.svc.Status(context.Background(), "alice"); !errors.Is(err, domain.ErrNotConnected) {
		t.Fatalf("connection should be forgotten: %v", err)
	}
}

func TestDisconnectForgets(t *testing.T) {
	r := newRig()
	r.connect(t, "alice", "alice")

	if err := r.svc.Disconnect(context.Background(), "alice"); err != nil {
		t.Fatal(err)
	}

	if _, err := r.svc.Credentials(context.Background(), "alice"); !errors.Is(err, domain.ErrNotConnected) {
		t.Fatalf("still connected: %v", err)
	}
}
