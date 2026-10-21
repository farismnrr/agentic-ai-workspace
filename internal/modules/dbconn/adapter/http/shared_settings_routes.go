package http

import (
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

func validSSLMode(mode string) bool {
	for _, candidate := range domain.SSLModes {
		if mode == candidate {
			return true
		}
	}
	return false
}

func legacyTLS(body sharedSettingsBody, clientCertSet bool) string {
	if body.SSLMode == "disable" {
		return "none"
	}
	if clientCertSet {
		return "full"
	}
	if body.SSLMode != "" {
		return "ca"
	}
	return "none"
}

func postgresSSLMode(tlsMode string) string {
	if tlsMode == "none" {
		return "disable"
	}
	return "verify-full"
}

func legacySharedResponse(response sharedSettingsResponse) fiber.Map {
	return fiber.Map{
		"kind": response.Kind, "host": response.Host, "port": response.Port,
		"database": response.Database, "user": response.User, "tls": response.TLS,
		"sslMode": response.SSLMode, "passwordSet": response.PasswordSet,
		"caSet": response.CACertSet, "clientCertSet": response.ClientCertSet,
		"clientKeySet": response.ClientKeySet, "configured": response.Configured,
	}
}

func (h *SharedSettingsHandler) kinds(c fiber.Ctx) error {
	kinds := make([]fiber.Map, 0, len(domain.SharedKinds))
	for _, kind := range domain.SharedKinds {
		var configured bool
		var err error
		if h.availability != nil {
			configured, err = h.availability.ProvisioningKind(c.Context(), domain.Kind(kind))
		} else {
			configured, err = h.service.Configured(c.Context(), kind)
		}
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(httpserver.Failure("internal"))
		}
		kinds = append(kinds, fiber.Map{
			"kind": kind, "configured": configured, "admin": httpserver.Role(c) == "admin",
		})
	}
	return c.JSON(kinds)
}
