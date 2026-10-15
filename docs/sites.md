# Sites

A site is a web app the AI runs in the sandbox so you can open it in your browser. You'll find it under Sites in the sidebar and as a link in the task.

## Using sites

- Ask the AI to build and run a web app. It asks first, then shows an Open link.
- The address looks like `/run/my-shop/<uuid>/`, on the same domain and port as the app. You don't need any DNS setup.
- Each project has one site. Starting it again gives it a new address, and old links stop working.
- A site is private by default: only you can open it, while signed in. Anyone else gets "This page is private" (403).
- You can make a site public so anyone with the link can open it. Only you can do this, from the Sites page. The AI can't.
- Sites run until you press Stop or the server restarts. There is no idle timeout.
- Limits: `SITE_MAX_PER_USER` sites per person and `SITE_MAX_TOTAL` per server (defaults 2 and 6).

## What a site can do

- It runs in the same sandbox as checks. It can reach package registries, but not your computer or other projects.
- It must listen on `127.0.0.1` at the port the AI chose. Use a normal server, not a hot-reload dev server. WebSockets aren't supported yet.
- The path prefix is removed before requests reach the app. Use relative links (`./style.css`); `BASE_PATH` holds the prefix.
- The page has no cookies, `localStorage` or service workers, and it can't touch your account. Requests over 1 MiB are refused.

## How sites are kept apart from your account

A site shares the domain with the main app. These protections are always on:

1. Every `/run` response sends `Content-Security-Policy: sandbox` without `allow-same-origin`. The page gets an opaque origin, so it can't read your cookies or storage, and its requests carry no login.
2. Endpoints that need your session refuse cross-site requests (`Sec-Fetch-Site: cross-site`), except plain navigations.
3. The proxy never forwards your session cookie or `Authorization` header. It drops `Set-Cookie`, `Clear-Site-Data`, `Service-Worker-Allowed` and similar headers from the app's responses.
4. The proxy talks only to the app's Unix socket. The container has no network, so the address can't point anywhere else.
5. Unknown, stopped or replaced addresses return 404. A private site returns 403 to anyone without your login or a valid ticket.

### How private sites work

A sandboxed page can't send your login, so the server uses a ticket instead:

1. You open a private site. The server checks your login.
2. It redirects you to `/run/<name>/<uuid>/~<ticket>/`. Relative links keep the ticket.
3. The ticket lasts 4 hours. It works only for that site address and that browser (and that IP when `CLIENT_IP_HEADER` is set).
4. It stops working when the site restarts or the server restarts.

An address copied to another machine won't work. Still, treat an address with a ticket like a password while it's valid.

## Technical notes

The sandbox exposes the app's port as a Unix socket (`socat`, started by `sandbox-entry` when `SITE_PORT` is set) in `DATA_DIR/sites`, mounted at `/run/site`. The server proxies `/run/<name>/<uuid>/*` to it from the same process and port. Sites are stored in the `sites` table and reset at startup, because containers don't survive a restart.
