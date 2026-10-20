package http

import (
	"errors"
	"net/url"

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
		app.Put("/api/providers/:id/context", auth, sameOrigin, h.setContext)
		app.Delete("/api/providers/:id", auth, sameOrigin, h.remove)
		app.Get("/api/models", auth, h.models)
	}
}

// providerID is the provider the path names. A plan's id has a colon, which a browser sends as %3A.
func providerID(c fiber.Ctx) string {
	id := c.Params("id")
	if decoded, err := url.PathUnescape(id); err == nil {
		return decoded
	}

	return id
}

type providerPayload struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	KeyHint string `json:"keyHint"`
	// ContextWindows maps every model to its context window in tokens (chosen or default).
	ContextWindows map[string]int64 `json:"contextWindows"`
	// Efforts maps every model to the effort levels worth offering for it; empty when it does not reason.
	Efforts  map[string][]string `json:"efforts"`
	Models   []string            `json:"models"`
	Verified bool                `json:"verified"`
	Enabled  bool                `json:"enabled"`
}

func toPayload(p domain.Provider) providerPayload {
	return providerPayload{
		ID: p.ID, Kind: string(p.Kind), Name: p.Name, BaseURL: p.BaseURL, KeyHint: p.KeyHint,
		Models: p.Models, Verified: p.Verified, Enabled: p.Enabled, ContextWindows: windows(p),
		Efforts: efforts(p),
	}
}

func efforts(p domain.Provider) map[string][]string {
	all := make(map[string][]string, len(p.Models))
	for _, model := range p.Models {
		info, ok := p.Catalog[model]
		if !ok {
			info = domain.ModelInfo{Name: model}
		}

		all[model] = domain.EffortLevels(p.Kind, info)
	}

	return all
}

func windows(p domain.Provider) map[string]int64 {
	all := make(map[string]int64, len(p.Models))
	for _, model := range p.Models {
		all[model] = domain.ContextWindow(model, p.ContextWindows, p.Catalog)
	}

	return all
}

func (h *Handler) list(c fiber.Ctx) error {
	h.service.SyncStale(c.Context(), httpserver.UserID(c))

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
	provider, err := h.service.Verify(c.Context(), httpserver.UserID(c), providerID(c))
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

	if err := h.service.SetEnabled(c.Context(), httpserver.UserID(c), providerID(c), body.Enabled); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) remove(c fiber.Ctx) error {
	if err := h.service.Remove(c.Context(), httpserver.UserID(c), providerID(c)); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

var badInput = []struct {
	err  error
	code string
}{
	{domain.ErrInvalidKind, "invalid_kind"},
	{domain.ErrInvalidName, "invalid_name"},
	{domain.ErrInvalidBaseURL, "invalid_base_url"},
	{domain.ErrAddressFixed, "address_fixed"},
	{domain.ErrInvalidKey, "invalid_key"},
	{domain.ErrTooMany, "too_many"},
	{domain.ErrInvalidContext, "invalid_context"},
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
