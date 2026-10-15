package application

import (
	"context"
	"time"
)

const (
	// syncEvery is how long a gateway's model list is trusted before it is read again.
	syncEvery = 30 * time.Second
	// syncTimeout keeps a slow or stopped gateway from holding up a page.
	syncTimeout = 4 * time.Second
)

// SyncStale reads again the model list of every gateway that has not been read lately, so a provider added behind
// it (a Claude account connected in 9router, say) appears without anyone pressing Refresh. A gateway that cannot be
// reached keeps the list it has: nothing is marked unverified, since the key was proven when it was added.
// Providers that are not gateways are left alone, because their catalogue rarely changes and is not free to fetch.
func (s *Service) SyncStale(ctx context.Context, userID string) {
	providers, err := s.store.List(ctx, userID)
	if err != nil {
		return
	}

	for _, provider := range providers {
		if provider.Enabled && provider.Verified && provider.Kind.CustomAddress() && s.due(provider.ID) {
			s.sync(ctx, userID, provider.ID)
		}
	}
}

// due reports whether a provider's list is stale, and claims the refresh, so that a burst of page loads
// reads it once.
func (s *Service) due(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if last, ok := s.synced[id]; ok && now.Sub(last) < syncEvery {
		return false
	}

	s.synced[id] = now

	return true
}

func (s *Service) sync(ctx context.Context, userID, id string) {
	ctx, cancel := context.WithTimeout(ctx, syncTimeout)
	defer cancel()

	credential, err := s.Credential(ctx, userID, id)
	if err != nil {
		return
	}

	models, err := s.verifier.Models(ctx, credential)
	if err != nil || len(models) == 0 {
		return
	}

	_ = s.store.SetVerified(ctx, userID, id, true, models)
}
