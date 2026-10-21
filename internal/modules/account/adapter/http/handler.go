package http

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler exposes the account use cases over JSON.
type Handler struct {
	service *application.Service
	secure  bool
}

// NewHandler builds the account HTTP adapter. secure marks cookies Secure.
func NewHandler(service *application.Service, secure bool) *Handler {
	return &Handler{service: service, secure: secure}
}

// Mount registers the /api/auth routes; sameOrigin guards the POSTs.
// Limits slow down guessing and mass sign-ups; each is a middleware.
type Limits struct {
	Register   fiber.Handler
	LoginIP    fiber.Handler
	LoginEmail fiber.Handler
}

func (h *Handler) Mount(sameOrigin fiber.Handler, limits Limits) httpserver.Mount {
	return func(app *fiber.App) {
		auth := app.Group("/api/auth", sameOrigin)
		auth.Post("/register", limits.Register, h.register)
		auth.Post("/login", limits.LoginIP, limits.LoginEmail, h.login)
		auth.Post("/logout", h.logout)
		auth.Get("/me", h.me)
		auth.Post("/password", h.changePassword)
		auth.Post("/delete", h.deleteAccount)
	}
}

type userPayload struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type credentials struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) register(c fiber.Ctx) error {
	var body credentials
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	user, session, err := h.service.Register(c.Context(), body.Name, body.Email, body.Password)
	if err != nil {
		return h.fail(c, err)
	}

	h.setSession(c, session.Token, session.ExpiresAt)

	return c.Status(fiber.StatusCreated).JSON(toPayload(user))
}

func (h *Handler) login(c fiber.Ctx) error {
	var body credentials
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	user, session, err := h.service.Login(c.Context(), body.Email, body.Password)
	if err != nil {
		return h.fail(c, err)
	}

	h.setSession(c, session.Token, session.ExpiresAt)

	return c.JSON(toPayload(user))
}

func (h *Handler) logout(c fiber.Ctx) error {
	if token := c.Cookies(httpserver.SessionCookieName); token != "" {
		if err := h.service.Logout(c.Context(), token); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
		}
	}

	h.setSession(c, "", time.Unix(0, 0))

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) me(c fiber.Ctx) error {
	user, err := h.service.Authenticate(c.Context(), c.Cookies(httpserver.SessionCookieName))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(httpserver.Failure("unauthenticated"))
	}

	return c.JSON(toPayload(user))
}

func (h *Handler) setSession(c fiber.Ctx, token string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     httpserver.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func (h *Handler) fail(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidName):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_name"))
	case errors.Is(err, domain.ErrInvalidEmail):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_email"))
	case errors.Is(err, domain.ErrWeakPassword):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("weak_password"))
	case errors.Is(err, domain.ErrEmailTaken):
		return c.Status(fiber.StatusConflict).JSON(httpserver.Failure("email_taken"))
	case errors.Is(err, domain.ErrWrongPassword):
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("wrong_password"))
	case errors.Is(err, domain.ErrInvalidCredentials):
		return c.Status(fiber.StatusUnauthorized).JSON(httpserver.Failure("invalid_credentials"))
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
}

func toPayload(user domain.User) userPayload {
	return userPayload{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role}
}
