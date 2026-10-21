package http

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

const (
	visitorHeader  = "X-Site-Visitor"
	visitorCookie  = "site_visitor"
	visitorIDBytes = 16
	visitorTTL     = 4 * time.Hour
)

// browser is the person behind a request as far as the proxy can tell: the jar that keeps their app login, and,
// when a page is being opened on a public site, the pass the page will carry.
type browser struct {
	jar  *jarSession
	pass string
}

// browserOf finds whose app login a request uses. A private site knows its visitor by the ticket they hold. A
// site on its own host keeps cookies in the browser. A public site has neither, so each browser gets a pass.
func (h *Handler) browserOf(c fiber.Ctx, site domain.Site, valid, preview bool) browser {
	switch {
	case preview:
		return browser{}
	case site.Visibility == domain.Private:
		if !valid {
			return browser{}
		}

		return browser{jar: h.jar.session(site.Token, h.holder(c))}
	default:
		return h.publicVisitor(c, site)
	}
}

// publicVisitor knows a visitor to a public site by a pass: signed, tied to the site, and sent two ways. A page the
// person opens carries it in a cookie of this server, which browsers send on navigation, and the page's own
// requests carry it in the X-Site-Visitor header, because a sandboxed page has no cookies to send. Opening a page
// without a valid pass gives a new one; anything else without one keeps no login.
func (h *Handler) publicVisitor(c fiber.Ctx, site domain.Site) browser {
	page := isPage(c)

	raw := c.Get(visitorHeader)
	if raw == "" && page {
		raw = c.Cookies(visitorCookie)
	}

	id, known := h.visitorID(site, raw)

	if !known {
		if !page {
			return browser{}
		}

		id = newVisitorID()
	}

	who := browser{jar: h.jar.session(site.Token, "v|"+id)}
	if page {
		who.pass = id + "." + h.service.Ticket(site.Token, "v|"+id)
	}

	return who
}

func (h *Handler) visitorID(site domain.Site, pass string) (string, bool) {
	id, ticket, found := strings.Cut(pass, ".")
	if !found || len(id) != 2*visitorIDBytes {
		return "", false
	}

	if _, err := hex.DecodeString(id); err != nil {
		return "", false
	}

	return id, h.service.ValidTicket(site.Token, "v|"+id, ticket)
}

func newVisitorID() string {
	b := make([]byte, visitorIDBytes)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}

// setVisitorCookie hands the pass to the browser, for the next page it opens. The cookie belongs to the site's own
// address, so no other site's page is given it.
func (h *Handler) setVisitorCookie(c fiber.Ctx, site domain.Site, who browser, err error) {
	if err != nil || who.pass == "" {
		return
	}

	c.Cookie(&fiber.Cookie{
		Name: visitorCookie, Value: who.pass, Path: site.BasePath() + "/", Secure: true, HTTPOnly: true,
		SameSite: "Lax", MaxAge: int(visitorTTL / time.Second),
	})
}
