package application

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/gateway/domain"
)

var errMissing = errors.New("missing")

type memStore map[string]string

func (m memStore) Get(_ context.Context, id string) (string, error) {
	if v, ok := m[id]; ok {
		return v, nil
	}

	return "", ErrUnset
}
func (m memStore) Set(_ context.Context, id, provider string) error { m[id] = provider; return nil }
func (m memStore) Delete(_ context.Context, id string) error        { delete(m, id); return nil }

type owners map[string]string

func (o owners) Owner(_ context.Context, id string) (string, string, error) {
	if u, ok := o[id]; ok {
		return u, id, nil
	}

	return "", "", errMissing
}

type fakeProviders struct {
	options map[string][]domain.Choice // by user
	keys    map[string]string          // by provider id
}

func (f fakeProviders) Eligible(_ context.Context, user string) ([]domain.Choice, error) {
	return f.options[user], nil
}

func (f fakeProviders) Credential(_ context.Context, _, id string) (string, string, error) {
	for _, list := range f.options {
		for _, c := range list {
			if c.ID == id {
				return c.BaseURL, f.keys[id], nil
			}
		}
	}

	return "", "", errMissing
}

func newService() (*Service, memStore, *fakeProviders) {
	store := memStore{}
	providers := &fakeProviders{
		options: map[string][]domain.Choice{
			"alice": {{ID: "gw", Name: "9router", BaseURL: "http://localhost:20128/v1"}},
			"bob":   {{ID: "bobgw", Name: "mine", BaseURL: "http://localhost:9999/v1"}},
		},
		keys: map[string]string{"gw": "secret"},
	}

	return NewService(store, owners{"p1": "alice", "p2": "bob"}, providers), store, providers
}

func TestOnlyTheOwnerChoosesAndOnlyFromTheirOwnGateways(t *testing.T) {
	ctx := context.Background()
	svc, store, _ := newService()

	if err := svc.Set(ctx, "alice", "p1", "gw"); err != nil {
		t.Fatal(err)
	}

	if err := svc.Set(ctx, "alice", "p1", "bobgw"); !errors.Is(err, domain.ErrIneligible) {
		t.Errorf("someone else's provider: %v", err)
	}

	if err := svc.Set(ctx, "bob", "p1", "bobgw"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("not the project's owner: %v", err)
	}

	if store["p1"] != "gw" {
		t.Errorf("the refused choices must not be saved: %v", store)
	}
}

func TestGrantNeedsAChoiceThatIsStillValid(t *testing.T) {
	ctx := context.Background()
	svc, store, providers := newService()

	if _, _, ok := svc.Grant(ctx, "p1"); ok {
		t.Fatal("nothing is granted before the owner chooses")
	}

	store["p1"] = "gw"

	target, env, ok := svc.Grant(ctx, "p1")
	if !ok || target != "localhost:20128" ||
		!slices.Equal(env, []string{"OPENAI_BASE_URL=http://localhost:20128/v1", "OPENAI_API_KEY=secret"}) {
		t.Fatalf("grant: %q %v %v", target, env, ok)
	}

	// A hand-written row for a provider that is not the owner's grants nothing.
	store["p2"] = "gw"
	if _, _, ok = svc.Grant(ctx, "p2"); ok {
		t.Error("another person's provider must not be granted")
	}

	// A removed or disabled provider ends the grant.
	providers.options["alice"] = nil
	if _, _, ok = svc.Grant(ctx, "p1"); ok {
		t.Error("a removed provider leaves no grant")
	}

	if view, err := svc.Get(ctx, "alice", "p1"); err != nil || view.Selected != "" {
		t.Errorf("a choice that is no longer valid shows as none: %+v %v", view, err)
	}
}

func TestGrantRefusesAnAddressOffThisComputer(t *testing.T) {
	svc, store, providers := newService()
	providers.options["alice"] = []domain.Choice{{ID: "gw", BaseURL: "http://example.com:20128/v1"}}
	store["p1"] = "gw"

	if _, _, ok := svc.Grant(context.Background(), "p1"); ok {
		t.Error("only a gateway on this computer can be granted")
	}
}

func TestClearEndsTheGrant(t *testing.T) {
	ctx := context.Background()
	svc, store, _ := newService()
	store["p1"] = "gw"

	if err := svc.Clear(ctx, "alice", "p1"); err != nil {
		t.Fatal(err)
	}

	if _, _, ok := svc.Grant(ctx, "p1"); ok {
		t.Error("cleared means no grant")
	}
}
