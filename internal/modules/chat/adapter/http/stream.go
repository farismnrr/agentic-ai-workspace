package http

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

type sseWriter struct {
	w *bufio.Writer
}

func (s sseWriter) event(name string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return
	}

	_ = s.w.Flush()
}

func streamHeaders(c fiber.Ctx) {
	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache, no-transform")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("X-Accel-Buffering", "no")
}

func (h *Handler) sendStream(c fiber.Ctx, userID, projectID, prompt string) error {
	streamHeaders(c)

	return c.SendStreamWriter(func(w *bufio.Writer) {
		out := sseWriter{w: w}
		onText := func(delta string) {
			out.event("delta", map[string]string{"text": delta})
		}

		userMsg, assistantMsg, err := h.service.Send(context.Background(), userID, projectID, prompt, onText)
		if err != nil {
			out.event("error", map[string]string{"error": err.Error()})
			return
		}

		out.event("done", map[string]any{
			"userMessage":      userMsg,
			"assistantMessage": assistantMsg,
		})
	})
}
