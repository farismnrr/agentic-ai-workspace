package http

import (
	"fmt"
	"html"
	"net/url"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

const errorPage = `<!doctype html><meta charset="utf-8"><meta name="viewport" content="width=device-width">` +
	`<title>Cannot connect</title><body style="font:16px system-ui;max-width:32rem;margin:4rem auto;padding:0 1rem">` +
	`<h1>We could not start this connection</h1><p>%s</p><p>Go back to the app and try again.</p>`

// authorize validates the request, then hands the browser to the SPA consent page.
func (h *Handler) authorize(c fiber.Ctx) error {
	requestID, err := h.service.Authorize(c.Context(), application.AuthorizeParams{
		ResponseType:        c.Query("response_type"),
		ClientID:            c.Query("client_id"),
		RedirectURI:         c.Query("redirect_uri"),
		Scope:               c.Query("scope"),
		State:               c.Query("state"),
		CodeChallenge:       c.Query("code_challenge"),
		CodeChallengeMethod: c.Query("code_challenge_method"),
		Resource:            c.Query("resource"),
		Nonce:               c.Query("nonce"),
	})
	if err != nil {
		return h.authorizeFailure(c, err)
	}

	return c.Redirect().Status(fiber.StatusFound).To("/oauth/consent?request=" + url.QueryEscape(requestID))
}

func (h *Handler) authorizeFailure(c fiber.Ctx, err error) error {
	oauthErr, ok := domain.AsError(err)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).SendString("server error")
	}

	if oauthErr.RedirectURI != "" {
		location, redirectErr := h.service.RedirectError(oauthErr)
		if redirectErr != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("server error")
		}

		return c.Redirect().Status(fiber.StatusFound).To(location)
	}

	// The redirect URI is untrusted, so the error is shown here, never sent there.
	c.Set(fiber.HeaderContentSecurityPolicy, "default-src 'none'; style-src 'unsafe-inline'")
	c.Type("html")

	return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf(errorPage, html.EscapeString(oauthErr.Description)))
}
