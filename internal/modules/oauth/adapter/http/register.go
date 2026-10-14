package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

type registrationRequest struct {
	ClientName     string   `json:"client_name"`
	AuthMethod     string   `json:"token_endpoint_auth_method"`
	ApplicationURI string   `json:"client_uri"`
	RedirectURIs   []string `json:"redirect_uris"`
}

type registrationResponse struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	AuthMethod   string   `json:"token_endpoint_auth_method"`
	RedirectURIs []string `json:"redirect_uris"`
	GrantTypes   []string `json:"grant_types"`
	ResponseType []string `json:"response_types"`
	IssuedAt     int64    `json:"client_id_issued_at"`
}

// register implements Dynamic Client Registration (RFC 7591) for public clients.
func (h *Handler) register(c fiber.Ctx) error {
	noStore(c)

	var body registrationRequest
	if err := c.Bind().JSON(&body); err != nil {
		return writeError(c, domain.NewError(domain.CodeInvalidClientMetadata, "body must be a JSON object"))
	}

	client, err := h.service.RegisterClient(c.Context(), application.Registration{
		Name: body.ClientName, RedirectURIs: body.RedirectURIs, AuthMethod: body.AuthMethod,
	})
	if err != nil {
		return writeError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(registrationResponse{
		ClientID: client.ID, ClientName: client.Name, RedirectURIs: client.RedirectURIs,
		GrantTypes: []string{"authorization_code", "refresh_token"}, ResponseType: []string{"code"},
		AuthMethod: "none", IssuedAt: time.Now().Unix(),
	})
}
