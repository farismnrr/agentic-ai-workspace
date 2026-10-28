# Sites

A site is a web app the AI runs in the sandbox so you can open it in your browser. You'll find it under Sites in the sidebar and as a link in the task.

## Using sites

- Ask the AI to build and run a web app. It asks first, then shows an Open link.
- The address looks like `/run/my-shop/`, on the same domain and port as the app. You don't need any DNS setup.
- `my-shop` is the project's ID, chosen when you create the project. No two projects share an ID.
- Each project has one site. Its address never changes: the ID is permanent, so stopping, starting or renaming the project does not move it.
- Old addresses with a UUID (`/run/my-shop/<uuid>/`) still lead to the same app.
- Press Start on the Sites page to run a stopped app again. You do not need the AI.
- An app started before this button existed needs the AI to start it once.
- A site is private by default: only you can open it, while signed in. Anyone else gets "This page is private" (403).
- You can make a site public so anyone with the link can open it. Only you can do this, from the Sites page. The AI can't.
- The address is the project's ID, so anyone can guess the address of a public site. Make a site public only if anyone may open it.
- Sites run until you press Stop or the server restarts. There is no idle timeout.
- A project runs one app. Ten projects can run ten apps, so there is no limit per person by default.
- `SITE_MAX_TOTAL` caps the apps one server runs at once (default 20); `SITE_MAX_PER_USER` can cap a person (default none).

## What a site can do

- It runs in the same sandbox as checks. It can reach package registries, but not your computer or other projects.
- Fonts are installed from npm (Fontsource), not loaded from Google Fonts, so they work in every stack and need no extra network access.
- Any language works: Rust, Go, Python, Node, PHP, Next.js, SvelteKit and more. Nuxt is only the default for new apps.
- It must listen on `127.0.0.1` at the port the AI chose. Use a normal server, not a hot-reload dev server. WebSockets aren't supported yet.
- The path prefix is removed before requests reach the app. Relative links (`./style.css`) are the right way.
- A link like `/style.css` in a page or stylesheet is moved under the address for you, so the page keeps its styling.
- The same goes for images, `fetch` and `history.pushState` that a page sets while it runs.
- `fetch` with `credentials: 'include'` and XHR with `withCredentials` are sent without credentials. The page has no cookies, and the browser would refuse them.
- Links built in pieces or hidden in `.js` files are not moved. Use relative paths there.
- When the AI starts an app, the server loads its pages and tells the AI about links that would not work.
- Starting is allowed during development. Completion checks follow task mode and impact; sensitive work keeps its required security checks.
- It also flags a stylesheet that is empty, never built (raw Tailwind) or answered with a web page.
- `screenshot` uses the project's stable address. It does not borrow your browser's login or private-site ticket.
- The page has no cookies, `localStorage` or service workers, and it can't touch your account. Requests over 1 MiB are refused.
- A login in your app still works: the server keeps the app's cookies for you and sends them back on the next request.
- Those cookies live in server memory only. They are per app and per browser, and the browser never receives them.
- A private app knows your browser by its ticket. A public app gives each browser a pass: a cookie for that app's address, and a header the page adds to its own requests.
- A public app's pass is never sent to the app, and a request without a valid pass keeps no login.
- A pass works for 4 hours. Whoever holds it, in the cookie or the page, uses that browser's login. Don't share a page's source or your cookies.
- They are lost when Workspace restarts or the app is started again, so you sign in again.

## Screenshots

- The AI can take a screenshot of your running app, on a desktop or a phone screen, to check how it looks. It shows in the thread.
- It uses a headless browser in the sandbox. That browser has no internet, so fonts and images from other websites don't load in the screenshot, but they do for you.
- It reaches only your project's own app.
- `screenshot_session` can log in or out with a test account stored as Project Env secrets, or clear the agent's local session.
- Enable **Manage screenshot login session** separately in Settings. Screenshot access alone does not allow login actions.
- Login needs email, password, submit and readiness CSS selectors. Hidden CSRF fields work when the app submits them with its form or JavaScript handler.
- The tool reports whether the requested readiness selector appeared. A screenshot or form submission alone does not prove login succeeded.
- The agent cookie jar is separate from your browser. It stays in server memory for 30 minutes idle or four hours total, and is removed on reset, task cleanup or restart.
- Agent screenshot sessions are limited to 16 per person and 256 across the server; each jar is capped at 128 cookies and 64 KiB.
- Logout submits the app's logout flow, then clears the local jar even if the request fails. The tool cannot confirm that the app revoked its server-side session.
- Apps that require browser storage, a real browser origin, MFA, CAPTCHA or SSO are unsupported. Pukat live compatibility has not been verified.
- Screenshots are kept with the task and deleted with it.

## How sites are kept apart from your account

A site shares the domain with the main app. These protections are always on:

1. Every `/run` response sends `Content-Security-Policy: sandbox`, without `allow-same-origin`.
   - The page gets an opaque origin.
   - It can't read your cookies or storage, and its requests carry no login.
2. Endpoints that need your session refuse cross-site requests (`Sec-Fetch-Site: cross-site`), except plain navigations.
3. The proxy never forwards your session cookie or `Authorization` header.
   - It also drops `Set-Cookie`, `Clear-Site-Data`, `Service-Worker-Allowed` and similar headers from the app's responses.
   - The app's own cookies are kept on the server, per app and visitor, and never sent to the browser.
4. The proxy talks only to the app's Unix socket. The container has no network, so the address can't point anywhere else.
5. Unknown or stopped addresses return 404. A private site returns 403 to anyone without your login or a valid ticket.

### How private sites work

A sandboxed page can't send your login, so the server uses a ticket instead:

1. You open a private site. The server checks your login.
2. It redirects you to `/run/<project-id>/~<ticket>/`. Relative links keep the ticket.
3. The ticket lasts 4 hours.
   - It works only for that site address and that browser.
   - It also checks the IP when `CLIENT_IP_HEADER` is set.
4. It stops working when the site restarts or the server restarts.

An address copied to another machine won't work. Still, treat an address with a ticket like a password while it's valid.
