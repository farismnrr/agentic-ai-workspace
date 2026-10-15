package http

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	defaultLimit = 100
	maxLimit     = 200
)

type activityPayload struct {
	At      time.Time `json:"at"`
	ID      string    `json:"id"`
	Space   string    `json:"space"`
	Client  string    `json:"client"`
	Tool    string    `json:"tool"`
	Summary string    `json:"summary"`
	Outcome string    `json:"outcome"`
	Detail  string    `json:"detail"`
	CanUndo bool      `json:"canUndo"`
	Undone  bool      `json:"undone"`
}

// entries lists the signed-in person's latest tool calls.
func (a *API) entries(c fiber.Ctx) error {
	userID, ok := a.signedIn(c)
	if !ok {
		return nil
	}

	limit := min(max(fiber.Query[int](c, "limit", defaultLimit), 1), maxLimit)

	entries, err := a.activity.List(c.Context(), userID, limit)
	if err != nil {
		return fail(c, err)
	}

	payload := make([]activityPayload, 0, len(entries))
	for _, entry := range entries {
		payload = append(payload, activityPayload{
			ID: entry.ID, At: entry.At, Space: entry.Space, Client: entry.Client, Tool: entry.Tool, Summary: entry.Summary,
			Outcome: string(entry.Outcome), Detail: entry.Detail, CanUndo: entry.CanUndo(), Undone: entry.UndoneAt != nil,
		})
	}

	return c.JSON(payload)
}

// undo reverses one of the signed-in person's changes.
func (a *API) undo(c fiber.Ctx) error {
	userID, ok := a.signedIn(c)
	if !ok {
		return nil
	}

	if err := a.activity.Undo(c.Context(), userID, c.Params("id")); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
