package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const connectionTestTimeout = 10 * time.Second

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Mount(auth, sameOrigin, testLimit fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/projects/:id/db", auth, h.get)
		app.Put("/api/projects/:id/db", auth, sameOrigin, h.put)
		app.Delete("/api/projects/:id/db", auth, sameOrigin, h.remove)
		app.Post("/api/projects/:id/db/test", auth, sameOrigin, testLimit, h.test)
	}
}

func (h *Handler) get(c fiber.Ctx) error {
	item, err := h.service.Get(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(item)
}

func (h *Handler) put(c fiber.Ctx) error {
	var body connectionBody
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}
	item, err := h.service.Save(c.Context(), httpserver.UserID(c), c.Params("id"), body.connection())
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(item)
}

func (h *Handler) remove(c fiber.Ctx) error {
	if err := h.service.Delete(c.Context(), httpserver.UserID(c), c.Params("id")); err != nil {
		return fail(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) test(c fiber.Ctx) error {
	var submitted *domain.Connection
	if len(c.Body()) != 0 {
		var body connectionBody
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(http.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
		}
		if body.Type != "" {
			conn := body.connection()
			submitted = &conn
		}
	}
	ctx, cancel := context.WithTimeout(c.Context(), connectionTestTimeout)
	defer cancel()
	err := h.service.Test(ctx, httpserver.UserID(c), c.Params("id"), submitted)
	if err == nil {
		return c.JSON(testResult{OK: true, Message: "Connection successful."})
	}
	var failure domain.ProbeError
	if errors.As(err, &failure) {
		return c.JSON(testResult{Category: failure.Category, Message: failureMessage(failure.Category)})
	}
	return fail(c, err)
}

type connectionBody struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
	EnvName  string `json:"envName"`
	Port     int    `json:"port"`
	TLS      bool   `json:"tls"`
}

func (b connectionBody) connection() domain.Connection {
	return domain.Connection{
		Kind: domain.Kind(b.Type), Host: b.Host, Port: b.Port, Database: b.Database,
		User: b.User, Password: b.Password, TLS: b.TLS, EnvName: b.EnvName,
	}
}

type testResult struct {
	Category domain.FailureCategory `json:"category,omitempty"`
	Message  string                 `json:"message"`
	OK       bool                   `json:"ok"`
}

func failureMessage(category domain.FailureCategory) string {
	switch category {
	case domain.HostNotFound:
		return "The database host could not be found."
	case domain.Refused:
		return "The database refused the connection."
	case domain.TimedOut:
		return "The connection took too long."
	case domain.AuthFailed:
		return "The database rejected the login."
	case domain.TLSFailed:
		return "A secure connection could not be established."
	case domain.Blocked:
		return "This address is not allowed for a connection test."
	case domain.Other:
		return "The database connection could not be verified."
	default:
		return "The database connection could not be verified."
	}
}

func fail(c fiber.Ctx, err error) error {
	if errors.Is(err, domain.ErrMissing) {
		return c.Status(http.StatusNotFound).JSON(httpserver.Failure("not_found"))
	}
	if errors.Is(err, domain.ErrInvalid) {
		return c.Status(http.StatusBadRequest).JSON(httpserver.Failure("invalid_connection"))
	}
	return c.Status(http.StatusInternalServerError).JSON(httpserver.Failure("internal"))
}
