# Design System

This guide defines the visual language for the AI workspace. It keeps the Masih Awam brand recognizable without copying the LMS implementation or technology choices.

## Principles

1. Keep the Masih Awam identity: warm near-black or paper-like surfaces, lime or olive as the main brand color, cool supporting accents, rounded geometry, and restrained motion.
2. Use semantic roles instead of literal colors in product UI.
3. Build hierarchy with surfaces, typography, spacing, and borders before adding more color.
4. Keep one theme, `masihawam-light`, as the single source of color meaning.
5. Prefer local reusable components, then daisyUI, then Tailwind utilities, and add custom CSS only when the design system needs a capability they do not provide.
6. Accessibility is part of the theme contract: visible focus, sufficient contrast, reduced-motion support, and status cues that do not rely on color alone.

## Color harmony

The brand palette uses a split-complementary relationship anchored on the Masih Awam primary hue.

- Primary is olive.
- Secondary moves into the blue family.
- Tertiary moves into the violet family.
- The order is intentionally `olive/lime -> blue -> violet`, so supporting emphasis progresses in one visual direction instead of jumping back and forth around the color wheel.

Status colors are not forced into the brand harmony. `info`, `success`, `warning`, and `error` keep conventional semantic hue families so users can distinguish meaning quickly.

## Theme names

The app ships with one custom daisyUI theme, `masihawam-light`. There is no dark theme or theme toggle.

## Semantic colors

| Product role | daisyUI role | Color | Typical use |
| --- | --- | --- | --- |
| Primary | `primary` | `#657C1A` | Main CTA, current selection, strongest brand emphasis |
| Secondary | `secondary` | `#275391` | Secondary actions, agent or AI emphasis |
| Tertiary | `accent` | `#77358D` | Contextual actions, tertiary emphasis |
| Neutral | `neutral` | `#2B2A2D` | Neutral controls and strong neutral surfaces |
| Info | `info` | `#0F6F84` | Informational feedback |
| Success | `success` | `#2F7D4A` | Successful operations |
| Warning | `warning` | `#A45400` | Risk, caution, attention |
| Error | `error` | `#B4233A` | Errors and destructive actions |

`accent` is the implementation name for the product's tertiary role.

Primary is a brand/action role. Success is a status role. Do not use primary olive or the lime marker alone to communicate successful completion.

## Base and surfaces

Use daisyUI base roles first:

- `base-200`: page canvas
- `base-100`: panels and cards
- `base-300`: stronger neutral separation
- `base-content`: primary text

The theme also exposes shared CSS variables when daisyUI roles are not specific enough:

- `--brand-text-secondary`
- `--brand-text-tertiary`
- `--brand-surface-soft`
- `--brand-surface`
- `--brand-surface-strong`
- `--brand-border`
- `--brand-border-strong`
- `--brand-shadow`

Feature components should not redefine these values.

## Text hierarchy

Use three levels of text emphasis:

1. Primary text: `base-content`
2. Secondary text: `--brand-text-secondary` or a suitable `base-content` opacity
3. Tertiary text: `--brand-text-tertiary` for low-emphasis metadata

Do not use tertiary text for critical instructions, validation messages, or important state.

## Interaction states

Color roles and interaction states are separate concepts.

Each interactive control may have:

- default
- hover
- active
- focus-visible
- selected
- disabled
- loading

Disabled is not a brand color. Keep the original semantic role understandable while reducing emphasis and preventing interaction.

Use the shared focus treatment. Never remove a visible focus indicator without an accessible replacement.

## Components

Use semantic classes such as:

- `btn-primary`
- `btn-secondary`
- `btn-accent`
- `alert-info`
- `alert-success`
- `alert-warning`
- `alert-error`
- `bg-base-100`
- `bg-base-200`
- `text-base-content`

Avoid literal product colors such as `bg-lime-*`, `text-violet-*`, or hard-coded Masih Awam hex values inside feature components.

Literal colors are acceptable only for content that is inherently visual, such as charts, generated previews, or media where semantic UI roles do not apply.

## Typography

The UI font is Inter Variable, self-hosted (Latin subset), with the platform sans-serif stack as fallback.

Headings (`h1`–`h4`) use Newsreader Variable (`font-display`), also self-hosted, for an editorial tone. Everything else, including the navbar and footer wordmark, body text, buttons, and form fields, uses Inter, which prioritizes readability and scanning speed. Heading fonts are set once in the base layer, so components do not repeat them. Metric-matched fallback fonts keep text from jumping while the web fonts load.

