package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type SharedSettingsHandler struct {
	service      *application.SharedSettingsService
	availability kindAvailability
}

type kindAvailability interface {
	ProvisioningKind(context.Context, domain.Kind) (bool, error)
}

func NewSharedSettingsHandler(
	service *application.SharedSettingsService, availability ...kindAvailability,
) *SharedSettingsHandler {
	handler := &SharedSettingsHandler{service: service}
	if len(availability) > 0 {
		handler.availability = availability[0]
	}
	return handler
}

func (h *SharedSettingsHandler) Mount(
	auth, sameOrigin, accessRead, accessWrite fiber.Handler,
) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/databases/kinds", auth, accessRead, h.kinds)
		app.Get("/api/settings/database", auth, accessRead, h.get)
		app.Put("/api/settings/database", auth, sameOrigin, accessWrite, h.put)
		app.Get("/api/settings/database/:kind", auth, accessRead, h.getKind)
		app.Put("/api/settings/database/:kind", auth, sameOrigin, accessWrite, h.putKind)
	}
}

type sharedSettingsBody struct {
	Host             string `json:"host"`
	Database         string `json:"database"`
	User             string `json:"user"`
	Password         string `json:"password"`
	SSLMode          string `json:"sslMode"`
	TLS              string `json:"tls"`
	CACert           string `json:"caCert"`
	ClientCert       string `json:"clientCert"`
	ClientKey        string `json:"clientKey"`
	RemoveCACert     bool   `json:"removeCaCert"`
	RemoveClientCert bool   `json:"removeClientCert"`
	Port             int    `json:"port"`
}

type sharedSettingsResponse struct {
	Kind          string `json:"kind"`
	Host          string `json:"host"`
	Database      string `json:"database"`
	User          string `json:"user"`
	SSLMode       string `json:"-"`
	TLS           string `json:"tls"`
	Port          int    `json:"port"`
	PasswordSet   bool   `json:"passwordSet"`
	Configured    bool   `json:"configured"`
	CACertSet     bool   `json:"caSet"`
	ClientCertSet bool   `json:"clientCertSet"`
	ClientKeySet  bool   `json:"clientKeySet"`
}

func (h *SharedSettingsHandler) get(c fiber.Ctx) error {
	return h.getSettings(c, "postgres", true)
}

func (h *SharedSettingsHandler) getKind(c fiber.Ctx) error {
	return h.getSettings(c, c.Params("kind"), false)
}

func (h *SharedSettingsHandler) getSettings(c fiber.Ctx, kind string, legacy bool) error {
	if !domain.IsSharedKind(kind) {
		return c.SendStatus(http.StatusNotFound)
	}
	view, err := h.service.Get(c.Context(), kind)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
	response := sharedResponse(view)
	if httpserver.Role(c) != "admin" {
		return c.JSON(fiber.Map{"configured": response.Configured})
	}
	if legacy {
		return c.JSON(legacySharedResponse(response))
	}
	return c.JSON(response)
}

func (h *SharedSettingsHandler) put(c fiber.Ctx) error {
	return h.putSettings(c, "postgres", true)
}

func (h *SharedSettingsHandler) putKind(c fiber.Ctx) error {
	return h.putSettings(c, c.Params("kind"), false)
}

func (h *SharedSettingsHandler) putSettings(c fiber.Ctx, kind string, legacy bool) error {
	if !domain.IsSharedKind(kind) {
		return c.SendStatus(http.StatusNotFound)
	}
	var body sharedSettingsBody
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}
	tlsMode, err := h.settingsTLSMode(c, kind, body)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrInvalidSharedSettings) {
			status = http.StatusBadRequest
		}
		return c.Status(status).JSON(httpserver.Failure("invalid_settings"))
	}
	return h.persistSettings(c, kind, legacy, body, tlsMode)
}

func (h *SharedSettingsHandler) persistSettings(
	c fiber.Ctx, kind string, legacy bool, body sharedSettingsBody, tlsMode string,
) error {
	sslMode := body.SSLMode
	if kind == "postgres" {
		sslMode = postgresSSLMode(tlsMode)
	}
	view, err := h.service.SaveWithRemovals(c.Context(), kind, domain.SharedSettings{
		Kind: kind, TLS: tlsMode,
		Host: body.Host, Port: body.Port, Database: body.Database, User: body.User,
		Password: body.Password, SSLMode: sslMode,
		CACert: body.CACert, ClientCert: body.ClientCert, ClientKey: body.ClientKey,
	}, body.RemoveCACert, body.RemoveClientCert)
	if errors.Is(err, domain.ErrInvalidSharedSettings) || errors.Is(err, domain.ErrInvalidPEM) ||
		errors.Is(err, application.ErrSharedPasswordRequired) {
		return c.Status(http.StatusBadRequest).JSON(httpserver.Failure("invalid_settings"))
	}
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}
	response := sharedResponse(view)
	if legacy {
		return c.JSON(legacySharedResponse(response))
	}
	return c.JSON(response)
}

func (h *SharedSettingsHandler) settingsTLSMode(
	c fiber.Ctx, kind string, body sharedSettingsBody,
) (string, error) {
	if body.TLS != "" {
		return body.TLS, nil
	}
	if body.SSLMode != "" && !validSSLMode(body.SSLMode) {
		return "", domain.ErrInvalidSharedSettings
	}
	clientCertSet := body.ClientCert != "" || body.ClientKey != ""
	if clientCertSet || body.RemoveClientCert {
		return legacyTLS(body, clientCertSet), nil
	}
	previous, err := h.service.Get(c.Context(), kind)
	if err != nil {
		return "", fmt.Errorf("read current database TLS settings: %w", err)
	}
	return legacyTLS(body, previous.ClientCertSet), nil
}

func sharedResponse(view application.SharedSettingsView) sharedSettingsResponse {
	return sharedSettingsResponse{
		Kind: view.Kind, Host: view.Host, Port: view.Port, Database: view.Database,
		User: view.User, SSLMode: view.SSLMode, TLS: view.TLS,
		PasswordSet: view.PasswordSet, Configured: view.Configured,
		CACertSet: view.CACertSet, ClientCertSet: view.ClientCertSet,
		ClientKeySet: view.ClientKeySet,
	}
}
