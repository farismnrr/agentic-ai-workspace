// Package http serves the GitHub connection endpoints.
package http

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const settingsPage = "/app/settings/integrations"

// Handler serves a person's GitHub connection. The person is always the signed-in one; nothing in a
// request can name another.
type Handler struct {
	service *application.Service
	appSlug string
}

// Limits guard the endpoints that talk to GitHub.
type Limits struct {
	Start    fiber.Handler
	Callback fiber.Handler
}

// NewHandler builds the GitHub HTTP adapter; appSlug builds the link for choosing repositories.
func NewHandler(service *application.Service, appSlug string) *Handler {
	return &Handler{service: service, appSlug: appSlug}
}

// Mount registers the routes. The callback is a plain browser navigation back from GitHub, so it
// has no same-origin check; its state is single use and tied to the signed-in person instead.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler, limits Limits) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/integrations/github", auth, h.status)
		app.Post("/api/integrations/github/start", auth, sameOrigin, limits.Start, h.start)
		app.Get("/api/integrations/github/callback", auth, limits.Callback, h.callback)
		app.Get("/api/integrations/github/repos", auth, h.repos)
		app.Delete("/api/integrations/github", auth, sameOrigin, h.disconnect)
	}
}

type statusPayload struct {
	Login      string `json:"login"`
	Name       string `json:"name"`
	ReposURL   string `json:"reposUrl"`
	Configured bool   `json:"configured"`
	Connected  bool   `json:"connected"`
}

func (h *Handler) status(c fiber.Ctx) error {
	out := statusPayload{Configured: h.service.Enabled()}
	if h.appSlug != "" {
		out.ReposURL = "https://github.com/apps/" + h.appSlug + "/installations/new"
	}

	conn, err := h.service.Status(c.Context(), httpserver.UserID(c))
	if err != nil && !errors.Is(err, domain.ErrNotConnected) {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	if err == nil {
		out.Connected, out.Login, out.Name = true, conn.Profile.Login, conn.Profile.DisplayName()
	}

	return c.JSON(out)
}

func (h *Handler) start(c fiber.Ctx) error {
	target, err := h.service.Start(httpserver.UserID(c))
	if errors.Is(err, domain.ErrNotConfigured) {
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("not_configured"))
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.JSON(fiber.Map{"url": target})
}

func (h *Handler) callback(c fiber.Ctx) error {
	code, state := c.Query("code"), c.Query("state")
	if reason := c.Query("error"); reason != "" || code == "" {
		log.Printf("github: sign-in refused: %q", reason)

		return c.Redirect().To(settingsPage + "?github=failed")
	}

	// GitHub sends people here without a state right after they install or change the app on
	// GitHub, an address it picks itself. Such a code is never trusted: a link could carry someone
	// else's. The person is sent through a sign-in this server started, which GitHub approves at once.
	if state == "" {
		return h.restart(c)
	}

	if err := h.service.Finish(c.Context(), httpserver.UserID(c), code, state); err != nil {
		log.Printf("github: sign-in failed: %v", err)

		return c.Redirect().To(settingsPage + "?github=failed")
	}

	return c.Redirect().To(settingsPage + "?github=connected")
}

func (h *Handler) restart(c fiber.Ctx) error {
	target, err := h.service.Start(httpserver.UserID(c))
	if err != nil {
		log.Printf("github: could not restart sign-in: %v", err)

		return c.Redirect().To(settingsPage + "?github=failed")
	}

	return c.Redirect().To(target)
}

func (h *Handler) disconnect(c fiber.Ctx) error {
	if err := h.service.Disconnect(c.Context(), httpserver.UserID(c)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.SendStatus(fiber.StatusNoContent)
}
