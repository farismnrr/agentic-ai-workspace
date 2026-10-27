# Design system

The visual language of the workspace. It keeps the Masih Awam brand recognizable without copying the LMS implementation or technology.

## Principles

1. **Keep the identity:** paper-like surfaces, olive or lime as the main color, cool supporting accents, rounded shapes, restrained motion.
2. **Use roles, not colors:** semantic roles in product UI, never literal colors.
3. **Hierarchy first:** build it with surfaces, type, spacing and borders before adding color.
4. **One theme:** `masihawam-light` is the single source of color meaning. No dark theme, no toggle.
5. **Reuse in order:** local component, then daisyUI, then Tailwind utilities, then custom CSS only if nothing else covers it.
6. **Accessibility is part of the theme:** visible focus, enough contrast, reduced motion, status never by color alone.

## Color

The brand trio is split-complementary, anchored on the olive primary: olive, then blue, then violet.

| Product role | daisyUI role | Color | Use |
| --- | --- | --- | --- |
| Primary | `primary` | `#657C1A` | Main action, current selection |
| Secondary | `secondary` | `#275391` | Secondary actions, AI emphasis |
| Tertiary | `accent` | `#77358D` | Contextual actions |
| Neutral | `neutral` | `#2B2A2D` | Neutral controls, strong surfaces |
| Info | `info` | `#0F6F84` | Informational feedback |
| Success | `success` | `#2F7D4A` | Successful operations |
| Warning | `warning` | `#A45400` | Risk, caution |
| Error | `error` | `#B4233A` | Errors, destructive actions |

- **Status colors** keep conventional hues. They are not forced into the brand trio.
- **Primary is a brand role, success is a status role.** Never show success with olive or the lime marker alone.
- **Theme values** live in `web/src/app.css`. Feature components must not redefine them.
- **Document viewer.** The page of a document site, made by `sandbox/docsite.py`, copies the theme values because a site cannot import them. Change both together.

## Surfaces and text

| Surface | Role |
| --- | --- |
| `base-200` | Page canvas |
| `base-100` | Panels and cards |
| `base-300` | Stronger separation |

Shared CSS variables cover what the roles do not: `--brand-surface`, `--brand-surface-soft`, `--brand-surface-strong`, `--brand-border`, `--brand-border-strong`, `--brand-shadow`.

Three levels of text:

1. **Primary:** `base-content`.
2. **Secondary:** `--brand-text-secondary`, or a `base-content` opacity.
3. **Tertiary:** `--brand-text-tertiary`, for low-emphasis details only. Never for instructions, errors or important state.

## Components

- **Use semantic classes:** `btn-primary`, `btn-secondary`, `btn-accent`, `alert-*`, `bg-base-100`, `text-base-content`.
- **Avoid literal colors** such as `bg-lime-*` or hard-coded brand hex inside features.
- **Literal colors are fine** for inherently visual content: charts, generated previews, media.

### Interaction states

- A control may be default, hover, active, focus-visible, selected, disabled or loading.
- These are states, not extra brand colors. Disabled keeps its role, with less emphasis and no interaction.
- Use the shared focus style. Never remove a visible focus indicator without an accessible replacement.

### Shape and depth

- **Controls** use a medium radius. **Cards and panels** use a larger one.
- **Pills** are for badges, compact selectors, and controls where shape shows grouping.
- **Borders and surface contrast** come before shadows.

### Popup menus

The account menu and the command palette share one look:

- `bg-base-100`, a hairline `--brand-border`, a medium radius, the `--brand-shadow` token (never a generic Tailwind shadow).
- A leading icon on every row, hairlines between groups, and Sign out in a group of its own.
- They close on an outside click and on Escape, and focus returns to the trigger.

### Empty states

Every empty list, panel or page uses the shared empty state:

- A small quiet icon, a one-line title that says what is missing, and a short reason with the next step.
- Compact variants inside cards and panels.
- One primary button, only when it solves the emptiness.
- A search or filter with no results says so and suggests changing it. A list you cannot fill yourself gets an explanation, no button.

