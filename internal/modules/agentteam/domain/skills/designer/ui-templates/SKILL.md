---
name: ui-templates
description: Start a new app's interface from an official Nuxt UI template instead of designing from zero, and adapt it when the request differs.
always: true
---

# Start from a template, not from zero

When the project is new or empty, do not design the interface from scratch. The toolbox holds the official Nuxt UI
templates (MIT, Nuxt UI v4) in `/usr/local/share/templates/<name>/`, read-only. They are a **UI reference only**:
take their pages, layouts, sections and components. Never take their backend, login, database or server code: those
follow the team's own stack (`stack-defaults`, `nuxt-app`). You cannot open the internet; read the files.

## 1. Pick the closest one
| Template | Use it for |
|---|---|
| `saas` | A product: landing, pricing, docs, blog, changelog, login and signup pages |
| `landing` | One marketing page: features, pricing, testimonials, FAQ |
| `dashboard` | The app behind the login: sidebar, inbox, customers, settings, charts, date pickers |
| `docs` | Documentation with generated navigation and search |
| `chat` | An AI chat screen with history |
| `portfolio` | Projects, blog posts and talks of a person or studio |
| `changelog` | Release notes |
| `editor` | A rich text editor (TipTap) |
| `calendar` | Day, week and month views with drag and drop |

A web product usually combines two: `saas` or `landing` for the public side and `dashboard` for the app. Run
`ls` on the template, read `app/pages`, `app/layouts`, `app/components` and `app/app.config.ts`, and read its
`package.json` for the modules it needs.

## 2. Adapt it when the request is different
Most requests are not the template. Keep what fits and change the rest:
- Swap, drop, reorder or add sections so the page tells this product's story; write the product's own words.
- Retheme through `ui.colors` in `app/app.config.ts` and `@theme` in `main.css` from your `DESIGN.md`, not by
  editing components. Keep every control a Nuxt UI component.
- Replace the template's demo pictures and logos: list what is needed under "Assets needed" (see the
  `designer-workflow` skill). Remove demo data, links and promotional pieces such as `TemplateMenu.vue`.
- Do not copy the template's demo copy, names or numbers into the product.

## 3. Record it
In `DESIGN.md` add a "Template" section: which template, which pages and components are kept, what you changed and
what you dropped. The Developer builds from this, wiring the real backend behind the same interface.
