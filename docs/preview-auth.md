# Authenticated site previews (opt-in)

The standard `/run/<site>` preview remains opaque-origin, cookie-free and sandboxed.
It is intentionally **not** suitable for testing a web app's login or file uploads.

For full-stack projects, an operator can configure `SITE_PREVIEW_DOMAIN` to a
**dedicated wildcard DNS/TLS zone**, for example `preview.example.net`. Route
`*.preview.example.net` to the Workspace reverse proxy and terminate HTTPS there.
The domain must not be the Workspace main origin, and must not share the Workspace
session cookie domain. Every site then has its **own hostname**:
`https://<site>.preview.example.net/run/<site>/`.

Opening a site from the normal Workspace `/run/<site>/` address redirects its
signed-in owner to that hostname. For private sites an expiring, browser-bound
ticket is carried on the initial URL and exchanged for an HttpOnly, Secure,
host-only preview-access cookie. The Workspace `session` cookie and the preview
ticket are never forwarded to project code. Only application cookies (e.g.
`pukat_session`) are proxied, with `Domain=` cookies dropped.

This is **not** a change to default sandbox behavior. Only configure this mode
on a separately controlled wildcard zone with HTTPS, proxy hostname validation,
and trusted TLS routing. Never set this domain to the Workspace main hostname.
State-changing requests from foreign origins are refused, and preview responses
cannot opt into credentialed wildcard CORS. App cookies work because the
application and its API now share the same browser origin.

Operator checklist:

- Provision wildcard DNS and certificate for the dedicated zone.
- Keep Workspace's main login cookie host-only (do not set `Domain=`).
- Set `SITE_PREVIEW_DOMAIN` to the wildcard zone suffix, without `*.`.
- Verify register, login, refresh, project isolation, and logout in a browser.
- Keep file-size limits, upload validation, storage quotas and CSRF controls in the app.
- Confirm an attacker-controlled preview cannot reach Workspace administration.

This mode requires operator DNS/TLS changes; merging code alone does **not**
activate authenticated previews.
