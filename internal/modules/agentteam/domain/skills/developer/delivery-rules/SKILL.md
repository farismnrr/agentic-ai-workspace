---
name: delivery-rules
description: How to deliver a product in slices that look and work like a product, and the traps of the preview address and of tests that touch real data.
always: true
---

# Delivery rules

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
- Nuxt under `nuxt-serve` handles pages by itself, but its own rules apply (see `nuxt-app`).
- `start_site` loads the home page and the pages it links to, and lists every link that would fail. Fix the list and
  start again; do not report the app as ready while it is not empty.

## 5. Never touch the real data in a test
Tests and scripts use their own temporary database (a file under `/tmp` or an in-memory one), and never delete,
reset or overwrite the project's real database or `.env`. A test that wipes real data is a bug even when it passes.

## 6. Keep the files readable
Write normal, formatted TypeScript: no minified one-liners, lines under about 120 characters, files one person can
read in one go, and no JavaScript files at all. The Architect and the next teammate must be able to read it.
