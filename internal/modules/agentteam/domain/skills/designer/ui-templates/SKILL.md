---
name: ui-templates
description: Adapt the Nuxt UI template a project was started from, or read the templates as inspiration when it has none.
always: true
---

# Start from a template, not from zero

A person can start a project from a template. Then the project folder has a `.template` file with its name, and the
template's `app/` and `public/` are already copied in: adapt that template (sections 2 and 3), do not pick or copy
another, and read the rest of it in `/usr/local/share/templates/<name>/` if you need to. When there is no
`.template` file the person started from empty: design with your skills and use the templates below only as
inspiration to read, never copy them in. The toolbox holds the official Nuxt UI
templates (MIT, Nuxt UI v4) in `/usr/local/share/templates/<name>/`, read-only. They are a **UI reference only**:
take their pages, layouts, sections and components. Never take their backend, login, database or server code: those
follow the team's own stack (`stack-defaults`, `nuxt-app`). You cannot open the internet; read the files.

## 1. The templates
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

Run `ls` on the template, read `app/pages`, `app/layouts`, `app/components` and `app/app.config.ts`, and read its
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
