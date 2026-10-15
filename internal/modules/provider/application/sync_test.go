package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

var errGatewayDown = errors.New("connection refused")

type fakeStore struct{ provider domain.Provider }

func (f *fakeStore) Create(context.Context, domain.Provider, string) error { return nil }

func (f *fakeStore) List(context.Context, string) ([]domain.Provider, error) {
	return []domain.Provider{f.provider}, nil
}

func (f *fakeStore) Find(context.Context, string, string) (domain.Provider, string, error) {
	return f.provider, "sealed", nil
}

func (f *fakeStore) SetVerified(_ context.Context, _, _ string, verified bool, models []domain.ModelInfo) error {
	f.provider.Verified = verified
	f.provider.Models = nil

	for _, model := range models {
		f.provider.Models = append(f.provider.Models, model.Name)
	}

	return nil
}

func (f *fakeStore) SetEnabled(context.Context, string, string, bool) error { return nil }

func (f *fakeStore) SetContextWindow(context.Context, string, string, string, int64) error {
	return nil
}

func (f *fakeStore) Delete(context.Context, string, string) error { return nil }

type fakeCipher struct{}

func (fakeCipher) Encrypt([]byte) (string, error) { return "sealed", nil }
func (fakeCipher) Decrypt(string) ([]byte, error) { return []byte("key"), nil }

type fakeVerifier struct {
	err    error
	models []domain.ModelInfo
	calls  int
}

func (f *fakeVerifier) Models(context.Context, domain.Credential) ([]domain.ModelInfo, error) {
	f.calls++

	return f.models, f.err
}

type fakeIDs struct{}

func (fakeIDs) ID() (string, error) { return "id", nil }

type syncRig struct {
	store    *fakeStore
	verifier *fakeVerifier
	service  *application.Service
	clock    time.Time
}

func newSyncRig(kind domain.Kind) *syncRig {
	rig := &syncRig{
		store: &fakeStore{provider: domain.Provider{
			ID: "p", Kind: kind, Verified: true, Enabled: true, Models: []string{"cx/old"},
		}},
		verifier: &fakeVerifier{models: []domain.ModelInfo{{Name: "cx/old"}, {Name: "cc/new"}}},
		clock:    time.Unix(1_000_000, 0),
	}
	now := func() time.Time { return rig.clock }
	rig.service = application.NewService(rig.store, fakeCipher{}, rig.verifier, fakeIDs{}, now)

	return rig
}

func TestAGatewayModelListIsReadAgainButNotOnEveryPageLoad(t *testing.T) {
	t.Parallel()

	rig := newSyncRig(domain.KindCompatible)
	rig.service.SyncStale(t.Context(), "u1")

	if len(rig.store.provider.Models) != 2 || rig.verifier.calls != 1 {
		t.Fatalf("a new provider must appear: %v, %d reads", rig.store.provider.Models, rig.verifier.calls)
	}

	rig.service.SyncStale(t.Context(), "u1")

	if rig.verifier.calls != 1 {
		t.Fatalf("a second page load right away must not read it again: %d reads", rig.verifier.calls)
	}

	rig.clock = rig.clock.Add(time.Minute)
	rig.service.SyncStale(t.Context(), "u1")

	if rig.verifier.calls != 2 {
		t.Fatalf("a stale list must be read again: %d reads", rig.verifier.calls)
	}
}

func TestOnlyGatewaysAreReadAgain(t *testing.T) {
	t.Parallel()

	rig := newSyncRig(domain.KindOpenAI)
	rig.service.SyncStale(t.Context(), "u1")

	if rig.verifier.calls != 0 {
		t.Fatal("a hosted provider's catalogue must not be fetched on a page load")
	}
}

func TestAnUnreachableGatewayKeepsItsListAndStaysVerified(t *testing.T) {
	t.Parallel()

	rig := newSyncRig(domain.KindCompatible)
	rig.verifier.err = errGatewayDown
	rig.service.SyncStale(t.Context(), "u1")

	if !rig.store.provider.Verified || len(rig.store.provider.Models) != 1 {
		t.Fatalf("a failed read must change nothing: %+v", rig.store.provider)
	}
}
