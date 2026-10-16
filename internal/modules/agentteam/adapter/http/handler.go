package http

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/agentteam/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// Handler serves a person's agent team.
type Handler struct {
	service *application.Service
}

// NewHandler builds the agent team HTTP adapter.
func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

// Mount registers GET and PUT /api/agent-team and POST /api/agent-team/reset.
func (h *Handler) Mount(auth, sameOrigin fiber.Handler) httpserver.Mount {
	return func(app *fiber.App) {
		app.Get("/api/agent-team", auth, h.get)
		app.Put("/api/agent-team", auth, sameOrigin, h.put)
		app.Post("/api/agent-team/reset", auth, sameOrigin, h.reset)
	}
}

type rolePayload struct {
	Key        string   `json:"key"`
	Name       string   `json:"name"`
	Purpose    string   `json:"purpose"`
	Prompt     string   `json:"prompt"`
	ProviderID string   `json:"providerId"`
	Model      string   `json:"model"`
	Effort     string   `json:"effort"`
	Access     []string `json:"access"`
	Lead       bool     `json:"lead"`
	Enabled    bool     `json:"enabled"`
	Preset     bool     `json:"preset"`
}

type teamPayload struct {
	Roles        []rolePayload `json:"roles"`
	UseByDefault bool          `json:"useByDefault"`
}

func toPayload(team domain.Team) teamPayload {
	roles := make([]rolePayload, 0, len(team.Roles))

	for _, r := range team.Roles {
		roles = append(roles, rolePayload{
			Key: r.Key, Name: r.Name, Purpose: r.Purpose, Prompt: r.Prompt, ProviderID: r.ProviderID,
			Model: r.Model, Effort: r.Effort, Access: append([]string{}, r.Access...), Lead: r.Lead,
			Enabled: r.Enabled, Preset: r.Preset(),
		})
	}

	return teamPayload{Roles: roles, UseByDefault: team.UseByDefault}
}

func (p teamPayload) team() domain.Team {
	roles := make([]domain.Role, 0, len(p.Roles))

	for _, r := range p.Roles {
		roles = append(roles, domain.Role{
			Key: r.Key, Name: r.Name, Purpose: r.Purpose, Prompt: r.Prompt, ProviderID: r.ProviderID,
			Model: r.Model, Effort: r.Effort, Access: r.Access, Lead: r.Lead, Enabled: r.Enabled,
		})
	}

	return domain.Team{Roles: roles, UseByDefault: p.UseByDefault}
}

func (h *Handler) get(c fiber.Ctx) error {
	team, err := h.service.Get(c.Context(), httpserver.UserID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.JSON(toPayload(team))
}

func (h *Handler) put(c fiber.Ctx) error {
	var body teamPayload
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	saved, err := h.service.Update(c.Context(), httpserver.UserID(c), body.team())
	if errors.Is(err, domain.ErrInvalid) {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_agent_team"))
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.JSON(toPayload(saved))
}

func (h *Handler) reset(c fiber.Ctx) error {
	team, err := h.service.Reset(c.Context(), httpserver.UserID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(httpserver.Failure("internal"))
	}

	return c.JSON(toPayload(team))
}
