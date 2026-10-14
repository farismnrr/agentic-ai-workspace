package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

type tokenBody struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
}

// token implements the token endpoint for the code and refresh grants.
func (h *Handler) token(c fiber.Ctx) error {
	noStore(c)

	var (
		response application.TokenResponse
		err      error
	)

	switch c.FormValue("grant_type") {
	case "authorization_code":
		response, err = h.service.ExchangeCode(c.Context(), application.ExchangeParams{
			Code:         c.FormValue("code"),
			RedirectURI:  c.FormValue("redirect_uri"),
			ClientID:     c.FormValue("client_id"),
			CodeVerifier: c.FormValue("code_verifier"),
			Resource:     c.FormValue("resource"),
		})
	case "refresh_token":
		response, err = h.service.Refresh(c.Context(), application.RefreshParams{
			RefreshToken: c.FormValue("refresh_token"),
			ClientID:     c.FormValue("client_id"),
			Resource:     c.FormValue("resource"),
		})
	default:
		err = domain.NewError(domain.CodeUnsupportedGrantType, "use authorization_code or refresh_token")
	}

	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(tokenBody{
		AccessToken: response.AccessToken, TokenType: "Bearer", ExpiresIn: response.ExpiresIn,
		RefreshToken: response.RefreshToken, Scope: response.Scope, IDToken: response.IDToken,
	})
}

// revoke implements RFC 7009; it answers 200 whether or not the token was known.
func (h *Handler) revoke(c fiber.Ctx) error {
	noStore(c)

	if token := c.FormValue("token"); token != "" {
		if err := h.service.RevokeToken(c.Context(), token); err != nil {
			return writeError(c, err)
		}
	}

	return c.SendStatus(fiber.StatusOK)
}
