package http

import (
	"bufio"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const heartbeat = 15 * time.Second

func isBinary(err error) bool { return errors.Is(err, domain.ErrBinary) }

// stream tells a page that the project's files may have changed, so it reads them again. It carries no file
// content: the page asks for what it shows.
func (h *Handler) stream(c fiber.Ctx) error {
	if _, err := h.open(c); err != nil {
		return fail(c, err)
	}

	signals, stop := h.changes.Follow(c.Params("id"))

	httpserver.StreamHeaders(c)

	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer stop()

		out := httpserver.SSE{W: w}
		if out.Comment("ready") {
			follow(out, signals)
		}
	})
}

// follow sends a changed event for every signal, and a ping now and then so the stream stays open.
func follow(out httpserver.SSE, signals <-chan struct{}) {
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()

	for {
		select {
		case _, open := <-signals:
			if !open || !out.Event(0, "changed", map[string]bool{"files": true}) {
				return
			}
		case <-ticker.C:
			if !out.Comment("ping") {
				return
			}
		}
	}
}
