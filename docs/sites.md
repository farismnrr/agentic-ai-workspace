# Sites

A site is a web app the AI starts in the sandbox so you can open it in your browser. You find your sites under **Sites** in the sidebar, and a running one also gets a link in the task thread.

## Using sites

- Ask the AI to build a web page or app and run it. It asks before starting the app, then shows an **Open** link.
- Each app has an address like `/run/my-shop/<random-uuid>/` on the same domain and port as the app. No DNS setup is needed.
- A project has one site. Starting it again replaces the running app and gives it a **new address**; old links stop working.
- **Private** (the default): the page opens only for you, while you are signed in. **Public**: anyone with the link can open it without signing in. Only you can change this, from the Sites page, and going public asks for confirmation. The AI cannot do it.
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
5. Unknown, stopped and replaced addresses all answer 404, so nothing reveals which sites exist.

A private site's page needs your login, but a sandboxed page's own requests (scripts, `fetch`) cannot carry it. Those are protected by the unguessable uuid in the address, which is replaced on every start. Treat a private link as a secret.

## Technical notes

The container publishes the app's port as a Unix socket (`socat`, started by `sandbox-entry` when `SITE_PORT` is set) in a folder under `DATA_DIR/sites` that is mounted at `/run/site`. The server proxies `/run/<name>/<uuid>/*` to that socket from the same process and port. Sites are tracked in the `sites` table and are reset at startup because containers do not survive a restart.
