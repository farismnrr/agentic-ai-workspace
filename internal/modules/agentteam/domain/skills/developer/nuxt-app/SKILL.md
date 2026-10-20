---
name: nuxt-app
description: The default way to build a new web app (when the project has no stack yet): Nuxt, SQLite, a login form of its own and security built in, always on the latest stable versions. Not for a project in another stack.
always: true
---

# Building an app: the default stack

This is how every new app is built, unless the person or the PRD asks for something else (then do that) or the
project already has a stack (then keep it). The person may not be technical: make the safe choice yourself and
never ask them about libraries.

## 1. The stack
- **TypeScript, never JavaScript.** Every file is `.ts` or `.vue` with `<script setup lang="ts">`; there are no
  `.js`, `.mjs` or `.cjs` files in the project (config files too: `nuxt.config.ts`, `eslint.config.ts`). Types are strict,
  and `any` is not used to silence an error.
- **Bun, not Node.** Bun installs the packages (`bun add`), runs the scripts and the tests (`bun test`), and runs the
  production server. Do not use npm, npx or `node`; use `bun`, `bunx` and `bun --bun run <script>`.
- Nuxt 4 with Vue 3, one server and one process. SQLite for data, through Bun's built-in `bun:sqlite`.
  **Nuxt UI (`@nuxt/ui`) for every component**, so the whole app is consistent: buttons, forms, cards, tables,
  navigation, modals, the landing sections and the dashboard shell all come from it, themed with the colours and
  fonts of `DESIGN.md`. `nuxt-security` for headers, rate limits and request limits. Better Auth for login when the
  app has accounts. `zod` to check what comes in. `@nuxt/eslint` with `eslint-plugin-security` as the linter.
  `chart.js` with `vue-chartjs` for charts.
- This is the default for a new web app. If the project already has code in another stack (Next.js, SvelteKit, Rust,
  Go, Python...) or the brief says the person asked for one, work in that stack and ignore the rest of this skill that
  is specific to Nuxt; the rules of `delivery-rules` still apply to every stack. Do not convert a project to Nuxt.
- Always the latest stable. Never copy a version from memory or an old tutorial: run `bun pm view <package> version`
  and check what a package requires before you pin anything old. Keep `bun.lock`. One exception you must know:
  TypeScript 7 has no `lib/tsc`, so `vue-tsc` (the Vue type checker) cannot run on it; install `typescript@6`, the
  newest that works, and say so. If another latest version breaks the build, step back one major, and report it.
  Read the current docs (nuxt.com, ui.nuxt.com, better-auth.com) for any API you are unsure of. If you cannot find
  it, put "Needs research: <question>" in your report instead of guessing.
- A database other than SQLite, or a login by Google or GitHub, only when the person asked for it.

## 2. Start the project
If the project has a `.template` file or `DESIGN.md` has a "Template" section, build the interface from that Nuxt UI
template (its `app/` is already in the project; read the rest in `/usr/local/share/templates/<name>/`; add the
modules its `package.json` lists, such as Nuxt Content or `@nuxt/image`): keep its pages, layouts and components as the Designer adapted them, and wire the
team's own backend behind them. Never take the template's login, server or data code.
Run `new-nuxt-app` in the project folder (with `run_command`). It copies the toolbox's Nuxt starter without overwriting a
file that exists, writes `package.json`, installs the latest stable packages with Bun, makes `.env` with a fresh
`NUXT_AUTH_SECRET`, and runs `nuxt prepare`. Do not use `bun create nuxt` or `npm create nuxt`: they download their
template from GitHub, which the sandbox cannot reach. What the starter already is, all in TypeScript:
- `nuxt.config.ts` with Nuxt UI, `nuxt-security` and `@nuxt/eslint`, the Bun preset and rate limits on the login routes;
  `app/app.vue` (`<UApp>`), `app/app.config.ts` (colours), `app/assets/css/main.css` (Tailwind, Nuxt UI and the font).
