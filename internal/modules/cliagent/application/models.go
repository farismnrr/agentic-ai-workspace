package application

import (
	"bytes"
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
)

const (
	catalogTTL = time.Hour
	retryAfter = time.Minute
)

type catalog struct {
	at     time.Time
	models []domain.Model
}

// Models are what a person can choose for a program's model. Claude Code lists none, so those are its
// documented names. Codex is asked the way it asks itself: the account's list when it can be read, else the list
// built into the program. That list can name a model the account cannot use; the program then says so when it runs.
func (s *Service) Models(ctx context.Context, user string, rt domain.Runtime) []domain.Model {
	if rt == domain.Claude {
		return domain.ClaudeModels()
	}

	k := key{user, rt}

	s.mu.Lock()
	cached := s.catalogs[k]
	s.mu.Unlock()

	if cached.models != nil && s.now().Sub(cached.at) < catalogTTL {
		return cached.models
	}

	models, err := s.readCatalog(ctx, user, rt)
	if err != nil {
		models, err = s.printed(ctx, user, rt, programOf(rt).bundled, nil)
	}

	if err != nil {
		// Do not ask again for a minute: a program that cannot be asked would slow every screen down.
		models = domain.OwnChoice()
		s.remember(k, models, s.now().Add(retryAfter-catalogTTL))

		return models
	}

	s.remember(k, models, s.now())

	return models
}

func (s *Service) remember(k key, models []domain.Model, at time.Time) {
	s.mu.Lock()
	s.catalogs[k] = catalog{at: at, models: models}
	s.mu.Unlock()
}

// readCatalog asks the program for the models of the person's account, which needs its sign-in and its network.
func (s *Service) readCatalog(ctx context.Context, user string, rt domain.Runtime) ([]domain.Model, error) {
	return s.printed(ctx, user, rt, programOf(rt).models, s.hosts[rt])
}

// printed runs the program's command and reads the list of models it prints.
func (s *Service) printed(
	ctx context.Context, user string, rt domain.Runtime, argv, hosts []string,
) ([]domain.Model, error) {
	var out bytes.Buffer

	ctx, cancel := context.WithTimeout(ctx, checkWait)
	defer cancel()

	job := Job{UserID: user, Argv: argv, Env: programOf(rt).env, Hosts: hosts}
	if _, err := s.sandbox.Run(ctx, job, &out); err != nil {
		return nil, domain.ErrUnavailable
	}

	return domain.ParseCodexCatalog(out.Bytes()) //nolint:wrapcheck // already descriptive
}
