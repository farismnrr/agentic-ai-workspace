package application_test

import (
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

func TestExistingLegacy9routerCannotUseCredentialsOrModels(t *testing.T) {
	t.Parallel()

	rig := newSyncRig(domain.KindCompatible)
	rig.store.provider.Name = "9router"
	rig.store.provider.BaseURL = "http://localhost:20128/v1"
	rig.service.SyncStale(t.Context(), "owner")
	if rig.verifier.calls != 0 {
		t.Fatal("legacy relay must not be contacted during model sync")
	}
	if _, err := rig.service.Credential(t.Context(), "owner", "p"); !errors.Is(err, domain.ErrAccountRelay) {
		t.Fatalf("legacy relay credentials were exposed: %v", err)
	}
	if err := rig.service.SetEnabled(t.Context(), "owner", "p", true); !errors.Is(err, domain.ErrAccountRelay) {
		t.Fatalf("legacy relay could be re-enabled: %v", err)
	}
	providers, err := rig.service.List(t.Context(), "owner")
	if err != nil || len(providers) != 1 || providers[0].Enabled {
		t.Fatalf("legacy relay visible as active: %v, %v", providers, err)
	}
	models, err := rig.service.Models(t.Context(), "owner")
	if err != nil || len(models) != 0 {
		t.Fatalf("legacy relay still offers models: %v, %v", models, err)
	}
}
