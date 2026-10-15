package bootstrap

import (
	"context"
	"errors"
	"fmt"

	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	codingmcp "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/adapter/mcp"
	codingdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	spacehttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/adapter/http"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
)

// The bridges below are the only place features meet: each adapts one
// feature's application API to the small interface another feature declared.

// sessions answers "who is signed in?" for the oauth and space adapters.
type sessions struct {
	account *accountapp.Service
}

func (s sessions) UserID(ctx context.Context, token string) (string, error) {
	user, err := s.account.Authenticate(ctx, token)
	if err != nil {
		return "", fmt.Errorf("authenticate session: %w", err)
	}

	return user.ID, nil
}

// eraser lets a feature's delete-everything use case run when an account is deleted.
type eraser func(ctx context.Context, userID string) error

func (e eraser) Erase(ctx context.Context, userID string) error { return e(ctx, userID) }

// directory lets the authorization server read profile claims.
type directory struct {
	account *accountapp.Service
}

func (d directory) Profile(ctx context.Context, userID string) (oauthapp.Profile, error) {
	user, err := d.account.Profile(ctx, userID)
	if err != nil {
		return oauthapp.Profile{}, fmt.Errorf("load profile: %w", err)
	}

	return oauthapp.Profile{ID: user.ID, Name: user.Name, Email: user.Email}, nil
}

// catalog presents the hosted spaces to the authorization server as resources.
type catalog struct {
	registry *spaceapp.Registry
}

func (c catalog) Find(uri string) (oauthapp.Resource, bool) {
	space, ok := c.registry.FindByResource(uri)
	if !ok {
		return oauthapp.Resource{}, false
	}

	return resourceOf(c.registry, space), true
}

func (c catalog) All() []oauthapp.Resource {
	spaces := c.registry.Spaces()
	resources := make([]oauthapp.Resource, 0, len(spaces))

	for _, space := range spaces {
		resources = append(resources, resourceOf(c.registry, space))
	}

	return resources
}

// verifier lets spaces check bearer tokens against the authorization server.
type verifier struct {
	oauth *oauthapp.Service
}

func (v verifier) Verify(ctx context.Context, token string) (spacehttp.Access, error) {
	info, err := v.oauth.VerifyAccessToken(ctx, token)
	if err != nil {
		return spacehttp.Access{}, fmt.Errorf("verify access token: %w", err)
	}

	return spacehttp.Access{
		UserID: info.UserID, Resource: info.Resource, Client: info.ClientName, Scopes: info.Scopes,
		ExpiresAt: unixTime(info.ExpiresAt),
	}, nil
}

// gate lets the coding tools ask whether a person switched a tool on.
type gate struct {
	permissions *spaceapp.Permissions
}

func (g gate) Allowed(ctx context.Context, userID, tool string) (bool, error) {
	enabled, err := g.permissions.Enabled(ctx, userID, "coding", codingdomain.SwitchOf(tool))
	if err != nil {
		return false, fmt.Errorf("check tool switch: %w", err)
	}

	return enabled, nil
}

// recorder lets the coding tools write to the activity log.
type recorder struct {
	log *spaceapp.ActivityLog
}

func (r recorder) Record(ctx context.Context, entry codingmcp.Entry) error {
	_, err := r.log.Record(ctx, spacedomain.Activity{
		UserID: entry.UserID, Space: "coding", Client: entry.Client, Tool: entry.Tool, Summary: entry.Summary,
		Detail: entry.Detail, Outcome: spacedomain.Outcome(entry.Outcome), Undo: wrapUndo(entry.Project, entry.Undo),
	})
	if err != nil {
		return fmt.Errorf("record activity: %w", err)
	}

	return nil
}

// undoer lets the activity log reverse a change recorded by the coding tools.
type undoer struct {
	tools *agentTools
}

// Undo reverses a change inside the project it was made in. The person is whoever the entry
// belongs to, and projects are only ever reached through that person's own log.
func (u undoer) Undo(_ context.Context, _ string, blob []byte) error {
	undo, err := unwrapUndo(blob)
	if err != nil {
		return err
	}

	change, err := codingdomain.ParseChange(undo.Change)
	if err != nil {
		return spacedomain.ErrNotUndoable
	}

	services, err := u.tools.services(undo.Project)
	if err != nil {
		return fmt.Errorf("open project: %w", err)
	}

	if err := services.Editor.Undo(change); err != nil {
		if errors.Is(err, codingdomain.ErrChanged) {
			return fmt.Errorf("%w: %w", spacedomain.ErrUndoRefused, err)
		}

		return fmt.Errorf("undo change: %w", err)
	}

	return nil
}