## Paper and marker

The landing page reads like paper: flat `base-200` canvas, `base-100` cards, hand-drawn line illustrations, and no decorative gradients or glows. Emphasis inside a heading uses the shared `Marker` component, a lime highlighter stroke driven by `--brand-marker`. Primary buttons on marketing pages may be pill-shaped.

## Shape and depth

The brand uses rounded controls and panels, but not every element should become a pill.

- Fields and controls use a medium radius.
- Cards and panels use a larger radius.
- Pills are reserved for badges, compact selectors, and controls where the shape communicates grouping.
- Use borders and surface contrast before adding heavy shadows.

## Motion

Motion should be restrained and functional.

Use short transitions for hover, focus, expansion, or state changes. Sections fade in once as they scroll into view, and the hero loads in with a short rise. Content is always visible without JavaScript. Respect `prefers-reduced-motion`; the global base layer already collapses nonessential animation and transition duration when reduction is requested, and scroll reveals only hide content when motion is allowed.

## Logo and icons

The logo is a rounded primary-olive tile with a white "o..n" mark, set beside the Inter wordmark "Masih Awam". The favicon, touch icon, and manifest icons live in `web/public/` and use the same tile.

## Sign-in and create-account screens

`/login` and `/register` use a 50/50 split: logo, a serif headline with one marker highlight, a short subtitle, and the form on the left; on the right, a quiet panel with one hand-drawn illustration and a short serif caption. The panel hides below the `lg` breakpoint so the form gets the full width on phones.

Forms use daisyUI inputs and the primary pill button. Every field has a visible label, errors appear as text under the field after it is touched or the form is submitted, and the password fields have a real show/hide button. The password strength meter always pairs its bars with a text label. These screens are UI only for now: nothing is sent to a server.

## Workspace screens

The signed-in workspace lives under `/app`: Home, Inbox, Projects, a project (Tasks, Activity, About), a task, and Settings. It uses a compact app scale rather than the landing-page scale: 14px text, small radii, sans headings, flat rows separated by hairlines, and no card shadows (only the composer and popovers float). Colors are exactly the theme's roles.

- **Sidebar:** New task, Search (Ctrl/Cmd+K), Inbox, All projects, then recent tasks grouped by day with a status dot and relative time, and a collapsible Projects list. Settings and the account menu sit at the bottom. It collapses to an icon rail on desktop and becomes a drawer on phones.
- **Home:** the app opens on the composer (project picker, attach, how much it should ask you, send), then a short "Needs you" list and what is running. The greeting is the one serif moment.
- **Inbox:** a list beside a detail pane. It gathers what needs the user (a permission request, a question, a finished task to review, a task that couldn't finish). It is not a global task list.
- **Task:** a conversation next to a work panel. The user's message is a soft bubble; the assistant's reply is plain text. Consecutive actions fold into one expandable line, and only questions, permission requests, and results appear as cards. The work panel has Plan, Proof (before and after, what was checked), Changes, Preview, and Files. On phones these become tabs. The composer sits at the bottom.

- **Settings:** a grouped left nav (AI, Preferences, Account) beside sections made of plain rows: the label and a short description on the left, the control on the right, hairline dividers, no cards around single controls. Adding a provider opens a dialog. Pages with editable values end with a sticky Save bar. On phones the nav becomes a scrolling row of tabs.

Task status is always an icon plus a text label (Waiting to start, Making a plan, Working on it, Needs your answer, Ready for your review, Done, Needs attention), never color alone. Permission requests show the exact action, a risk label, and scoped choices (Allow once, Always allow in this project, Don't allow). Project tiles use the brand trio through `primary`, `secondary`, and `accent`. These screens are a preview with sample data: buttons that would save or send show a short "preview" notice instead of pretending to succeed.

## Accessibility

For normal text, target WCAG AA contrast. Focus indicators must remain easy to see in both themes.

Do not communicate success, warning, or error by color alone. Pair important states with text, iconography, labels, or another structural cue.

## Source of truth

Theme values live in `web/src/app.css`.

Self-hosted font faces live in `web/src/shared/theme/fonts.css`.

Reusable visual components live in `web/src/shared/ui/`.

When the visual language changes, update this guide and the theme source together.
