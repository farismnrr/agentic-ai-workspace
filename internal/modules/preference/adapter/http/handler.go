package http

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves a person's preferences.
type Handler struct {
	service *application.Service
}

// NewHandler builds the preference HTTP adapter.
func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

// Mount registers GET and PUT /api/preferences.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/preferences", auth, h.get)
		app.Put("/api/preferences", auth, sameOrigin, h.put)
	}
}

type payload struct {
	DefaultProviderID string        `json:"defaultProviderId"`
	DefaultModel      string        `json:"defaultModel"`
	PersonaName       string        `json:"personaName"`
	TalkStyle         string        `json:"talkStyle"`
	Language          string        `json:"language"`
	About             string        `json:"about"`
	Rules             []domain.Rule `json:"rules"`
}

func toPayload(p domain.Preferences) payload {
	return payload{
		DefaultProviderID: p.DefaultProviderID, DefaultModel: p.DefaultModel, PersonaName: p.PersonaName,
		TalkStyle: p.TalkStyle, Language: p.Language, About: p.About, Rules: append([]domain.Rule{}, p.Rules...),
	}
}

func (h *Handler) get(c fiber.Ctx) error {
	prefs, err := h.service.Get(c.Context(), httpserver.UserID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.JSON(toPayload(prefs))
}

func (h *Handler) put(c fiber.Ctx) error {
	var body payload
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	saved, err := h.service.Update(c.Context(), httpserver.UserID(c), domain.Preferences{
		DefaultProviderID: body.DefaultProviderID, DefaultModel: body.DefaultModel, PersonaName: body.PersonaName,
		TalkStyle: body.TalkStyle, Language: body.Language, About: body.About, Rules: body.Rules,
	})
	if errors.Is(err, domain.ErrInvalid) {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_preferences"))
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.JSON(toPayload(saved))
}
