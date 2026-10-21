package http

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type SharedSettingsHandler struct {
	service *application.SharedSettingsService
}

func NewSharedSettingsHandler(service *application.SharedSettingsService) *SharedSettingsHandler {
	return &SharedSettingsHandler{service: service}
}

func (h *SharedSettingsHandler) Mount(auth, sameOrigin fiber.Handler, access ...fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		if len(access) > 1 {
			app.Get("/api/settings/database", auth, access[0], h.get)
			app.Put("/api/settings/database", auth, sameOrigin, access[1], h.put)
			return
		}
		app.Get("/api/settings/database", auth, h.get)
		app.Put("/api/settings/database", auth, sameOrigin, h.put)
	}
}

type sharedSettingsBody struct {
	Host             string `json:"host"`
	Database         string `json:"database"`
	User             string `json:"user"`
	Password         string `json:"password"`
	SSLMode          string `json:"sslMode"`
	CACert           string `json:"caCert"`
	ClientCert       string `json:"clientCert"`
	ClientKey        string `json:"clientKey"`
	RemoveCACert     bool   `json:"removeCaCert"`
	RemoveClientCert bool   `json:"removeClientCert"`
	Port             int    `json:"port"`
}

type sharedSettingsResponse struct {
	Host          string `json:"host"`
	Database      string `json:"database"`
	User          string `json:"user"`
	SSLMode       string `json:"sslMode"`
	Port          int    `json:"port"`
	PasswordSet   bool   `json:"passwordSet"`
	Configured    bool   `json:"configured"`
	CACertSet     bool   `json:"caSet"`
	ClientCertSet bool   `json:"clientCertSet"`
	ClientKeySet  bool   `json:"clientKeySet"`
}

func (h *SharedSettingsHandler) get(c fiber.Ctx) error {
	view, err := h.service.Get(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
	response := sharedResponse(view)
	if httpserver.Role(c) != "admin" {
		return c.JSON(fiber.Map{"configured": response.Configured})
	}
	return c.JSON(response)
}

func (h *SharedSettingsHandler) put(c fiber.Ctx) error {
	var body sharedSettingsBody
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}
	view, err := h.service.SaveWithRemovals(c.Context(), domain.SharedSettings{
		Host: body.Host, Port: body.Port, Database: body.Database, User: body.User,
		Password: body.Password, SSLMode: body.SSLMode,
		CACert: body.CACert, ClientCert: body.ClientCert, ClientKey: body.ClientKey,
	}, body.RemoveCACert, body.RemoveClientCert)
	if errors.Is(err, domain.ErrInvalidSharedSettings) || errors.Is(err, domain.ErrInvalidPEM) ||
		errors.Is(err, application.ErrSharedPasswordRequired) {
		return c.Status(http.StatusBadRequest).JSON(httpserver.Failure("invalid_settings"))
	}
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
	return c.JSON(sharedResponse(view))
}

func sharedResponse(view application.SharedSettingsView) sharedSettingsResponse {
	return sharedSettingsResponse{
		Host: view.Host, Port: view.Port, Database: view.Database,
		User: view.User, SSLMode: view.SSLMode,
		PasswordSet: view.PasswordSet, Configured: view.Configured,
		CACertSet: view.CACertSet, ClientCertSet: view.ClientCertSet,
		ClientKeySet: view.ClientKeySet,
	}
}
