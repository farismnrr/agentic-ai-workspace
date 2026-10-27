package http

import (
	"bufio"
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// resumeFrom is where a returning viewer left off: the browser sends Last-Event-ID when it reconnects.
func resumeFrom(c fiber.Ctx) int64 {
	raw := c.Get("Last-Event-ID")
	if raw == "" {
		raw = c.Query("after")
	}

	seq, _ := strconv.ParseInt(raw, 10, 64)

	return max(seq, 0)
}

// taskStream follows one task: its thread entries as they happen and the model's text as it is written.
func (h *Handler) taskStream(c fiber.Ctx) error {
	userID, id := httpserver.UserID(c), c.Params("id")
	if _, err := h.service.Get(c.Context(), userID, id); err != nil {
		return fail(c, err)
	}

	cursor := resumeFrom(c)
	sub := h.hub.Subscribe(application.TaskTopic(id))

	httpserver.StreamHeaders(c)

	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer sub.Close()

		out := httpserver.SSE{W: w}
		if out.Comment("ready") && h.push(out, userID, id, &cursor, true) {
			h.followTask(out, sub, userID, id, &cursor)
		}
	})
}

func (h *Handler) followTask(out httpserver.SSE, sub *application.Subscription, userID, id string, cursor *int64) {
	ticker := time.NewTicker(application.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case update, open := <-sub.C:
			if !open || !h.deliver(out, update, userID, id, cursor) {
				return
			}
		case <-ticker.C:
			// A heartbeat also repairs any wake-up that was dropped under load.
			if !out.Comment("ping") || !h.push(out, userID, id, cursor, false) {
				return
			}
		}
	}
}

func (h *Handler) deliver(out httpserver.SSE, update application.Update, userID, id string, cursor *int64) bool {
	if update.Delta != "" {
		return out.Event(0, "delta", map[string]string{"text": update.Delta})
	}

	return h.push(out, userID, id, cursor, true)
}

// push sends the task and any thread entries after the cursor. Without force it stays quiet when nothing is new.
func (h *Handler) push(out httpserver.SSE, userID, id string, cursor *int64, force bool) bool {
	ctx := context.Background()

	task, err := h.service.Get(ctx, userID, id)
	if err != nil {
		return false
	}

	events, err := h.service.Events(ctx, userID, id, *cursor)
	if err != nil {
		return false
	}

	if len(events) == 0 && !force {
		return true
	}

	payload := make([]eventPayload, 0, len(events))
	for _, event := range events {
		payload = append(payload, eventPayload{Event: event, CanUndo: event.CanUndo()})
		*cursor = max(*cursor, event.Version)
	}

	return out.Event(*cursor, "update", map[string]any{"task": toPayload(task), "events": payload})
}

// workspaceStream tells a viewer that something they own changed, so lists can refresh at once.
func (h *Handler) workspaceStream(c fiber.Ctx) error {
	sub := h.hub.Subscribe(application.UserTopic(httpserver.UserID(c)))

	httpserver.StreamHeaders(c)

	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer sub.Close()

		out := httpserver.SSE{W: w}
		if out.Comment("ready") {
			followWorkspace(out, sub)
		}
	})
}

func followWorkspace(out httpserver.SSE, sub *application.Subscription) {
	ticker := time.NewTicker(application.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case update, open := <-sub.C:
			if !open || !out.Event(0, "changed", map[string]string{"taskId": update.TaskID}) {
				return
			}
		case <-ticker.C:
			if !out.Comment("ping") {
				return
			}
		}
	}
}
