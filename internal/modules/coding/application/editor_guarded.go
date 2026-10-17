package application

import (
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

// WriteIfUnchanged saves a person's edit of a file, but only if the file still has the content they opened:
// baseHash is the ContentHash they were shown, and empty means the file must not exist yet. The agent may have
// changed it meanwhile, and a quiet overwrite would lose its work. The old content is backed up like any change.
func (e *Editor) WriteIfUnchanged(raw, content, baseHash string) (Applied, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rel, err := writablePath(raw)
	if err != nil {
		return Applied{}, err
	}

	if len(content) > domain.MaxWriteBytes {
		return Applied{}, domain.ErrTooLarge
	}

	previous, existed, err := e.previous(rel)
	if err != nil {
		return Applied{}, err
	}

	if (!existed && baseHash != "") || (existed && domain.ContentHash(previous) != baseHash) {
		return Applied{}, domain.ErrStale
	}

	change, err := e.apply(rel, domain.OpWrite, previous, []byte(content), existed)
	if err != nil {
		return Applied{}, err
	}

	return Applied{Detail: "saved", Change: change}, nil
}
