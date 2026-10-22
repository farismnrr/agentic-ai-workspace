# Authenticated site previews (opt-in)

The standard `/run/<site>` preview remains opaque-origin, cookie-free and sandboxed.
An app can still keep a login there, private or public: the server holds the app's cookies (see `sites.md`).
It is still **not** suitable for file uploads, `localStorage`, or anything that needs a real browser origin.

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

## Agent screenshot sessions

The separate `screenshot_session` tool uses an in-memory cookie jar owned by one
user, project, task and running app container. It does not use a person's browser
cookies, preview ticket or the visitor cookie jar.

The tool can fill email and password inputs, click one selected submit control,
and check one caller-selected readiness indicator. Native browser input allows
regular form submission and JavaScript submit handlers that send JSON. Hidden
CSRF form values are submitted by the page itself.

The tool receives names of secret Project Env values. The server checks project
ownership and sends their values to the short-lived browser container on stdin.
It never returns the values. Login success means only that the requested
readiness selector appeared; screenshots are not proof of account permissions.

The local jar expires after 30 minutes idle or four hours total. Reset clears it.
Logout submits the app's own form and clears the local jar even if submission
fails, but cannot confirm server-side revocation. Browser storage, MFA, CAPTCHA,
SSO and external redirects are unsupported. Pukat live behavior remains unverified.

Operator checklist:

- Provision wildcard DNS and certificate for the dedicated zone.
- Keep Workspace's main login cookie host-only (do not set `Domain=`).
- Set `SITE_PREVIEW_DOMAIN` to the wildcard zone suffix, without `*.`.
- Verify register, login, refresh, project isolation, and logout in a browser.
- Keep file-size limits, upload validation, storage quotas and CSRF controls in the app.
- Confirm an attacker-controlled preview cannot reach Workspace administration.

This mode requires operator DNS/TLS changes; merging code alone does **not**
activate authenticated previews.