## Typography

- **UI font:** Inter Variable, self-hosted (Latin subset), platform sans-serif as fallback.
- **Headings** (`h1` to `h4`): Newsreader Variable (`font-display`), self-hosted, for an editorial tone.
- **Everything else** (navbar, wordmark, body, buttons, fields) uses Inter.
- **Set once:** heading fonts live in the base layer, so components do not repeat them.
- **Metric-matched fallbacks** stop text from jumping while fonts load.

## Brand details

- **Paper:** the landing page uses a flat `base-200` canvas, `base-100` cards and hand-drawn line illustrations. No gradients or glows.
- **Marker:** emphasis inside a heading uses the shared `Marker` component, a lime highlighter stroke driven by `--brand-marker`.
- **Marketing buttons:** primary buttons may be pill-shaped.
- **Logo:** a rounded primary-olive tile with a white "o..n" mark, beside the Inter wordmark "Masih Awam". Favicon and manifest icons in `web/public/` use the same tile.

## Motion

- **Short and functional:** brief transitions for hover, focus, expansion and state changes.
- **Scroll reveals:** sections fade in once. The hero loads with a short rise.
- **No JavaScript needed:** content is always visible.
- **Reduced motion:** the base layer shortens nonessential animation, and reveals only hide content when motion is allowed.

## Screens

### Sign-in and create account

- `/login` and `/register` split 50/50: the form on the left, a quiet illustration panel with a serif caption on the right.
- The panel hides below `lg`, so phones get a full-width form.
- Forms use daisyUI inputs and the primary pill button.
- Every field has a visible label. Errors show under the field once it is touched or the form is submitted.
- Password fields have a real show/hide button. The strength meter always pairs its bars with a text label.

### Error pages

- `/not-found` (404) and `/forbidden` (403) are one centred column: logo, hand-drawn illustration, "Error 404/403", a serif headline with one marker word, one plain sentence, two buttons.
- Any unknown address opens the 404 page. Private sites send people to the 403 page.
- Both carry real status codes and `noindex`.

### Workspace

The signed-in app lives under `/app` and uses a compact scale: 14px text, small radii, sans headings, flat rows with hairlines, no card shadows. Only the composer and popups float.

Forms and detail sections reuse `SettingsSection` and `FormField`; lists stay
flat and use shared empty states instead of nested cards.

- **Pages:** Home, Inbox, Projects, a project, a task, Settings.
- **Sidebar:** icon rail on desktop, drawer on tablets.
- **Phones:** below the `md` breakpoint, a bottom bar with Home, Inbox, Projects, Sites and Settings (daisyUI `dock`, with counts). The menu button still opens the sidebar.
- **Task page:** the bottom bar steps aside, so the conversation keeps the whole screen.
- **Status:** every status has a distinct icon and readable label, never color
  alone. Icon-only uses retain an accessible name.
- **Approvals:** show the exact action, a risk label, and Allow or Don't allow.
- **Task result:** summarizes verified checks and AI claims in plain language.
  "Proven to work" appears only for verified, passing checks.
- **No fakes:** screens never show sample data or pretend buttons. Destructive actions ask for confirmation.

## Accessibility

- Target WCAG AA contrast for normal text.
- Keep focus indicators easy to see.
- Never show success, warning or error by color alone. Add text, an icon or a label.
- Status updates remain understandable without color, and working animations
  honor reduced-motion preferences.
- Timeline labels identify whether an action was performed by ChatGPT/Gemini,
  recorded by Masih Awam, approved by you, or executed by the Orchestrator.
- External assistant reports are labelled as reported. Only evidence observed
  or run by the server can be presented as verified.

## Source of truth

- **Theme values:** `web/src/app.css`.
- **Font faces:** `web/src/shared/theme/fonts.css`.
- **Reusable visuals:** `web/src/shared/ui/`.

When the visual language changes, update this guide and the theme together.
