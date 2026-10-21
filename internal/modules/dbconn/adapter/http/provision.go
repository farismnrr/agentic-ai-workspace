package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type provisionBody struct {
	Kind string `json:"kind"`
}

func (h *Handler) provisionShared(c fiber.Ctx) error {
	return h.provision(c, domain.Postgres)
}

func (h *Handler) provisionKind(c fiber.Ctx) error {
	kind := domain.Postgres
	if len(c.Body()) > 0 {
		var body provisionBody
		if err := c.Bind().JSON(&body); err != nil {
			return c.Status(http.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
		}
		if body.Kind != "" {
			kind = domain.Kind(body.Kind)
		}
	}
	if kind != domain.Postgres && kind != domain.MySQL && kind != domain.MongoDB {
		return c.Status(http.StatusBadRequest).JSON(httpserver.Failure("invalid_kind"))
	}
	return h.provision(c, kind)
}

func (h *Handler) provision(c fiber.Ctx, kind domain.Kind) error {
	ctx, cancel := context.WithTimeout(c.Context(), sharedProvisionTimeout)
	defer cancel()
	item, err := h.service.Provision(ctx, httpserver.UserID(c), c.Params("id"), kind)
	if err == nil {
		return c.JSON(item)
	}
	var failure application.ProvisionError
	if errors.As(err, &failure) {
		return c.Status(provisionStatus(failure)).JSON(httpserver.Failure(string(failure.Category)))
	}
	return fail(c, err)
}

func provisionStatus(failure application.ProvisionError) int {
	switch failure.Category {
	case application.ProvisionInProgress, application.ManualConnectionExists, application.EnvironmentNameInUse:
		return http.StatusConflict
	case application.AdminNotConfigured, application.KindNotConfigured, application.AdminURLInvalid:
		return http.StatusServiceUnavailable
	case application.AdminRejected, application.InsufficientPrivileges:
		return http.StatusBadGateway
	case application.ProvisionFailed, application.CompensationFailed:
		if failure.Remote {
			return http.StatusBadGateway
		}
		return http.StatusInternalServerError
	case application.LocalSaveFailed, application.EnvironmentWriteFailed:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
