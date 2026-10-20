package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

type fakePlans struct {
	switched map[string]bool
	signedIn []string
	windows  map[string]int64
	plans    []domain.Provider
}

func (f *fakePlans) Connected(context.Context, string) []domain.Provider { return f.plans }

func (f *fakePlans) SetEnabled(_ context.Context, _, id string, enabled bool) error {
	f.switched[id] = enabled

	return nil
}

func (f *fakePlans) SetWindow(_ context.Context, _, id, model string, tokens int64) error {
	f.windows[id+"/"+model] = tokens

	return nil
}

func (f *fakePlans) SignOut(_ context.Context, _, id string) error {
	f.signedIn = append(f.signedIn, id)

	return nil
}

func newPlanRig(enabled bool) (*application.Service, *fakePlans) {
	plans := &fakePlans{switched: map[string]bool{}, windows: map[string]int64{}, plans: []domain.Provider{{
		ID: "plan:chatgpt", Name: "ChatGPT", Kind: domain.KindPlan, Verified: true, Enabled: enabled,
		Models: []string{"gpt-x"},
	}}}
	own := domain.Provider{ID: "own", Name: "Own", Verified: true, Enabled: true, Models: []string{"m"}}
	store := &fakeStore{provider: own}
	service := application.NewService(store, fakeCipher{}, &fakeVerifier{}, fakeIDs{}, time.Now).WithPlans(plans)

	return service, plans
}

func TestPlansAreListedAmongProvidersAndTheirModelsArePickable(t *testing.T) {
	t.Parallel()

	service, _ := newPlanRig(true)

	providers, err := service.List(t.Context(), "u")
	if err != nil || len(providers) != 2 || providers[0].ID != "plan:chatgpt" {
		t.Fatalf("List = %+v, %v", providers, err)
	}

	options, err := service.Models(t.Context(), "u")
	if err != nil || len(options) != 2 || options[0].ProviderID != "plan:chatgpt" || options[0].Model != "gpt-x" {
		t.Fatalf("Models = %+v, %v", options, err)
	}
}

func TestSwitchedOffPlanOffersNoModels(t *testing.T) {
	t.Parallel()

	service, _ := newPlanRig(false)

	options, err := service.Models(t.Context(), "u")
	if err != nil || len(options) != 1 || options[0].ProviderID != "own" {
		t.Fatalf("Models = %+v, %v", options, err)
	}
}

func TestPlanSwitchAndSignOutGoToThePlans(t *testing.T) {
	t.Parallel()

	service, plans := newPlanRig(true)

	if err := service.SetEnabled(t.Context(), "u", "plan:chatgpt", false); err != nil || plans.switched["plan:chatgpt"] {
		t.Fatalf("SetEnabled: %v %v", err, plans.switched)
	}

	if err := service.Remove(t.Context(), "u", "plan:chatgpt"); err != nil || len(plans.signedIn) != 1 {
		t.Fatalf("Remove: %v %v", err, plans.signedIn)
	}

	if _, err := service.Credential(t.Context(), "u", "plan:chatgpt"); err == nil {
		t.Fatal("a plan has no key to hand out")
	}
}

func TestAPlansWindowCanBeSetForAModelItListsOnly(t *testing.T) {
	t.Parallel()

	service, plans := newPlanRig(true)

	if err := service.SetContextWindow(t.Context(), "u", "plan:chatgpt", "gpt-x", 120_000); err != nil {
		t.Fatal(err)
	}

	if plans.windows["plan:chatgpt/gpt-x"] != 120_000 {
		t.Fatalf("windows = %v", plans.windows)
	}

	if err := service.SetContextWindow(t.Context(), "u", "plan:chatgpt", "other", 120_000); err == nil {
		t.Fatal("a model the plan does not list must be refused")
	}
}

func TestAPlansChosenWindowIsTheOneAnswered(t *testing.T) {
	t.Parallel()

	service, plans := newPlanRig(true)
	plans.plans[0].ContextWindows = map[string]int64{"gpt-x": 90_000}

	if got := service.ContextWindow(t.Context(), "u", "plan:chatgpt", "gpt-x"); got != 90_000 {
		t.Fatalf("window = %d, want the chosen 90000", got)
	}
}
