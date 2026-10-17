package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type payload struct {
	StartedAt  time.Time `json:"startedAt"`
	ProjectID  string    `json:"projectId"`
	URL        string    `json:"url"`
	Visibility string    `json:"visibility"`
	Status     string    `json:"status"`
	Port       int       `json:"port"`
}

func toPayload(s domain.Site) payload {
	return payload{
		ProjectID: s.ProjectID, URL: s.BasePath() + "/", Visibility: string(s.Visibility),
		Status: string(s.Status), Port: s.Port, StartedAt: s.StartedAt,
	}
}

func (h *Handler) list(c fiber.Ctx) error {
	sites, err := h.service.List(c.Context(), httpserver.UserID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	out := make([]payload, 0, len(sites))
	for _, site := range sites {
		out = append(out, toPayload(site))
	}

	return c.JSON(fiber.Map{"sites": out})
}

func (h *Handler) stop(c fiber.Ctx) error {
	if err := h.service.Stop(c.Context(), httpserver.UserID(c), c.Params("project")); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) start(c fiber.Ctx) error {
	site, err := h.service.Restart(c.Context(), httpserver.UserID(c), c.Params("project"))
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(toPayload(site))
}

func (h *Handler) visibility(c fiber.Ctx) error {
	var in struct {
		Visibility string `json:"visibility"`
	}

	if err := c.Bind().Body(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("bad_request"))
	}

	if err := h.service.SetVisibility(c.Context(), httpserver.UserID(c), c.Params("project"), in.Visibility); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func fail(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	case errors.Is(err, domain.ErrNeverStarted):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("never_started"))
	case errors.Is(err, domain.ErrTooMany), errors.Is(err, domain.ErrServerBusy):
		return c.Status(fiber.StatusTooManyRequests).JSON(httpserver.Failure("too_many_sites"))
	case errors.Is(err, domain.ErrNotStarted), errors.Is(err, domain.ErrNoSandboxes):
		return c.Status(fiber.StatusBadGateway).JSON(httpserver.Failure("not_started"))
	case errors.Is(err, domain.ErrBadChoice):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("bad_visibility"))
	}

	return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
}
