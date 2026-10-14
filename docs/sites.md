# Sites

A site is a web app the AI starts in the sandbox so you can open it in your browser. You find your sites under **Sites** in the sidebar, and a running one also gets a link in the task thread.

## Using sites

- Ask the AI to build a web page or app and run it. It asks before starting the app, then shows an **Open** link.
- Each app has an address like `/run/my-shop/<random-uuid>/` on the same domain and port as the app. No DNS setup is needed.
- A project has one site. Starting it again replaces the running app and gives it a **new address**; old links stop working.
- **Private** (the default): only you can open it, while you are signed in. Anyone else, or a browser that is not signed in, gets a 403 page with a **Sign in** button that brings you back. **Public**: anyone with the link can open it without signing in. Only you can change this, from the Sites page, and going public asks for confirmation. The AI cannot do it.
- Apps run until you press **Stop** (Sites page or the sidebar) or the server restarts. There is no idle timeout, so stop apps you no longer need. Each person can run `SITE_MAX_PER_USER` apps and the server `SITE_MAX_TOTAL` (defaults 2 and 6).

## What an app can and cannot do

- It runs in the same sandbox as checks: no network except package registries, no access to your computer or other projects.
- It must listen on `127.0.0.1` at the port the AI chose, and stay running. Use a normal server, not a dev server with hot reload (WebSockets are not supported yet).
- The page is shown under a path, which is removed before the request reaches the app. Links inside the app should be relative (`./style.css`); `BASE_PATH` is available as an environment variable.
- The page runs in a browser sandbox: it has no cookies, `localStorage` or service workers, and it cannot touch your account. Requests larger than 1 MiB are refused.

## How the app is kept apart from the main app

The app shares the domain with the main app, so the browser itself must keep them apart. Every layer below is always on:

1. Every response under `/run` carries `Content-Security-Policy: sandbox …` without `allow-same-origin`. The browser gives the page an opaque origin: it cannot read your cookies or storage, cannot register a service worker, and its requests carry no login. The app's own security headers are replaced.
2. Main-app endpoints that need your session refuse requests the browser marks `Sec-Fetch-Site: cross-site` (except plain navigations). Requests from a sandboxed page are cross-site.
3. The proxy never forwards your session cookie or `Authorization` header to the app, and drops `Set-Cookie`, `Clear-Site-Data`, `Service-Worker-Allowed` and similar headers from its answers. Hop-by-hop headers are removed first, so a client cannot make the proxy drop its own headers.
4. The proxy only talks to the app's Unix socket. The container has no network, so nothing else can be reached, and the address never chooses the destination.
5. Unknown, stopped and replaced addresses answer 404. A private site answers 403 to anyone without your login or a valid ticket, whatever headers the request claims to send.

**How private works.** The browser runs a site's page in a sandbox, so what the page loads itself (scripts, styles, `fetch`) carries no login. When you open a private site, the server checks your login and sends you to an address with a short-lived, signed **ticket** (`/run/<name>/<uuid>/~<ticket>/`). Relative links then keep the ticket, so the page works. The uuid alone opens nothing, and pages themselves always need your login, so a copied address is useless to someone who is not signed in as you. Tickets last 12 hours, belong to one site address, and stop working when the site is started again or the server restarts. A form post from the page uses the ticket too, because browsers do not send your login with it.

## Technical notes

The container publishes the app's port as a Unix socket (`socat`, started by `sandbox-entry` when `SITE_PORT` is set) in a folder under `DATA_DIR/sites` that is mounted at `/run/site`. The server proxies `/run/<name>/<uuid>/*` to that socket from the same process and port. Sites are tracked in the `sites` table and are reset at startup because containers do not survive a restart.
