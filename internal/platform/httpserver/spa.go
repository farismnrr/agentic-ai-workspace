package httpserver

import (
	"io/fs"
	"strings"

	"github.com/gofiber/fiber/v3"
	staticmiddleware "github.com/gofiber/fiber/v3/middleware/static"
)

const (
	assetsPrefix        = "/assets/"
	immutableCacheValue = "public, max-age=31536000, immutable"
)

func registerSPA(app *fiber.App, frontend fs.FS) {
	app.Use("/", staticmiddleware.New("", staticmiddleware.Config{
		FS:              frontend,
		ModifyResponse:  cacheHashedAssets,
		NotFoundHandler: missingAssetNotFound,
	}))
	app.Get("/*", spaFallback(frontend))
}

// cacheHashedAssets marks Vite's content-hashed files as safe to cache forever
// and makes every other static file revalidate so deploys show up immediately.
func cacheHashedAssets(c fiber.Ctx) error {
	if strings.HasPrefix(c.Path(), assetsPrefix) {
		c.Set(fiber.HeaderCacheControl, immutableCacheValue)
		return nil
	}

	c.Set(fiber.HeaderCacheControl, "no-cache")
	if strings.HasSuffix(c.Path(), ".webmanifest") {
		c.Set(fiber.HeaderContentType, "application/manifest+json")
	}

	return nil
}

// missingAssetNotFound keeps a missing hashed asset a real 404 instead of the HTML shell.
func missingAssetNotFound(c fiber.Ctx) error {
	if strings.HasPrefix(c.Path(), assetsPrefix) {
		return fiber.ErrNotFound
	}

	return c.Next()
}

func spaFallback(frontend fs.FS) fiber.Handler {
	return func(c fiber.Ctx) error {
		index, err := fs.ReadFile(frontend, "index.html")
		if err != nil {
			return fiber.ErrInternalServerError
		}

		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Set(fiber.HeaderCacheControl, "no-cache")
		return c.Send(index)
	}
}
