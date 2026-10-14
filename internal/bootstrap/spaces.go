package bootstrap

import (
	"time"

	oauthapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	oauthdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
	spaceapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/application"
	spacedomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
)

// newRegistry lists the spaces this server offers.
func newRegistry(cfg config.Config) *spaceapp.Registry {
	return spaceapp.NewRegistry(cfg.Connector(), codingSpace())
}

func resourceOf(registry *spaceapp.Registry, space spacedomain.Space) oauthapp.Resource {
	scopes := make([]oauthdomain.Scope, 0, len(space.Scopes))
	for _, scope := range space.Scopes {
		scopes = append(scopes, oauthdomain.Scope{ID: scope.ID, Description: scope.Description})
	}

	return oauthapp.Resource{URI: registry.ResourceURI(space), Name: space.Name, Scopes: scopes}
}

func unixTime(seconds int64) time.Time {
	return time.Unix(seconds, 0)
}
