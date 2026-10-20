package application

import (
	"bytes"
	"context"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
)

const catalogTTL = time.Hour

type catalog struct {
	at     time.Time
	models []domain.Model
}

// Models are what a person can choose for a program's model. Claude Code lists none, so those are its
// documented names. Codex prints its own list without needing a sign-in, which is read as it is; when it
// cannot be read, the program's own choice is all there is.
func (s *Service) Models(ctx context.Context, user string, rt domain.Runtime) []domain.Model {
	if rt == domain.Claude {
		return domain.ClaudeModels()
	}

	s.mu.Lock()
	cached := s.catalog
	s.mu.Unlock()

	if cached.models != nil && s.now().Sub(cached.at) < catalogTTL {
		return cached.models
	}

	var printed bytes.Buffer

	ctx, cancel := context.WithTimeout(ctx, checkWait)
	defer cancel()

	prog := programOf(rt)
	if _, err := s.sandbox.Run(ctx, Job{UserID: user, Argv: prog.models, Env: prog.env}, &printed); err != nil {
		return domain.OwnChoice()
	}

	models, err := domain.ParseCodexCatalog(printed.Bytes())
	if err != nil {
		return domain.OwnChoice()
	}

	s.mu.Lock()
	s.catalog = catalog{at: s.now(), models: models}
	s.mu.Unlock()

	return models
}
