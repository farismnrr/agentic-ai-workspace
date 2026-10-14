package http

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves the app's provider endpoints. A key is accepted when adding a
// provider and never returned.
type Handler struct {
	service *application.Service
}

// NewHandler builds the provider HTTP adapter.
func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

// Mount registers the routes; auth guards all, sameOrigin the state-changing ones.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		// Auth is attached per route: a group-level middleware on /api would also
		// intercept routes registered later, such as the bearer-protected MCP endpoints.
		app.Get("/api/providers", auth, h.list)
		app.Post("/api/providers", auth, sameOrigin, h.add)
		app.Post("/api/providers/:id/verify", auth, sameOrigin, h.verify)
		app.Put("/api/providers/:id", auth, sameOrigin, h.setEnabled)
		app.Delete("/api/providers/:id", auth, sameOrigin, h.remove)
		app.Get("/api/models", auth, h.models)
	}
}

type providerPayload struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Name     string   `json:"name"`
	BaseURL  string   `json:"baseUrl"`
	KeyHint  string   `json:"keyHint"`
	Models   []string `json:"models"`
	Verified bool     `json:"verified"`
	Enabled  bool     `json:"enabled"`
}

func toPayload(p domain.Provider) providerPayload {
	return providerPayload{
		ID: p.ID, Kind: string(p.Kind), Name: p.Name, BaseURL: p.BaseURL, KeyHint: p.KeyHint,
		Models: p.Models, Verified: p.Verified, Enabled: p.Enabled,
	}
}

func (h *Handler) list(c fiber.Ctx) error {
	providers, err := h.service.List(c.Context(), httpserver.UserID(c))
	if err != nil {
		return fail(c, err)
	}

	payload := make([]providerPayload, 0, len(providers))
	for _, provider := range providers {
		payload = append(payload, toPayload(provider))
	}

	return c.JSON(payload)
}

func (h *Handler) add(c fiber.Ctx) error {
	var body struct {
		Kind    string `json:"kind"`
		Name    string `json:"name"`
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	provider, err := h.service.Add(c.Context(), httpserver.UserID(c), application.AddInput{
		Kind: domain.Kind(body.Kind), Name: body.Name, BaseURL: body.BaseURL, APIKey: body.APIKey,
	})
	if err != nil {
		return fail(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toPayload(provider))
}

func (h *Handler) verify(c fiber.Ctx) error {
	provider, err := h.service.Verify(c.Context(), httpserver.UserID(c), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}

	return c.JSON(toPayload(provider))
}

func (h *Handler) setEnabled(c fiber.Ctx) error {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	if err := h.service.SetEnabled(c.Context(), httpserver.UserID(c), c.Params("id"), body.Enabled); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) remove(c fiber.Ctx) error {
	if err := h.service.Remove(c.Context(), httpserver.UserID(c), c.Params("id")); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) models(c fiber.Ctx) error {
	options, err := h.service.Models(c.Context(), httpserver.UserID(c))
	if err != nil {
		return fail(c, err)
	}

	type option struct {
		ProviderID   string `json:"providerId"`
		ProviderName string `json:"providerName"`
		Model        string `json:"model"`
	}

	payload := make([]option, 0, len(options))
	for _, o := range options {
		payload = append(payload, option{ProviderID: o.ProviderID, ProviderName: o.ProviderName, Model: o.Model})
	}

	return c.JSON(payload)
}

var badInput = []struct {
	err  error
	code string
}{
	{domain.ErrInvalidKind, "invalid_kind"},
	{domain.ErrInvalidName, "invalid_name"},
	{domain.ErrInvalidBaseURL, "invalid_base_url"},
	{domain.ErrInvalidKey, "invalid_key"},
	{domain.ErrTooMany, "too_many"},
}

func fail(c fiber.Ctx, err error) error {
	for _, known := range badInput {
		if errors.Is(err, known.err) {
			return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure(known.code))
		}
	}

	switch {
	case errors.Is(err, domain.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(httpserver.Failure("not_found"))
	case errors.Is(err, domain.ErrVerifyFailed):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(httpserver.Failure("verify_failed"))
	}

	return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
}
