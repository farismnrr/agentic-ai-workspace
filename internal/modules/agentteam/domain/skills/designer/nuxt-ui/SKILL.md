---
name: nuxt-ui
description: Design and style a Nuxt app with Nuxt UI components, so every screen is consistent, instead of writing component CSS by hand.
always: true
---

# Designing with Nuxt UI

When the project is a Nuxt app (the default for anything new), the interface is made of Nuxt UI components. Do not
hand-write buttons, inputs, cards, tables, menus, modals or page sections: use the component, and spend your
effort on the design decisions: the colours, the type, the layout, the order, the copy and the states.

For a new project the sections come from an official template: read the `ui-templates` skill first.

## 1. Theme it from DESIGN.md
- Colours: map the direction's accent to a Tailwind palette name in `app/app.config.ts`
  (`ui.colors.primary`, and `neutral`). If the brand needs a colour Tailwind lacks, define all shades 50 to 950 in
  `main.css` under `@theme static` first.
- Type: set `--font-sans` (and `--font-mono`) in `main.css` under `@theme`.
- Beyond that, change a component through its `ui` prop or `app.config.ts` (`ui.button.defaultVariants`), not by
  overriding its CSS. Write plain CSS only for what no component covers (a hero illustration, a custom grid).
- Keep the tokens in `DESIGN.md` and say which Nuxt UI settings carry them.

## 2. Components for each part of a product (verified in a built app)
- **Landing page**: `UHeader` + `UNavigationMenu` (links) with `UButton` actions in its `#right` slot; `UMain`;
  `UPageHero` (title, description, `:links`); `UPageSection` (`:features` with icons) for benefits; `UPricingPlans`
  for plans; `UPageCTA` to close; `UFooter`. Alternate the sections; give each one a purpose.
- **Register and login**: `UPageCard` holding a `UAuthForm` (title, description, `:fields`, `:submit`), centred on the page.
- **The app behind the login**: `UDashboardGroup` > `UDashboardSidebar` (with a vertical `UNavigationMenu`) and
  `UDashboardPanel` (`#header` with `UDashboardNavbar`, `#body` for the page). This is the shell of every app page.
- **Dashboard**: a grid of `UCard` number cards (small muted label, large figure, a `UBadge` for the change); a `UCard`
  holding the chart (`vue-chartjs`, inside `<ClientOnly>`); recent activity as a list or `UTable`; `UEmpty` (icon,
  title, a description that says the first step) when there is nothing yet.
- **Forms**: `UForm` with `UFormField` and `UInput`, `USelect`, `UTextarea`, `UCheckbox`; validation messages from the
  schema. **Feedback**: `useToast()` after a save, `UAlert` for a notice, `UModal` to confirm a delete.
- **Lists and data**: `UTable` for rows, with search in a `UInput` above it; `UBadge` for statuses (text, not colour only).
If you are not sure of a component's props, put "Needs research: Nuxt UI <component> props" in your report instead of
guessing: the Researcher can read ui.nuxt.com.

## 3. Rules
- One look across every page: same shell, same spacing, same component variants. A page that uses its own
  hand-made button next to `UButton` is a defect.
- Icons are `i-lucide-<name>`, bundled locally (`@iconify-json/lucide`).
- Check dark mode only if the design asks for it; Nuxt UI supports it, and nothing else needs doing.
- Looking at the result is still required: screenshot every page, desktop and mobile, before you report.
