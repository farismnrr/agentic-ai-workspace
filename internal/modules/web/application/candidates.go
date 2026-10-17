package application

import (
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

const (
	candidateTTL = time.Hour
	// candidateCap bounds what is remembered per project.
	candidateCap = 60
)

// candidates remembers the pictures a project's latest searches returned, so a download can only fetch one of
// those: it carries a known licence, and the model cannot make the server fetch an address of its own choosing.
type candidates struct {
	now   func() time.Time
	byURL map[string]map[string]candidate
	mu    sync.Mutex
}

type candidate struct {
	at    time.Time
	image domain.Image
}

func newCandidates(now func() time.Time) *candidates {
	return &candidates{now: now, byURL: map[string]map[string]candidate{}}
}

func (c *candidates) remember(projectID string, images []domain.Image) {
	c.mu.Lock()
	defer c.mu.Unlock()

	known := c.byURL[projectID]
	if known == nil {
		known = map[string]candidate{}
		c.byURL[projectID] = known
	}

	now := c.now()
	for address, old := range known {
		if now.Sub(old.at) > candidateTTL {
			delete(known, address)
		}
	}

	for _, image := range images {
		if len(known) >= candidateCap {
			break
		}

		known[image.URL] = candidate{image: image, at: now}
	}
}

func (c *candidates) find(projectID, address string) (domain.Image, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	found, ok := c.byURL[projectID][address]
	if !ok || c.now().Sub(found.at) > candidateTTL {
		return domain.Image{}, false
	}

	return found.image, true
}
