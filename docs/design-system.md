# Design System

This guide defines the visual language for the AI workspace. It keeps the Masih Awam brand recognizable without copying the LMS implementation or technology choices.

## Principles

1. Keep the Masih Awam identity: warm near-black or paper-like surfaces, lime as the main brand color, violet and cyan as supporting accents, rounded geometry, and restrained motion.
2. Use semantic roles instead of literal colors in product UI.
3. Build hierarchy with surfaces, typography, spacing, and borders before adding more color.
4. Keep light and dark themes equivalent in meaning, not necessarily identical in raw color values.
5. Prefer local reusable components, then daisyUI, then Tailwind utilities, and add custom CSS only when the design system needs a capability they do not provide.
6. Accessibility is part of the theme contract: visible focus, sufficient contrast, reduced-motion support, and status cues that do not rely on color alone.

## Theme names

The app ships with two custom daisyUI themes:

- `masihawam-light`
- `masihawam-dark`

The first visit follows the operating-system color preference. A manual theme change is saved in the browser for later visits.

## Semantic colors

| Product role | daisyUI role | Light | Dark | Typical use |
| --- | --- | --- | --- | --- |
| Primary | `primary` | `#657C1A` | `#C8FF4D` | Main CTA, current selection, strongest brand emphasis |
| Secondary | `secondary` | `#6F56B3` | `#A78BFA` | Secondary actions, agent or AI emphasis |
| Tertiary | `accent` | `#147A91` | `#7CE7FF` | Contextual actions, tertiary emphasis |
| Neutral | `neutral` | `#2B2A2D` | `#2A2A30` | Neutral controls and strong neutral surfaces |
| Info | `info` | `#16697A` | `#55D9F0` | Informational feedback |
| Success | `success` | `#2F7D4A` | `#6EE7A0` | Successful operations |
| Warning | `warning` | `#A45400` | `#FF9F43` | Risk, caution, attention |
| Error | `error` | `#B4233A` | `#FF6B6B` | Errors and destructive actions |

`accent` is the implementation name for the product's tertiary role.

Primary is a brand/action role. Success is a status role. Do not use primary lime to communicate successful completion.

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

The default UI font stack is Inter when available, then the platform sans-serif stack.

The brand may use editorial typography for marketing or high-level storytelling, but workspace controls and dense product surfaces should prioritize readability and scanning speed.

## Shape and depth

The brand uses rounded controls and panels, but not every element should become a pill.

- Fields and controls use a medium radius.
- Cards and panels use a larger radius.
- Pills are reserved for badges, compact selectors, and controls where the shape communicates grouping.
- Use borders and surface contrast before adding heavy shadows.

## Motion

Motion should be restrained and functional.

Use short transitions for hover, focus, expansion, or state changes. Respect `prefers-reduced-motion`; the global base layer already collapses nonessential animation and transition duration when reduction is requested.

## Accessibility

For normal text, target WCAG AA contrast. Focus indicators must remain easy to see in both themes.

Do not communicate success, warning, or error by color alone. Pair important states with text, iconography, labels, or another structural cue.

## Source of truth

Theme values live in `web/src/app.css`.

Theme selection logic lives in `web/src/shared/theme/theme.ts`.

Reusable visual components live in `web/src/shared/ui/`.

When the visual language changes, update this guide and the theme source together.
