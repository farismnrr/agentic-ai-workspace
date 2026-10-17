---
name: nuxt-app
description: How to build a new web app for the person: Nuxt, SQLite, a login form of its own and security built in, always on the latest stable versions.
always: true
---

# Building an app: the default stack

This is how every new app is built, unless the person or the PRD asks for something else (then do that) or the
project already has a stack (then keep it). The person may not be technical: make the safe choice yourself and
never ask them about libraries.

## 1. The stack
- Nuxt 4 with Vue 3 and TypeScript, one server and one process. SQLite for data. `nuxt-security` for headers, rate
  limits and request limits. Better Auth for login when the app has accounts. `zod` to check what comes in.
  `@nuxt/eslint` with `eslint-plugin-security` as the linter. Styling follows `DESIGN.md`; `@nuxt/ui` is fine.
- Always the latest stable. Never copy a version from memory or an old tutorial: run `npm view <package> version`
  and read its `engines` against `node -v` (the sandbox has Node 24). Install without pinning to an old number, keep
  `package-lock.json`, and read the current docs (nuxt.com, better-auth.com) for any API you are unsure of. If you
  cannot find it, put "Needs research: <question>" in your report instead of guessing.
- A database other than SQLite, or a login by Google or GitHub, only when the person asked for it.

## 2. Start the project
In the project folder: `npm create nuxt@latest -- . --template minimal --no-install --gitInit false --packageManager npm --no-modules --force`,
then `npm install nuxt-security better-auth zod` and `npm install -D @nuxt/eslint eslint eslint-plugin-security`.
Add the modules to `nuxt.config.ts`:
```ts
export default defineNuxtConfig({
  compatibilityDate: 'latest',
  modules: ['nuxt-security', '@nuxt/eslint'],
  runtimeConfig: { authSecret: '' }, // filled by NUXT_AUTH_SECRET
  security: { rateLimiter: { tokensPerInterval: 150, interval: 300000 } },
  routeRules: { '/api/auth/**': { security: { rateLimiter: { tokensPerInterval: 10, interval: 60000 } } } },
})
```
Make `eslint.config.mjs`:
`import withNuxt from './.nuxt/eslint.config.mjs'`, `import security from 'eslint-plugin-security'`,
`export default withNuxt(security.configs.recommended)`, and add the script `"lint": "eslint ."`.
Write the secret once: `printf 'NUXT_AUTH_SECRET=%s\n' "$(openssl rand -hex 32)" > .env`, and put `.env`, `data/`
and `.output` in `.gitignore`. A secret is never written into code, a page or a report.

## 3. Data with SQLite
`server/infrastructure/db.ts` (Node's built-in SQLite: nothing to compile, works offline; it is an adapter, see
`clean-code` for the layers):
```ts
import { DatabaseSync } from 'node:sqlite'
import { mkdirSync } from 'node:fs'
mkdirSync('data', { recursive: true })
export const db = new DatabaseSync('data/app.sqlite')
db.exec('PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;')
```
Keep your own tables in numbered `.sql` files and apply them in a Nitro plugin, tracking progress with
`PRAGMA user_version`. Queries are always prepared with `?` values (`db.prepare('SELECT * FROM items WHERE id = ?').get(id)`),
never built by joining strings.

## 4. Login (only when the app has accounts)
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

## 5. Safe by default
Check every body and query with `zod` (`readValidatedBody`, `getValidatedQuery`). Vue escapes text; never use
`v-html` on anything a visitor wrote. Secrets stay on the server (`runtimeConfig`, not `runtimeConfig.public`). Allow only
the HTTP methods a route needs. Limit file uploads by size and type. Show errors in plain words and never the stack.

## 6. Build and start
Install and build with `run_command` (`npm install`, `npm run lint`, `npm run build`); the app then runs from
`.output`. Start it with `start_site`, port 3000:
`node /usr/local/share/serve/nuxt-serve.ts node --env-file=.env .output/server/index.mjs`.
`nuxt-serve` makes Nuxt work under the address prefix the sites proxy uses; do not run `.output` directly.
In the preview the page has no cookies, so a login cannot stay signed in there. That is the preview's sandbox, not
a fault in the app: test accounts with `curl` against the API instead (sign up, sign in, then call a protected
route with the returned cookie) and say so in your report. Never weaken a security setting to make the preview work.
Before `start_site` the Security teammate must have checked the code; if you are told it has not, report back.

## 7. Report
Say in plain words what the app does, the stack, which commands you ran and how they ended, and what you could not check.
