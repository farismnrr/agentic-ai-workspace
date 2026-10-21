package http

import (
	"net/url"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

func (h *Handler) redirectAllowedRequest(
	c fiber.Ctx, site domain.Site, segment, ticket string, valid, owner, preview bool, rest string,
) (bool, error) {
	if !preview && owner && h.previewDomain != "" && isPage(c) {
		return true, h.redirectPreview(c, site, segment, rest)
	}
	if !h.needsRedirect(c, site, segment, ticket, valid) {
		return false, nil
	}
	target := url.URL{
		Path:     site.BasePath() + segmentPath(segment) + "/" + rest,
		RawQuery: string(c.Request().URI().QueryString()),
	}
	return true, c.Redirect().Status(fiber.StatusFound).To(target.String())
}

func (h *Handler) redirectPreview(c fiber.Ctx, site domain.Site, segment, rest string) error {
	dest := url.URL{
		Scheme: "https", Host: site.Slug + "." + h.previewDomain,
		Path:     site.BasePath() + segmentPath(segment) + "/" + rest,
		RawQuery: string(c.Request().URI().QueryString()),
	}
	return c.Redirect().Status(fiber.StatusFound).To(dest.String())
}
