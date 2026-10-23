package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

// Permissions decides which tools a person has switched on.
type Permissions struct {
	registry *Registry
	store    PermissionStore
	now      func() time.Time
}

// NewPermissions wires the permission use cases.
func NewPermissions(registry *Registry, store PermissionStore, now func() time.Time) *Permissions {
	return &Permissions{registry: registry, store: store, now: now}
}

// For returns every tool of a space with its effective on/off state.
func (p *Permissions) For(ctx context.Context, userID string, space domain.Space) (map[string]bool, error) {
	chosen, err := p.store.ToolStates(ctx, userID, space.Slug)
	if err != nil {
		return nil, fmt.Errorf("load tool states: %w", err)
	}

	states := make(map[string]bool, len(space.Tools))

	for _, tool := range space.Tools {
		enabled, ok := chosen[tool.Name]
		states[tool.Name] = enabled || (!ok && tool.DefaultOn)
	}

	return states, nil
}

// Enabled reports whether a person may use a tool right now.
func (p *Permissions) Enabled(ctx context.Context, userID, slug, tool string) (bool, error) {
	space, ok := p.registry.FindBySlug(slug)
	if !ok {
		return false, domain.ErrUnknownSpace
	}

	states, err := p.For(ctx, userID, space)
	if err != nil {
		return false, err
	}

	enabled, known := states[tool]
	if !known {
		return false, domain.ErrUnknownTool
	}

	return enabled, nil
}

// Set switches one tool on or off for a person.
func (p *Permissions) Set(ctx context.Context, userID, slug, tool string, enabled bool) error {
	space, ok := p.registry.FindBySlug(slug)
	if !ok {
		return domain.ErrUnknownSpace
	}

	if _, known := space.FindTool(tool); !known {
		return domain.ErrUnknownTool
	}

	if err := p.store.SetTool(ctx, userID, slug, tool, enabled, p.now()); err != nil {
		return fmt.Errorf("save tool state: %w", err)
	}

	return nil
}

// SkipApprovals reports whether a person lets the assistant run the enabled tools of a space without
// asking. A tool that is switched off stays off either way.
func (p *Permissions) SkipApprovals(ctx context.Context, userID, slug string) (bool, error) {
	if _, ok := p.registry.FindBySlug(slug); !ok {
		return false, domain.ErrUnknownSpace
	}

	skip, err := p.store.SkipApprovals(ctx, userID, slug)
	if err != nil {
		return false, fmt.Errorf("load approval setting: %w", err)
	}

	return skip, nil
}

// SetSkipApprovals turns "skip approvals" on or off for one space of a person.
func (p *Permissions) SetSkipApprovals(ctx context.Context, userID, slug string, skip bool) error {
	space, ok := p.registry.FindBySlug(slug)
	if !ok {
		return domain.ErrUnknownSpace
	}
	if !space.SupportsApprovals {
		return domain.ErrApprovalsUnsupported
	}

	if err := p.store.SetSkipApprovals(ctx, userID, slug, skip, p.now()); err != nil {
		return fmt.Errorf("save approval setting: %w", err)
	}

	return nil
}
