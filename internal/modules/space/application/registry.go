package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"

// Registry is the set of spaces this server hosts.
type Registry struct {
	publicURL string
	spaces    []domain.Space
}

// NewRegistry lists the hosted spaces under the public origin.
func NewRegistry(publicURL string, spaces ...domain.Space) *Registry {
	return &Registry{publicURL: publicURL, spaces: spaces}
}

// Spaces returns every hosted space.
func (r *Registry) Spaces() []domain.Space {
	return r.spaces
}

// ResourceURI returns a hosted space's resource identifier.
func (r *Registry) ResourceURI(space domain.Space) string {
	return space.ResourceURI(r.publicURL)
}

// FindByResource returns the space whose resource identifier is uri.
func (r *Registry) FindByResource(uri string) (domain.Space, bool) {
	for _, space := range r.spaces {
		if space.ResourceURI(r.publicURL) == uri {
			return space, true
		}
	}

	return domain.Space{}, false
}

// FindBySlug returns the hosted space with the given slug.
func (r *Registry) FindBySlug(slug string) (domain.Space, bool) {
	for _, space := range r.spaces {
		if space.Slug == slug {
			return space, true
		}
	}

	return domain.Space{}, false
}
