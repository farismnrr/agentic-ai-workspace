package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/space/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

type scopePayload struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type toolPayload struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Scope       string `json:"scope"`
	Group       string `json:"group"`
	ReadOnly    bool   `json:"readOnly"`
	Enabled     bool   `json:"enabled"`
}

type spacePayload struct {
	Slug   string         `json:"slug"`
	Name   string         `json:"name"`
	Path   string         `json:"path"`
	URL    string         `json:"url"`
	Scopes []scopePayload `json:"scopes"`
	Tools  []toolPayload  `json:"tools"`
}

// list returns every hosted space with the person's tool switches.
func (a *API) list(c fiber.Ctx) error {
	userID, ok := a.signedIn(c)
	if !ok {
		return nil
	}

	payload := []spacePayload{}

	for _, space := range a.registry.Spaces() {
		states, err := a.permissions.For(c.Context(), userID, space)
		if err != nil {
			return fail(c, err)
		}

		payload = append(payload, a.describe(space, states))
	}

	return c.JSON(payload)
}

func (a *API) describe(space domain.Space, states map[string]bool) spacePayload {
	payload := spacePayload{
		Slug: space.Slug, Name: space.Name, Path: space.Path(), URL: a.registry.ResourceURI(space),
		Scopes: []scopePayload{}, Tools: []toolPayload{},
	}

	for _, scope := range space.Scopes {
		payload.Scopes = append(payload.Scopes, scopePayload{ID: scope.ID, Description: scope.Description})
	}

	for _, tool := range space.Tools {
		payload.Tools = append(payload.Tools, toolPayload{
			Name: tool.Name, Title: tool.Title, Description: tool.Description, Scope: tool.Scope,
			Group: tool.Group, ReadOnly: tool.ReadOnly, Enabled: states[tool.Name],
		})
	}

	return payload
}

// setTool switches one tool on or off for the signed-in person.
func (a *API) setTool(c fiber.Ctx) error {
	userID, ok := a.signedIn(c)
	if !ok {
		return nil
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(httpserver.Failure("invalid_body"))
	}

	if err := a.permissions.Set(c.Context(), userID, c.Params("slug"), c.Params("tool"), body.Enabled); err != nil {
		return fail(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