- `app/utils/brand.ts`: the one place that names the product. Replace its words with the PRD's.
- Layouts `default` (public header and footer) and `app` (the dashboard shell with a sidebar and sign out); pages
  `index` (landing: hero, benefits, pricing, call to action), `login`, `register` (working forms) and `app/index`
  (dashboard with an empty state); `app/middleware/auth.global.ts` (every `/app` page needs a session) and `app/error.vue`.
- `server/infrastructure/db.ts` (`bun:sqlite`) and `auth.ts` (Better Auth), `server/api/auth/[...all].ts`, `server/api/me.get.ts`
  and `server/plugins/migrate.ts`; `eslint.config.ts`, `tsconfig.json`, `tests/smoke.test.ts`, `.gitignore`.
It type-checks, lints, tests and builds as it is. Your slices turn it into the PRD's product: set the colours and the font
from `DESIGN.md` (install its Fontsource package, `bun add @fontsource-variable/<name>`, import it in `main.css`), replace the words and the sections of the landing page, add the feature pages and routes, and put the
real numbers on the dashboard. Keep what works (the login, the middleware, the layouts); change what the PRD changes.
Scripts: `typecheck`, `lint`, `test`, `build`. Add packages the PRD needs with `bun add <package>`. Bun does not run a
dependency's install script unless it is listed under `trustedDependencies` in `package.json`; add one only when a
package really needs it (a native module). A secret is never written into code, a page or a report; `.env`, `data/` and
`.output` are already in `.gitignore`.

## 3. Data with SQLite
The starter already has `server/infrastructure/db.ts` (below). Your tables and queries go beside it.
`server/infrastructure/db.ts` (Bun's built-in SQLite: nothing to compile, works offline; it is an adapter, see
`clean-code` for the layers):
```ts
import { Database } from 'bun:sqlite'
import { mkdirSync } from 'node:fs'

mkdirSync('data', { recursive: true })

export const db = new Database('data/app.sqlite', { create: true, strict: true })
db.exec('PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;')
```
Keep your own tables in numbered `.sql` files and apply them in a Nitro plugin, tracking progress with
`PRAGMA user_version`. Queries are always prepared with `?` values (`db.prepare('SELECT * FROM items WHERE id = ?').get(id)`),
never built by joining strings.

## 4. Login (only when the app has accounts)
The starter already has all of this working; read it before you change it, and keep it when the PRD has accounts. If
the PRD has no accounts, remove the login pages, the middleware and the auth files together.
The app has its own register and login forms, in the app's own design, backed by Better Auth.
- `server/infrastructure/auth.ts`:
  `const config = useRuntimeConfig()` and
  `export const auth = betterAuth({ database: db, secret: config.authSecret, basePath: `${config.app.baseURL.replace(/\/$/, '')}/api/auth`, emailAndPassword: { enabled: true, minPasswordLength: 10 } })`
  with `import { betterAuth } from 'better-auth'` and `import { db } from './db'`. The `basePath` must include the app's
  base URL: under the sites proxy the app lives below an address prefix, and without it every login route answers 404.
- `server/api/auth/[...all].ts`: `import { auth } from '../../infrastructure/auth'` and
  `export default defineEventHandler((event) => auth.handler(toWebRequest(event)))`.
- `server/plugins/migrate.ts`: `import { getMigrations } from 'better-auth/db/migration'` and the same `auth` import; in a
  `defineNitroPlugin`, run `const { runMigrations } = await getMigrations(auth.options); await runMigrations()`.
- Forms post to `${useRuntimeConfig().app.baseURL}api/auth/sign-up/email` (`{ name, email, password }`) and
  `.../sign-in/email` (`{ email, password }`) with `$fetch`, and show a plain-language error. Sign out is
  `.../sign-out`.
- Protect a page or an API by reading the session on the server: `await auth.api.getSession({ headers: event.headers })`
  and answer 401 when it is null. Never trust a check that exists only in the browser.
