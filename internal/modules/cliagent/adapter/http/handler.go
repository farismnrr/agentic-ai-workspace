// Package http serves the endpoints behind "sign in with your own plan".
package http

import (
	"context"
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/cliagent/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Service is what the endpoints need from the use cases.
type Service interface {
	Status(ctx context.Context, user string, rt domain.Runtime) (domain.State, error)
	Start(ctx context.Context, user string, rt domain.Runtime) (domain.Prompt, error)
	Submit(ctx context.Context, user string, rt domain.Runtime, code string) error
	Disconnect(ctx context.Context, user string, rt domain.Runtime) error
}

// Handler serves a person's own sign-ins. The person is always the signed-in one.
type Handler struct{ service Service }

// NewHandler builds the adapter.
func NewHandler(service Service) *Handler { return &Handler{service: service} }

// Mount registers the routes; start and code are rate limited because each one runs a container.
func (h *Handler) Mount(auth, sameOrigin, limit fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/subscriptions/:program", auth, h.status)
		app.Post("/api/subscriptions/:program/start", auth, sameOrigin, limit, h.start)
		app.Post("/api/subscriptions/:program/code", auth, sameOrigin, limit, h.code)
		app.Delete("/api/subscriptions/:program", auth, sameOrigin, limit, h.disconnect)
	}
}

type modelPayload struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Efforts []string `json:"efforts"`
}

type statePayload struct {
	Prompt    *domain.Prompt `json:"prompt,omitempty"`
	Models    []modelPayload `json:"models"`
	Connected bool           `json:"connected"`
}

// models are what the person can choose for the program, so the screens never name a model themselves.
func models(rt domain.Runtime) []modelPayload {
	all := domain.Models(rt)
	payload := make([]modelPayload, 0, len(all))

	for _, model := range all {
		payload = append(payload, modelPayload{ID: model.ID, Name: model.Name, Efforts: model.Efforts})
	}

	return payload
}

func (h *Handler) status(c fiber.Ctx) error {
	rt, err := domain.Parse(c.Params("program"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}

	state, err := h.service.Status(c.Context(), httpserver.UserID(c), rt)
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(statePayload{Connected: state.Connected, Prompt: state.Prompt, Models: models(rt)})
}

func (h *Handler) start(c fiber.Ctx) error {
	rt, err := domain.Parse(c.Params("program"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}

	prompt, err := h.service.Start(c.Context(), httpserver.UserID(c), rt)
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(prompt)
}

func (h *Handler) code(c fiber.Ctx) error {
	rt, err := domain.Parse(c.Params("program"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}

	var body struct {
		Code string `json:"code"`
	}

	if err = c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	if err = h.service.Submit(c.Context(), httpserver.UserID(c), rt, body.Code); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) disconnect(c fiber.Ctx) error {
	rt, err := domain.Parse(c.Params("program"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}

	if err = h.service.Disconnect(c.Context(), httpserver.UserID(c), rt); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// fail answers with a stable code the screen can turn into plain words.
func fail(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrBadCode):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(httpserver.Failure("bad_code"))
	case errors.Is(err, domain.ErrNoSignIn):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("no_sign_in"))
	case errors.Is(err, domain.ErrFailed):
		return c.Status(fiber.StatusBadGateway).JSON(httpserver.Failure("sign_in_failed"))
	case errors.Is(err, domain.ErrUnavailable):
		return c.Status(fiber.StatusServiceUnavailable).JSON(httpserver.Failure("unavailable"))
	default:
		log.Printf("subscriptions: %v", err)

		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
}
