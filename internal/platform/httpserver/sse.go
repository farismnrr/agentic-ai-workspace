package httpserver

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// SSE writes server-sent events; every method reports false once the viewer has gone.
type SSE struct {
	W *bufio.Writer
}

// Event sends one named event with a JSON payload; an id above zero lets a returning viewer resume from it.
func (s SSE) Event(id int64, name string, payload any) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		return true
	}

	if id > 0 {
		if _, err := fmt.Fprintf(s.W, "id: %d\n", id); err != nil {
			return false
		}
	}

	if _, err := fmt.Fprintf(s.W, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return false
	}

	return s.W.Flush() == nil
}

// Comment sends a line proxies and browsers ignore, which keeps an idle stream open.
func (s SSE) Comment(text string) bool {
	if _, err := fmt.Fprintf(s.W, ": %s\n\n", text); err != nil {
		return false
	}

	return s.W.Flush() == nil
}

// StreamHeaders sets what a stream needs so no proxy buffers it.
func StreamHeaders(c fiber.Ctx) {
	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache, no-transform")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("X-Accel-Buffering", "no")
}
