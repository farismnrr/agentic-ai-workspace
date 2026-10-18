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
The app is opened under an address like `/run/<name>/<uuid>/`, and the proxy removes that prefix from the request. So
links, forms, stylesheets, scripts and redirects must be relative (`./css/style.css`, `./admin`), or follow
`BASE_PATH` (the prefix, in the environment), never start with `/`. Check by loading a page two folders deep. Nuxt
under `nuxt-serve` already handles this.

## 5. Never touch the real data in a test
Tests and scripts use their own temporary database (a file under `/tmp` or an in-memory one), and never delete,
reset or overwrite the project's real database or `.env`. A test that wipes real data is a bug even when it passes.

## 6. Keep the files readable
Write normal, formatted TypeScript: no minified one-liners, lines under about 120 characters, files one person can
read in one go, and no JavaScript files at all. The Architect and the next teammate must be able to read it.
