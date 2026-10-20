package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"sync"

	cliagentsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/adapter/sqlite"
	cliagentdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
	providerdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

var programs = []cliagentdomain.Runtime{cliagentdomain.Claude, cliagentdomain.ChatGPT}

// planCatalog lists the plans a person signed in to among their providers, each with the models its program takes.
type planCatalog struct {
	plans    *plans
	switches *cliagentsqlite.Switches
}

// Connected lists the plans the person is signed in to. A plan that cannot be checked right now is left out.
func (c planCatalog) Connected(ctx context.Context, userID string) []providerdomain.Provider {
	found := make([]*providerdomain.Provider, len(programs))

	var wg sync.WaitGroup

	for i, rt := range programs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			state, err := c.plans.service.Status(ctx, userID, rt)
			if err != nil || !state.Connected {
				return
			}

			enabled, err := c.switches.Enabled(ctx, userID, string(rt))
			if err != nil {
				enabled = true
			}

			plan := planProvider(rt, enabled, c.plans.service.Models(ctx, userID, rt))
			found[i] = &plan
		}()
	}

	wg.Wait()

	connected := []providerdomain.Provider{}

	for _, plan := range found {
		if plan != nil {
			connected = append(connected, *plan)
		}
	}

	return connected
}

func planProvider(rt cliagentdomain.Runtime, enabled bool, models []cliagentdomain.Model) providerdomain.Provider {
	plan := providerdomain.Provider{
		ID: providerdomain.PlanPrefix + string(rt), Name: programNames[string(rt)], Kind: providerdomain.KindPlan,
		Verified: true, Enabled: enabled, Models: make([]string, 0, len(models)),
		Catalog: make(map[string]providerdomain.ModelInfo, len(models)),
	}

	for _, model := range models {
		plan.Models = append(plan.Models, model.ID)
		plan.Catalog[model.ID] = providerdomain.ModelInfo{
			Name: model.Name, ContextWindow: model.ContextWindow,
			Thinking: providerdomain.Thinking{
				Reported: true, Reasoning: len(model.Efforts) > 0, CanDisable: true, Levels: model.Efforts,
			},
		}
	}

	return plan
}

// SetEnabled switches one of the person's plans on or off.
func (c planCatalog) SetEnabled(ctx context.Context, userID, id string, enabled bool) error {
	rt, err := cliagentdomain.Parse(strings.TrimPrefix(id, providerdomain.PlanPrefix))
	if err != nil {
		return fmt.Errorf("find plan: %w", providerdomain.ErrNotFound)
	}

	if err = c.switches.Set(ctx, userID, string(rt), enabled); err != nil {
		return fmt.Errorf("switch plan: %w", err)
	}

	return nil
}

// SignOut has the program forget the person's plan.
func (c planCatalog) SignOut(ctx context.Context, userID, id string) error {
	rt, err := cliagentdomain.Parse(strings.TrimPrefix(id, providerdomain.PlanPrefix))
	if err != nil {
		return fmt.Errorf("find plan: %w", providerdomain.ErrNotFound)
	}

	if err = c.plans.service.Disconnect(ctx, userID, rt); err != nil {
		return fmt.Errorf("sign out: %w", err)
	}

	return nil
}
