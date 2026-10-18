---
name: stack-defaults
description: The default stack and layout for a new app, and when to depart from it.
always: true
---

# Default stack for a new app

Design new apps on this stack unless the person asks for another or the project already has one:
Nuxt 4 (Vue 3, TypeScript) as the single server and UI, Nuxt UI for every component (no hand-made buttons, inputs,
cards, tables or navigation), SQLite through Node's built-in `node:sqlite`, `nuxt-security`
for headers and limits, Better Auth for accounts with the app's own register and login forms, `zod` for input checks.
Everything is the latest stable release; check `npm view <package> version` and the `engines` of each package against
the sandbox's Node 24 before you choose a library.

Layout: pages and components in `app/`, HTTP routes in `server/api/`, helpers in `server/utils/`, SQL files in
`server/db/migrations/`, one `data/app.sqlite` file. One process, no separate backend, no ORM unless the data model
is large. Keep the design small: name the files to add, the tables, the routes and who may call each.

Depart from it only for a reason you can say in one sentence, or when the person asks (another database, another
language, a login by Google or GitHub). A second stack needs the Researcher's current documentation first.
Think about abuse as you design: who can read or change each record, what a visitor can send, what is stored.
