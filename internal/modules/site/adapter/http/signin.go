package http

import (
	"fmt"
	"html"
	"net/url"

	"github.com/gofiber/fiber/v3"
)

const signInPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>This app is private</title>
<style>
body{margin:0;min-height:100vh;display:grid;place-items:center;font:16px/1.5 system-ui,sans-serif;
background:#f6f6f4;color:#1f2328}
main{max-width:26rem;padding:2rem;text-align:center}
h1{font-size:1.25rem;margin:0 0 .5rem}p{margin:0;color:#57606a}
a{display:inline-block;margin-top:1.25rem;padding:.6rem 1.2rem;border-radius:.5rem;background:#1f2328;color:#fff;
text-decoration:none}
a:focus-visible{outline:3px solid #0969da;outline-offset:2px}
</style></head><body><main>
<h1>This app is private</h1>
<p>Only the person who made it can open it. Sign in with that account to continue.</p>
<a href="/login?next=%s">Sign in</a>
</main></body></html>`

// signIn answers a page request that the person may not open with a page that explains why and
// leads to sign-in, which then brings them back. It is plain HTML with no scripts.
func signIn(c fiber.Ctx) error {
	setSecure(c)
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")

	next := url.QueryEscape(string(c.Request().URI().RequestURI()))

	return c.Status(fiber.StatusForbidden).SendString(fmt.Sprintf(signInPage, html.EscapeString(next)))
}
