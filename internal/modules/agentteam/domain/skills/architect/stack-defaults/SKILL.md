---
name: stack-defaults
description: The default stack and layout for a new app, and when to depart from it.
always: true
---

# Default stack for a new app

This is the default for a new web app with screens, not the only stack. An existing project keeps its own stack and
language (Next.js, SvelteKit, Rust, Go, Python, plain HTML...), a stack the person asked for is used, and something
that is not a web app gets the language that fits it. A Rust or Go service is never made into a Nuxt app. Design a new
web app on this stack unless one of those applies:
Nuxt 4 (Vue 3, TypeScript) as the single server and UI, Nuxt UI for every component (no hand-made buttons, inputs,
cards, tables or navigation), SQLite through Bun's built-in `bun:sqlite`, TypeScript everywhere (no JavaScript files), Bun as the runtime and package
manager, `nuxt-security`
for headers and limits, Better Auth for accounts with the app's own register and login forms, `zod` for input checks.
Everything is the latest stable release; check `bun pm view <package> version` and what each package requires before
you choose a library, and prefer ones that run on Bun.

Layout: pages and components in `app/`, HTTP routes in `server/api/`, helpers in `server/utils/`, SQL files in
`server/db/migrations/`, one `data/app.sqlite` file. One process, no separate backend, no ORM unless the data model
is large. Keep the design small: name the files to add, the tables, the routes and who may call each.

Depart from it only for a reason you can say in one sentence, or when the person asks (another database, another
language, a login by Google or GitHub). A second stack needs the Researcher's current documentation first.
Think about abuse as you design: who can read or change each record, what a visitor can send, what is stored.
