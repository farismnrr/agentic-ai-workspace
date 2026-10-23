# Design: Coding Solo and Orchestrator

## Direction

Follow the existing `masihawam-light` workspace. Keep the interface compact,
quiet, and familiar: flat rows, hairline borders, semantic status colors, and
olive only for the selected mode or primary action. Do not add a new palette,
decorative art, or another page shell.

Use plain language. **Coding — langsung** means one AI works directly on the
task. **Orchestrator — melalui tim** means a lead coordinates teammates. The
selected mode belongs to that task and remains visible on its detail page.

The existing Composer, Settings, and task header establish the layout. The
research reference for the Composer choice is the native daisyUI select with
`FormField`; for Settings, reuse the current Space selector, tool rows, and
`SettingsSection` patterns. This extends those screens without changing the
workspace shell.

## Tokens and type

- Theme: the existing `masihawam-light` in `web/src/app.css`.
- Primary selection and action: `primary`; informational notices: `info`;
  unavailable team and save failures: `error`; neutral surfaces: `base-100`,
  `base-200`, and `base-300`.
- Borders and secondary text use the existing `--brand-border`,
  `--brand-border-strong`, and `--brand-text-secondary` variables.
- Workspace text remains 14px Inter with the existing 22px line height.
  Headings in the signed-in workspace stay sans-serif.
- Use existing daisyUI `select`, `badge`, `alert`, `toggle`, and button classes.
  Do not add feature CSS or color values for these changes.

## Layout

- **Container:** inherit the current workspace content width. Composer follows
  Home's content column; task detail follows the existing split task screen;
  Settings follows its existing content region. Preserve 16px phone and 24px
  desktop page gutters where those screens already use them.
- **Grid:** Composer is one column at 390, 768, and 1280px. Settings is one
  column at 390px; at 768 and 1280px keep the existing 11rem Space list beside
  a flexible detail panel. Task detail keeps its current one-column phone view
  and conversation plus 26rem work panel at wide desktop.
- **Gutters:** 16px between related controls, 24px between groups, and 32px
  between major Settings sections. Keep the existing compact 8px row rhythm.
- **Composer, in order:** prompt field, then its existing controls row. Add the
  mode `FormField` to the controls area before Send, with a visible label and a
  short hint. At phone width, controls may wrap; mode and Send remain reachable
  without horizontal scrolling. Keep the prompt and send action visible in the
  initial viewport.
- **Task detail:** show the saved mode badge beside the title and status badge
  in the existing header flow. It must remain visible after reload. On phones,
  allow wrapping beneath the title rather than clipping or pushing controls
  off-screen.
- **Settings:** retain the current Space list and detail panel. Coding and
  Orchestrator are separate selectable entries. The detail panel shows its
  own address, tool permissions, and supported approval setting. Omit
  **Skip approvals** entirely for Orchestrator (`supportsApprovals: false`);
  show it for Coding (`true`). Do not add a third layer of cards.
- **Alignment:** labels align with their controls; Space names align with the
  existing selector; status badges sit in the task title metadata row. Keep
  text left-aligned. No new content is intentionally centered.
- **Hierarchy:** one main action per area. Composer's Send stays primary;
  mode is a compact, clearly labelled choice. Body instructions should wrap at
  45–75 characters where possible. Give the mode select a 44px minimum height.
  Settings switch rows must provide a 44px touch target. Keep visible focus.

## Screens, copy, and states

### Composer

Use a native `<select class="select">` inside the shared `FormField`, following
the existing Database Settings field pattern. Initial value is Coding. The
choices are:

1. **Coding — langsung**
2. **Orchestrator — melalui tim**

Hint: “Coding works directly on this task. Orchestrator coordinates a team.”
Send an explicit `team` value for either selection. While sending, keep the
choice readable and disable it with the rest of the in-flight composer.

If team setup is unavailable, show an inline error near the mode choice or
composer notices: “Orchestrator is unavailable because the team is not ready.
Ask your workspace owner to check the team setup, then try again.” Identify the
problem in text and keep the submitted prompt available for recovery. Do not
silently switch the choice to Coding.

### Task detail

Render the stored value as a compact text badge: **Coding** or
**Orchestrator · team**. Use the saved task field, not a current default.
Existing tasks keep the mode already stored on them. Failure states keep the
same badge and show the recovery guidance in the existing task status/thread
area; do not add an unrequested fallback action.

### Settings → Spaces & tools

Keep the current loading spinner, load error, and empty state. Add Orchestrator
to the existing Space selector. Selecting it shows its own connector address
and tool switches. Tool switch saves keep the existing optimistic behavior and
rollback/error message. Explain that these switches control external MCP
access; tasks started in the app still follow Coding permissions.

The Orchestrator detail does not show **Skip approvals**. The approval UI for
task work remains unchanged; questions and approvals are handled in the app.
Keyboard users can move through the Space selector, address, and permission
switches in document order. Selection and enabled state must be conveyed by
text or native control state, not color alone.

## Accessibility and responsive behavior

- Keep the visible label and hint associated with the mode select via
  `FormField`'s `for`, `aria-describedby`, and `aria-invalid` contract.
- A native select supports arrow-key selection; Tab reaches it and Send in
  order. Do not replace it with a custom menu.
- Preserve the global `:focus-visible` treatment. Ensure wrapping controls do
  not overlap or become clipped at 320px, 390px, and tablet widths.
- Error and save-failure states use readable text and the existing alert/error
  semantics. Announce asynchronous errors with the existing notice pattern.
- Honor the existing reduced-motion rules; no new motion is needed.

## Reuse map for the Developer

- Composer: `home/Composer.svelte`, `components/ComposerBox.svelte`,
  `components/ComposerControls.svelte`, and `components/FormField.svelte`.
  The shared composer accepts a `tools` snippet for additional controls.
- Settings: `settings/SpacesTab.svelte`, `SpaceTools.svelte`,
  `ConnectorAddress.svelte`, `SkipApprovals.svelte`, and
  `settings/parts/SettingsSection.svelte` / `SettingRow.svelte`.
- Detail badge: extend the existing mode badge in
  `task/TaskHeader.svelte`; preserve the existing task status badge.
- Keep all visual changes within these existing screens. No new shared UI
  abstraction is needed for this single feature.

## Assets needed

None. Existing controls, semantic badges, and status patterns are sufficient.

## Styling foundation

No base stylesheet change is needed. `web/src/app.css` already supplies the
required theme, compact workspace typography, semantic roles, focus outline,
and reduced-motion behavior. Reuse those tokens and daisyUI classes; do not
introduce feature-scoped CSS.

## Template

No `.template` file is present. This is an adaptation of the existing signed-in
workspace UI, not a new landing page or application shell. Keep the current
Composer and Settings structure; no landing, login, or dashboard rebuild is
part of this change.

## Review status

Design pass only. No running app or screenshot was available, so visual
inspection of desktop and mobile screens remains for the styling pass after
the Developer's candidate is running.
