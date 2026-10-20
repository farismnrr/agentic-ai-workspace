---
name: delivery-rules
description: How to deliver a product in slices that look and work like a product, and the traps of the preview address, of fonts that cannot be downloaded and of tests that touch real data.
always: true
---

# Delivery rules

## 0. Keep the checks in proportion
A finished test is not a finished job. While you build, run the cheap check of what you touched (the type check, the
tests of those files) and go on. The full lint, the whole test suite, the production build and every audit run once,
in the last slice or when the brief asks. Do not run them after every slice, and do not start or restart the app
yourself (only DevOps does, when someone needs to look).

## 1. One slice at a time
Your brief names one slice of the PRD's todo (or less). Do that slice completely, run it, and report. If the brief
asks for the whole app, build the first coherent slice, say what is left in your report, and stop; the lead sends the
next one. A slice is done when a person could use it: the pages exist, the data is real, the errors are handled.

## 2. The shell comes first, and it uses the design
The first slice is the shell: layout, navigation, the landing page, register and login. Build it with the tokens and
classes of `DESIGN.md` and its stylesheet from the first file; do not invent colours, sizes or spacing. A page with
no styling applied is not finished, whoever styles it later.

## 3. It must look like a product
A web product has a public landing page, register and login, an app with a navigation shell, and a dashboard with
key numbers (big number, label, change), charts with real data, recent activity, quick actions and empty states
that say what to do first. Lists have a search or filter when they can grow, forms have clear labels and messages,
destructive actions ask first and every save shows feedback. Do not ship a page that is only a heading and a table.

## 4. The address prefix (any stack)
The app is opened under an address like `/run/<name>/` (a private app: `/run/<name>/~<ticket>/`), and the proxy
removes that prefix from every request. A link that starts with `/` therefore leaves the app and asks the platform,
which answers with its own page: the stylesheet does not apply, or the page is "not found". So:
- Use relative links everywhere: HTML (`href`, `src`, `action`), CSS `url()`, `fetch('./api/items')`, redirects.
  From a page at the root write `./style.css`; from `/admin/users/` write `../../style.css`, one `../` per folder.
  A page in a folder must end its address with `/`, or its relative links resolve one folder too high.
- Never write `/style.css`, `/api/x`, `href="/login"`, `window.location = '/'`, and never hard-code `/run/...`.
- The ticket is part of the prefix of a private app, so only relative links (and `BASE_PATH` for the server side)
  keep it; a link built from `BASE_PATH` by hand loses it for scripts and styles. Prefer relative.
- Vite and similar bundlers: set `base: './'` in `vite.config.ts` (relative assets), use hash or memory routing
  (`createWebHashHistory`), and `fetch('./api/...')`. Plain HTML or a server that renders pages: relative paths, and
  `redirect('./login')` or a `Location` header with a relative value.
- Express, Hono, Flask and the like: build every link and `Location` header relative; read nothing from the host.
- The platform is a safety net, not a licence. It moves a root link (`/style.css`) in the pages and stylesheets the app
  sends, and in `src`, `href`, `fetch`, `history.pushState` and the like that a page sets while it runs, under the
  address. It cannot see an address a script builds from pieces, text inside `.js` files, `location.assign` or a server
  that redirects to `/login` (that one is fixed on the way back). Write relative links anyway; do not lean on it.
- A client router in history mode (React Router, Vue Router, SvelteKit, Next.js) reads `location.pathname`, which holds
  the address, so it needs a base. Read `window.__SITE_BASE__` (the platform sets it before any script runs) for the
  router's `basename` or `base`. Next.js fixes its base at build time: build with `basePath` set to `/run/<project ID>`
  (the ID never changes; take it from an env var, never hard-code it in the source) and start the app through
  `bun /usr/local/share/serve/site-serve.ts <start command>` with `BASE_PATH=/run/<project ID>`, which also carries the
  ticket of a private app. Then write links the Next way (`next/link`, `usePathname`) and do not prefix anything by hand:
  a link prefixed twice is how an address becomes `/run/<id>/run/<id>`.
- The stylesheet must be the built one. Tailwind (`@tailwind`, `@import "tailwindcss"`, `@apply`) and Sass only work
  after the build: link the file the build wrote (`dist/`, `.output/public/`), never the source. A page that shows as
  plain text and blue links means the stylesheet is missing, empty, uncompiled or answered with a web page; `start_site`
  reports all four, and so does a link inside a stylesheet (`url()`, `@import`). Fix them before you report.
- A server built for a base path and started through `site-serve` (Nuxt, Next with `basePath`) handles its pages by
  itself; for Nuxt see `nuxt-app` for the rest. Every other server (Rust, Go, Python, Express...) runs as it is and
  writes relative links.
- You do not start the app: the DevOps teammate does. Its `start_site` loads the home page and the pages it links to,
  and lists every link that would fail. If the orchestrator sends you that list, fix it and report; never call the
  app ready while it is not empty. In your report give the command and port that start the app.

## 5. Never touch the real data in a test
Tests and scripts use their own temporary database (a file under `/tmp` or an in-memory one), and never delete,
reset or overwrite the project's real database or `.env`. A test that wipes real data is a bug even when it passes.

## 6. Keep the files readable
Write normal, formatted TypeScript: no minified one-liners, lines under about 120 characters, files one person can
read in one go, and no JavaScript files at all. The Architect and the next teammate must be able to read it.

## 7. Fonts come from npm, never from a font website (any stack)
The sandbox reaches package registries only, so a font cannot be downloaded from Google Fonts, a CDN or any other
website, in a build or in a command. Do not link `fonts.googleapis.com`, do not `curl` a font, and do not build with
a plugin that fetches fonts at build time. Every Google font is also an npm package from Fontsource, which works in
any stack and also keeps the font pinned and the page free of third-party requests:
- Name: the font's name in lower case with hyphens (`DM Serif Display` is `dm-serif-display`).
- With a bundler (Vite, Nuxt, Next, SvelteKit, Astro and the like): `<package manager> add @fontsource-variable/<name>`,
  or `@fontsource/<name>` when the family has no variable version, import its CSS once (`@import
  "@fontsource-variable/<name>";` in the main stylesheet, or `import` in the entry file), and use the family name
  `'<Name> Variable'` (a static package keeps `'<Name>'` and takes one import per weight: `@fontsource/<name>/400.css`).
- Plain HTML and CSS without a build: `bun add @fontsource-variable/<name>`, copy
  `node_modules/@fontsource-variable/<name>/files/<name>-latin-wght-normal.woff2` into the project (for a static
  package `files/<name>-latin-400-normal.woff2`, one file per weight) and write the `@font-face` yourself with a relative
  `url(./fonts/...)` and `font-display: swap`.
- Keep only the weights and the latin subset the page uses. Always end the stack with a system font.
- If a template or an example links Google Fonts, replace that link with the above.

## 8. Look at a page with `screenshot`, never by installing a browser
The sandbox already has a headless Chromium (`chromium-headless-shell`) and has no network. `npx playwright install`,
`puppeteer` downloads and `apt` for a browser all fail and cost a round. To see a page, call `screenshot` (the app has
to be running: only DevOps starts it, so report "Needs the app started" and go on). Its report lists the failed
requests, a page without styling, sideways scroll and console errors. A project that has Playwright tests runs them
with `PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1` (already set) and `launchOptions: { executablePath:
'/usr/bin/chromium-headless-shell', args: ['--no-sandbox'] }`; do not add `playwright` to a project that has no tests of
its own for a screenshot.