- Passwords are hashed by Better Auth; never store, log or return one. Keep the rate limit on `/api/auth/**`.

## 4b. The pages of a product
The starter has the shell below with placeholder words; your slices fill it with the PRD's content and add the feature pages.
A product is not one page. Use two layouts and these routes unless the PRD says otherwise:
- `app/layouts/landing.vue` (public header, footer) for `app/pages/index.vue`, the landing page: hero with the value in
  one line and one call to action, benefits, how it works, pricing or the call to action again, FAQ, footer.
- `app/pages/login.vue` and `app/pages/register.vue`: a centred card, labelled fields, one primary button, plain-language
  errors, a link to the other page. A signed-in visitor is sent on to the app.
- `app/layouts/app.vue` (sidebar or top navigation, the person's name, sign out) for everything under `app/pages/app/`.
  `app/pages/app/index.vue` is the dashboard: three or four number cards (big number, label, change), one or two charts
  of real stored data, recent activity, quick actions and empty states that say what to do first. Then one page per
  feature (list, create, edit, delete), `settings.vue`, and `app/error.vue` as the 404.
- A route middleware in `app/middleware/` sends a visitor without a session to `/login` for every `/app` page; the API
  still checks the session on the server as well.

Links and files under the address prefix. The app lives at `/run/<name>/` and Nuxt's `app.baseURL` carries that
prefix (set by `site-serve`), so a hand-written address starting with `/` leaves the app:
- Pages: `<NuxtLink to="/login">`, `navigateTo('/app')` and `useRoute` all add the base themselves. Never write
  `<a href="/login">` for an app page.
- Calls to the server, as the starter does: `$fetch(`${useRuntimeConfig().app.baseURL}api/...`)` or
  `useFetch('api/me', { baseURL: useRuntimeConfig().app.baseURL })`; never `fetch('/api/...')`.
- Pictures and other files from `public/`: `useRuntimeConfig().app.baseURL + 'logo.svg'`, or `import` them from
  `app/assets/` so the build adds the base. CSS `url()` points only at files under `app/assets/` (Vite rewrites those).
- Never hard-code `/run/...` and never turn `app.baseURL` into a relative path (`./`); Nuxt does not support that on a
  server. The DevOps teammate's `start_site` loads your pages and reports any link that would not work: fix every
  one of them when you are sent the list.

## 5. Safe by default
Check every body and query with `zod` (`readValidatedBody`, `getValidatedQuery`). Vue escapes text; never use
`v-html` on anything a visitor wrote. Secrets stay on the server (`runtimeConfig`, not `runtimeConfig.public`). Allow only
the HTTP methods a route needs. Limit file uploads by size and type. Show errors in plain words and never the stack.

## 6. Build and start
Check with `run_command`, one step per call (a command has about 2 GB of memory, and an install followed by a type
check in the same call can run out). In a slice that is not the last (the brief says which), run only
`bun install` when you added packages, `bun --bun run typecheck` and the tests of what you touched
(`bun test <path>`): that is enough to hand the slice on. In the last slice, or when the brief asks for the full
check, run `bun --bun run typecheck`, `bun --bun run lint`, `bun test` and `bun --bun run build`; every one must pass
before you report. The app then runs from `.output`. You do not start it: the DevOps teammate does, and your report says so.
Give it the start command, port 3000: `bun /usr/local/share/serve/site-serve.ts bun .output/server/index.mjs`.
`site-serve` makes Nuxt work under the address prefix the sites proxy uses; do not run `.output` directly.
In the preview the page has no cookies, so a login cannot stay signed in there. That is the preview's sandbox, not
a fault in the app: test accounts with `curl` against the API instead (sign up, sign in, then call a protected
route with the returned cookie) and say so in your report. Never weaken a security setting to make the preview work.
The Security teammate checks the code once, when the work is done; it does not hold up starting the app while you build.

## 7. Report
Say in plain words what the app does, the stack, which commands you ran and how they ended, and what you could not check.
